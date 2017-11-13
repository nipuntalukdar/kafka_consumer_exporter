package main

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/Shopify/sarama"
)

const (
	minMetadataFetchInterVal int64 = 100
	metaDataRefreshTime            = 5 * time.Second
	topicOffsetRefreshTime         = 1 * time.Second
	groupRefreshTime               = 2 * time.Second
)

type availableOffset struct {
	err    bool
	offset int64
}

type kbroker struct {
	id   int32
	addr string
	br   *sarama.Broker
}

type part_details struct {
	part      int32
	leader_id int32
}

type metaData struct {
	topics_in_error        []string
	topic_part_notok       map[string][]int32
	topic_part_ok          map[string][]*part_details
	topic_part_offsets     map[string]map[int32]*availableOffset
	last_part_offset_fetch time.Time
	group_offsets          map[string]map[string]map[int32]int64
	group_members          map[string]int
	last_group_fetch       time.Time
	brokers                []*kbroker
	last_fetch             time.Time

	fetch_success bool
}

type kafka_client struct {
	broker_list     []string
	required_topics []string
	required_groups map[string][]string
	client          sarama.Client
	conf            *sarama.Config
	mdata           *metaData
	lock            *sync.Mutex
}

func (mdata *metaData) _errorTopicCount() int {
	return len(mdata.topics_in_error)
}

func (mdata *metaData) _errorPartitionCount() int {
	ret := 0
	for _, v := range mdata.topic_part_notok {
		ret += len(v)
	}
	return ret
}

func (mdata *metaData) _goodPartitionCount() int {
	ret := 0
	for _, v := range mdata.topic_part_ok {
		ret += len(v)
	}
	return ret
}

func newKafkaClient(broker_list, topic_list []string, group_map map[string][]string) *kafka_client {
	conf := sarama.NewConfig()
	conf.Version = sarama.V0_10_1_0
	conf.Metadata.RefreshFrequency = 2 * time.Minute
	var (
		client sarama.Client
		err    error
	)
	for i := 0; i < 10; i++ {
		client, err = sarama.NewClient(broker_list, conf)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
		log.Printf("Retrying kafka client initialiazation")
	}
	check_error_exit("Kafka client initialiazation problem", err)
	mdata := &metaData{topic_part_notok: make(map[string][]int32),
		topic_part_ok: make(map[string][]*part_details), topics_in_error: make([]string, 0)}
	// Monitor the topics in group list as well
	for _, topics := range group_map {
		topic_list = append(topic_list, topics...)
	}
	topic_list = unique_list(topic_list)
	return &kafka_client{broker_list: broker_list, required_topics: topic_list,
		client: client, required_groups: group_map, conf: conf, mdata: mdata, lock: &sync.Mutex{}}
}

func (cl *kafka_client) getRequiredTopicCount() int {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	return len(cl.required_topics)
}

func (cl *kafka_client) getBrokers() (ret []*kbroker) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	return cl._getBrokers()
}

func (cl *kafka_client) _getBrokers() (ret []*kbroker) {
	cl._getMetadata()
	if cl.mdata == nil || len(cl.mdata.brokers) == 0 {
		return
	}
	ret = make([]*kbroker, len(cl.mdata.brokers))
	copy(ret, cl.mdata.brokers)
	return
}

func (cl *kafka_client) getGroupOffsets() map[string]map[string]map[int32]int64 {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl._getGroupOffsets()
	ret := make(map[string]map[string]map[int32]int64)
	if cl.mdata == nil {
		return ret
	}
	for group, topics := range cl.mdata.group_offsets {
		ret[group] = make(map[string]map[int32]int64)
		for topic, parts := range topics {
			ret[group][topic] = make(map[int32]int64)
			for part, offset := range parts {
				ret[group][topic][part] = offset
			}
		}
	}
	return ret
}

func (cl *kafka_client) getGroupMemberCount() (grpmembers map[string]int) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl._getGroupOffsets()
	grpmembers = make(map[string]int)
	if cl.mdata == nil {
		return
	}
	for group, count := range cl.mdata.group_members {
		grpmembers[group] = count
	}
	return
}

func (cl *kafka_client) _getGroupOffsets() {
	cl._getMetadata()
	if cl.mdata == nil {
		return
	}
	if time.Now().Sub(cl.mdata.last_group_fetch) < groupRefreshTime {
		return
	}
	group_offsets := make(map[string]map[string]map[int32]int64)
	cl.mdata.group_offsets = group_offsets
	cl.mdata.group_members = make(map[string]int)
	addelem := 0
	var offr *sarama.OffsetFetchRequest
	for group, topics := range cl.required_groups {
		offr = &sarama.OffsetFetchRequest{ConsumerGroup: group, Version: 1}
		for _, topic := range topics {
			okay_partitions, ok := cl.mdata.topic_part_ok[topic]
			if ok {
				for _, okay_partition := range okay_partitions {
					offr.AddPartition(topic, okay_partition.part)
				}
				addelem++
			}
		}
		if addelem > 0 {
			addelem = 0
			coordinator_broker, err := cl.client.Coordinator(group)
			if err != nil {
				log.Printf("Offset fetch, error: %v", err)
				continue
			}
			descGrs := &sarama.DescribeGroupsRequest{Groups: []string{group}}
			resp, err := coordinator_broker.DescribeGroups(descGrs)
			if err == nil && resp != nil {
				cl.mdata.group_members[group] = len(resp.Groups[0].Members)
			}
			offsetdetails, err := coordinator_broker.FetchOffset(offr)
			if err == nil {
				_populateGroupOffsets(group, offsetdetails, cl.mdata.group_offsets)
			}
		}
	}
	cl.mdata.last_group_fetch = time.Now()
}

func _populateGroupOffsets(group string, offsetdetails *sarama.OffsetFetchResponse,
	group_offsets map[string]map[string]map[int32]int64) {
	if len(offsetdetails.Blocks) == 0 {
		return
	}
	group_details := group_offsets[group]
	if group_details == nil {
		group_details = make(map[string]map[int32]int64)
		group_offsets[group] = group_details
	}
	for topic, partresps := range offsetdetails.Blocks {
		group_details[topic] = make(map[int32]int64)
		for part, offr := range partresps {
			if offr.Err == sarama.ErrNoError {
				group_details[topic][part] = offr.Offset
			} else {
				group_details[topic][part] = -1
			}
		}
	}
}

func (cl *kafka_client) getTopicPartStats() (good, bad map[string]int, err error) {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl._getMetadata()
	if cl.mdata == nil || !cl.mdata.fetch_success {
		log.Printf("Some problem ")
		err = errors.New("Metadata fecth problem")
		return
	}
	good = copyMapCountPartOk(cl.mdata.topic_part_ok)
	bad = copyMapCount(cl.mdata.topic_part_notok)
	return
}

func (cl *kafka_client) getApiVersions() {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl._getMetadata()
	brokers := cl.client.Brokers()
	if len(brokers) == 0 {
		log.Printf("Empty broker list")
		return
	}

	br := brokers[rand.Intn(len(brokers))]
	br.Open(cl.conf)
	defer br.Close()
	apir, err := br.ApiVersions(&sarama.ApiVersionsRequest{})
	if err != nil || apir.Err != sarama.ErrNoError {
		log.Printf("Problem in fetching API version")
		return
	}
	for _, a_version := range apir.ApiVersions {
		log.Printf("API Version %v", a_version)
	}
}

func (cl *kafka_client) activeBrokerCount() int {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	cl._getMetadata()
	return len(cl.client.Brokers())
}

func (cl *kafka_client) getTopicOffsets() (ret map[string]map[int32]int64) {
	ret = make(map[string]map[int32]int64)
	cl.lock.Lock()
	defer cl.lock.Unlock()
	mdata := cl._getMetadata()
	if mdata == nil {
		log.Printf("Metadata fetch errror")
		return
	}
	if time.Now().Sub(cl.mdata.last_part_offset_fetch) > topicOffsetRefreshTime {
		cl._getTopicOffsets()
	}
	for topic, partoffset := range cl.mdata.topic_part_offsets {
		ret[topic] = make(map[int32]int64)
		for part, offset := range partoffset {
			if offset.err {
				continue
			}
			out, ok := ret[topic]
			if !ok {
				out = make(map[int32]int64)
				ret[topic] = out
			}
			out[part] = offset.offset
		}
	}
	return
}

func (cl *kafka_client) _getTopicOffsets() {
	brs := cl.client.Brokers()
	if len(brs) == 0 {
		log.Printf("No available brokers to fetch metdata")
		return
	}
	id_to_br := make(map[int32]*sarama.Broker)
	for _, br := range brs {
		id_to_br[br.ID()] = br
	}
	offrs := make(map[int32]*sarama.OffsetRequest)
	for topic, partitions := range cl.mdata.topic_part_ok {
		for _, partition := range partitions {
			offr := offrs[partition.leader_id]
			if offr == nil {
				offr = &sarama.OffsetRequest{Version: int16(1)}
				offrs[partition.leader_id] = offr
			}
			offr.AddBlock(topic, partition.part, -1, 1)
		}
	}
	var (
		err       error
		offr_resp *sarama.OffsetResponse
		offset    int64
	)

	off_resp_all := make([]*sarama.OffsetResponse, 0)
	for brid, offr := range offrs {
		br, ok := id_to_br[brid]
		if !ok {
			log.Printf("Broker for id % not found to fetch metadata", brid)
			continue
		}
		err = br.Open(cl.conf)
		if err != nil && err != sarama.ErrAlreadyConnected {
			continue
		}
		offr_resp, err = br.GetAvailableOffsets(offr)
		br.Close()
		if err != nil {
			continue
		}
		off_resp_all = append(off_resp_all, offr_resp)
	}
	if len(off_resp_all) == 0 {
		log.Printf("No topic offset fetch happened")
		return
	}
	cl.mdata.topic_part_offsets = make(map[string]map[int32]*availableOffset)
	for _, offr_resp = range off_resp_all {
		version := offr_resp.Version
		for topic, block := range offr_resp.Blocks {
			_, ok := cl.mdata.topic_part_offsets[topic]
			if !ok {
				cl.mdata.topic_part_offsets[topic] = make(map[int32]*availableOffset)
			}
			for partition, ofr_block := range block {
				if version == 0 {
					offset = ofr_block.Offsets[0]
				} else {
					offset = ofr_block.Offset
				}
				cl.mdata.topic_part_offsets[topic][partition] = &availableOffset{err: ofr_block.Err != sarama.ErrNoError, offset: offset}
			}
		}
	}
	cl.mdata.last_part_offset_fetch = time.Now()
}

func (cl *kafka_client) getMetadata() *metaData {
	cl.lock.Lock()
	defer cl.lock.Unlock()
	return cl._getMetadata()
}

func _getBrokersFromMetadata(mrd *sarama.MetadataResponse) (brokers []*kbroker) {
	brokers = make([]*kbroker, 0)
	for _, br := range mrd.Brokers {
		brokers = append(brokers, &kbroker{id: br.ID(), addr: br.Addr(), br: br})
	}
	return
}

func (cl *kafka_client) _getMetadata() *metaData {
	if cl.mdata != nil && time.Now().Sub(cl.mdata.last_fetch) < metaDataRefreshTime {
		return cl.mdata
	}
	cl.mdata = nil
	num_brokers := len(cl.client.Brokers())
	var br *sarama.Broker
	attempts := 0
	refresh_attempts := 0
	var err error = nil
	for {
		for {
			br = cl.client.Brokers()[rand.Intn(num_brokers)]
			err := br.Open(cl.conf)
			if err == nil || err == sarama.ErrAlreadyConnected {
				break
			}
			attempts++
			if attempts == 10 {
				log.Printf("Error: Couldn't connect to any broker")
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if err == nil || err == sarama.ErrAlreadyConnected {
			break
		}
		if refresh_attempts < 6 {
			time.Sleep(200 * time.Millisecond)
			cl.client.RefreshMetadata(cl.required_topics...)
			refresh_attempts++
			attempts = 0
		} else {
			log.Printf("Error: Broker connection problem")
			return nil
		}
	}

	defer br.Close()
	mrd, err := br.GetMetadata(&sarama.MetadataRequest{Topics: cl.required_topics})
	if err != nil {
		log.Printf("Metadata fetch error :%v", err)
		return nil
	}
	cl.mdata = &metaData{topic_part_notok: make(map[string][]int32),
		topic_part_ok: make(map[string][]*part_details), topics_in_error: make([]string, 0)}
	cl.mdata.last_fetch = time.Now()
	cl.mdata.brokers = _getBrokersFromMetadata(mrd)
	for _, tmd := range mrd.Topics {
		if tmd.Err != sarama.ErrNoError {
			cl.mdata.topics_in_error = append(cl.mdata.topics_in_error, tmd.Name)
			continue
		}
		for _, pmedata := range tmd.Partitions {
			if pmedata.Err != sarama.ErrNoError {
				_, ok := cl.mdata.topic_part_notok[tmd.Name]
				if !ok {
					cl.mdata.topic_part_notok[tmd.Name] = []int32{pmedata.ID}
				} else {
					cl.mdata.topic_part_notok[tmd.Name] = append(cl.mdata.topic_part_notok[tmd.Name],
						pmedata.ID)
				}
			} else {
				_, ok := cl.mdata.topic_part_ok[tmd.Name]
				if !ok {
					cl.mdata.topic_part_ok[tmd.Name] = []*part_details{&part_details{pmedata.ID, pmedata.Leader}}
				} else {
					cl.mdata.topic_part_ok[tmd.Name] = append(cl.mdata.topic_part_ok[tmd.Name],
						&part_details{pmedata.ID, pmedata.Leader})
				}
			}
		}
	}
	cl.mdata.fetch_success = true
	return cl.mdata
}
