package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listen_address = flag.String("listen_address", ":10001", "http port where metrics are published")
	metrics_url    = flag.String("metrics_url", "/metrics", "URL where mettrics is accessible")
	kafka_brokers  = flag.String("kafka_brokers", "127.0.0.1:9092", "Comma-separated list of Kafka brokers")
	namespace      = flag.String("namespace", "kafka", "Namespace for  metrics")
	topics         = flag.String("topics", "", "Comma separated list of kafka topics")
	gometrics      = flag.Bool("with-go-metrics", false, "Should we import go runtime and http handler metrics")
	groups         arrayFlags
)

func main() {
	var group_map map[string][]string
	flag.Var(&groups, "group", "consumer-group and topics in the form of group1:topic1,topic2,topic3 etc")
	flag.Parse()
	*kafka_brokers = check_input_str(*kafka_brokers, "Empty broker list")
	*topics = check_input_str(*topics, "Empty topics list")
	broker_list := get_list_from_string(*kafka_brokers, ",", "Empty broker list")
	topic_list := get_list_from_string(*topics, ",", "Empty string list")
	topic_list = unique_list(topic_list)
	if groups.GetElCount() > 0 {
		group_map = get_group_map(&groups, "Incorrrect values")
	}
	*metrics_url = check_input_str(*metrics_url, "Empty metrics url")
	*namespace = check_input_str(*namespace, "Empty namspace string")
	client := newKafkaClient(broker_list, topic_list, group_map)
	if client.getRequiredTopicCount() < 1 {
		log.Fatal("Topic count for stats is 0")
	}
	log.Printf("Starting kafka exporter")

	if !*gometrics {
		registry := prometheus.NewRegistry()
		if err := registry.Register(newExporter(client, *namespace)); err != nil {
			log.Fatalf("Registration error: %v", err)
		}
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		http.Handle(*metrics_url, handler)
	} else {
		prometheus.DefaultRegisterer.MustRegister(newExporter(client, *namespace))
		http.Handle(*metrics_url, promhttp.Handler())
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
				<html>
					<head><title>Kafka Consumer Exporter </title></head>
					<body>
							<h1>Kafka exporter </h1>
							<p><a href='` + *metrics_url + `'>Metrics</a></p>
					</body>
				</html>`))
	})
	log.Printf("Listening on %s, url: %s", *listen_address, *metrics_url)
	if err := http.ListenAndServe(*listen_address, nil); err != nil {
		log.Fatalf("Error in starting http listner: %v", err)
	}
}
