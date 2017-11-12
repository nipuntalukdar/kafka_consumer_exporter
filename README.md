*kafka consumer stats exporter for Prometheus*
==============================================

Kafka consumer exporter 
-----------------------

This exporter can be used to export offset details of topics, get consumer offsets, get consumer
counts in a group, kafka cluster brokers information, consumer lags, topic and partition details etc.

For consumer lags and offsets  monitoring, it works with the new high level consumers
only right now. 
The old high level consumer where offsets are stored in Zookeeper and where consumer-coordinations happen
through Zookeeper is not tested.

Installation
------------
You need to have golang installed to install the tool. I haven't uploaded any prebuilt binary for
any platoform. I will be adding a dockerfile to build a Docker image shortly. 

Run the below command:

**$ go get github.com/nipuntalukdar/kafka_consumer_exporter**

**$ go install github.com/nipuntalukdar/kafka_consumer_exporter**


Running the exporter
--------------------

An example:
**$ kafka_consumer_exporter -group agroup:topica,topicb -group anothergroup:topicx,topicy,topicz -topics mytopic,yourtopic **

Detailed usage shown below:
```bash
$ kafka_consumer_exporter -h
Usage of ./kafka_consumer_exporter:
  -group value
    	consumer-group and topics in the form of group1:topic1,topic2,topic3 etc
  -kafka_brokers string
    	Comma-separated list of Kafka brokers (default "127.0.0.1:9092")
  -listen_address string
    	http port where metrics are published (default ":10001")
  -metrics_url string
    	URL where mettrics is accessible (default "/metrics")
  -namespace string
    	Namespace for  metrics (default "kafka")
  -topics string
    	Comma separated list of kafka topics
  -with-go-metrics
    	Should we import go runtime and http handler metrics
```
