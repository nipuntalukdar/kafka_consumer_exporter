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
**$ kafka_consumer_exporter -group agroup:topica,topicb -group anothergroup:topicx,topicy,topicz -topics mytopic,yourtopic**

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

How to Create Grafana Dashboard
-------------------------------
**Clone this repository.**
Go to directory **dashboardgen** under the repository directory.
Update the input.json with the topics and consumer groups you want to monitor. For example, if you want to monitor group "SomeGroup" which consumes from topis TopicA, TopicB, TopicC and also want to monitor another consumer group "SecretGroup" which consumes from topics SecretA, SecretB and also if you want to monitor extra topics ExtraTopicA, ExtraTopicB,  then you should update the **input.json** as shown below:

```bash
        {
          "consumergroups": {
            "SomeGroup": [
              "TopicA",
              "TopicB",
              "TopicC"
            ],
            "SecretGroup": [
              "SecretA",
              "SecretB"
            ]
          },
          "topics": [
            "TopicA",
            "TopicB",
            "TopicC",
            "SecretA",
            "SecretB",
            "ExtraTopicA",
            "ExtraTopicB"
          ]
        }

```


The dashboard needs an environment (so that we may cater for multiple Kafka cluster). A Prometheus target may look like as the one shown below:

```bash
         - job_name: 'KafkaTopicAndConsumer'
            honor_labels: true
            static_configs:
            - targets: ['localhost:10001']
              labels:
                env: 'myenv'
```

Now, you issue the below command (from dashboardgen directory). We are assuming environment label griven to Prometheus exporter is myenv.

**$ python kafkadashboard.py myenv   > kafkadashboard.json**

kafkadashboard.json may be imported to Grafana now !!!


Grafana Dashboard
-----------------
**A sample dashboard screenshot is given below:**

<img width="800" alt="screenshot1" src="https://user-images.githubusercontent.com/1930383/55284052-bf359f00-538c-11e9-91a7-61475b4205cd.PNG">\
<img width="800" alt="screenshot2" src="https://user-images.githubusercontent.com/1930383/55284053-c8bf0700-538c-11e9-8890-ca276dd4f2eb.PNG">\
<img width="800" alt="screenshot3" src="https://user-images.githubusercontent.com/1930383/55284058-d1afd880-538c-11e9-975a-87145b3a19bb.PNG">
