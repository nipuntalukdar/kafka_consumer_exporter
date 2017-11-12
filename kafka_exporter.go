package main

import (
	"fmt"
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

// Exporter collects kakfka topics and consumer metrics

type Exporter struct {
	client         *kafka_client
	namespace      string
	callCount      prometheus.Counter
	brokerCount    prometheus.Gauge
	consumerCount  *prometheus.GaugeVec
	topicPartCount *prometheus.GaugeVec
	brokers        *prometheus.GaugeVec
	highestOffset  *prometheus.GaugeVec
	consumerLag    *prometheus.GaugeVec
	consumeroffset *prometheus.GaugeVec
}

func newExporter(client *kafka_client, namespace string) *Exporter {
	exporter := &Exporter{client: client, namespace: namespace}
	exporter.callCount = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "exporter_called",
		Help:      "How many times exporter was called",
	})

	exporter.consumerCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "consumer_count",
		Help:      "Consumer count for a consumer group",
	}, []string{"group"})

	exporter.topicPartCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "topic_partition",
		Help:      "Topic partition count in good and bad state",
	}, []string{"topic", "state"})

	exporter.brokers = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "brokers",
		Help:      "broker id and address",
	}, []string{"id", "addr"})

	exporter.brokerCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "active_broker_count",
		Help:      "Number of brokers up",
	})

	exporter.highestOffset = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "highestoffset",
		Help:      "highest available offset of a partition",
	}, []string{"topic", "partition"})

	exporter.consumerLag = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "consumerlag",
		Help:      "current lag of consumer",
	}, []string{"group", "topic", "partition"})

	exporter.consumeroffset = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "consumeroffset",
		Help:      "current offset of consumer",
	}, []string{"group", "topic", "partition"})

	return exporter
}

// Describe describes all the metrics ever exported.  It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.callCount.Describe(ch)
	e.consumerCount.Describe(ch)
	e.brokerCount.Describe(ch)
	e.topicPartCount.Describe(ch)
	e.brokers.Describe(ch)
	e.highestOffset.Describe(ch)
	e.consumerLag.Describe(ch)
	e.consumeroffset.Describe(ch)
}

func (e *Exporter) collectConsumerCount(ch chan<- prometheus.Metric) {
	e.consumerCount.Reset()
	group_consumer_count := e.client.getGroupMemberCount()
	for group, member_count := range group_consumer_count {
		e.consumerCount.WithLabelValues(group).Set(float64(member_count))
	}
	e.consumerCount.Collect(ch)
}

func (e *Exporter) collectOffsetDetails(ch chan<- prometheus.Metric) {
	e.highestOffset.Reset()
	e.consumeroffset.Reset()
	hoffsets := e.client.getTopicOffsets()
	consoffsets := e.client.getGroupOffsets()
	if len(hoffsets) > 0 {
		for topic, partoffsets := range hoffsets {
			for part, offset := range partoffsets {
				e.highestOffset.WithLabelValues(topic, fmt.Sprintf("%d", part)).Set(float64(offset))
			}
		}
		e.highestOffset.Collect(ch)
	}
	if len(consoffsets) > 0 {
		for group, topics := range consoffsets {
			for topic, partoffs := range topics {
				for part, offset := range partoffs {
					e.consumeroffset.WithLabelValues(group, topic, fmt.Sprintf("%d", part)).Set(float64(offset))
					if hoffsets[topic] != nil {
						highest_off_set, ok := hoffsets[topic][part]
						if ok {
							e.consumerLag.WithLabelValues(group, topic,
								fmt.Sprintf("%d", part)).Set(float64(highest_off_set - offset))
						}
					}
				}
			}
		}
		e.consumerLag.Collect(ch)
		e.consumeroffset.Collect(ch)
	}
}

func (e *Exporter) collectTopicPart(ch chan<- prometheus.Metric) {
	e.topicPartCount.Reset()
	good, bad, err := e.client.getTopicPartStats()
	if err != nil {
		return
	}
	for topic, partcount := range good {
		e.topicPartCount.WithLabelValues(topic, "good").Set(float64(partcount))
	}
	for topic, partcount := range bad {
		e.topicPartCount.WithLabelValues(topic, "bad").Set(float64(partcount))
	}
	e.topicPartCount.Collect(ch)
}

func (e *Exporter) collectHighestOffsets(ch chan<- prometheus.Metric) {
	e.highestOffset.Reset()
	hoffsets := e.client.getTopicOffsets()
	if hoffsets == nil {
		return
	}
	for topic, partoffsets := range hoffsets {
		for part, offset := range partoffsets {
			e.highestOffset.WithLabelValues(topic, fmt.Sprintf("%d", part)).Set(float64(offset))
		}
	}
	e.highestOffset.Collect(ch)
}

func (e *Exporter) collecConsumerOffsets(ch chan<- prometheus.Metric) {
	e.consumeroffset.Reset()
	consoffsets := e.client.getGroupOffsets()
	for group, topics := range consoffsets {
		for topic, partoffs := range topics {
			for part, offset := range partoffs {
				e.consumeroffset.WithLabelValues(group, topic, fmt.Sprintf("%d", part)).Set(float64(offset))
			}
		}
	}
	e.consumeroffset.Collect(ch)
}

func (e *Exporter) collectBrokers(ch chan<- prometheus.Metric) {
	e.brokers.Reset()
	brokers := e.client.getBrokers()
	if brokers == nil {
		log.Printf("Broker detail collection failure")
		return
	}
	for _, broker := range brokers {
		e.brokers.WithLabelValues(fmt.Sprintf("%d", broker.id), broker.addr).Set(float64(1))
	}
	e.brokers.Collect(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.callCount.Inc()
	e.callCount.Collect(ch)
	v := e.client.activeBrokerCount()
	e.brokerCount.Set(float64(v))
	e.brokerCount.Collect(ch)
	e.collectTopicPart(ch)
	e.collectBrokers(ch)
	e.collectConsumerCount(ch)
	/*e.collectHighestOffsets(ch)
	e.collecConsumerOffsets(ch)*/
	e.collectOffsetDetails(ch)
	return
}
