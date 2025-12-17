package exporter

import (
	"github.com/OSBA-eco-digit/leaf/internal/promclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"log"
	"sync"
)

type LeafExporter struct {
	client  *promclient.Client
	metrics []*prometheus.Desc
}

func NewLeafExporter(client *promclient.Client) *LeafExporter {
	return &LeafExporter{
		client: client,
		metrics: []*prometheus.Desc{
			prometheus.NewDesc("leaf_kepler_node_power_watts", "Kepler node power metric", nil, nil),
		},
	}
}

func (e *LeafExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.metrics {
		ch <- m
	}
}
func (e *LeafExporter) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		val, err := e.client.QueryMetric(`kepler_node_cpu_watts{instance="compute04"}`)
		if err != nil {
			log.Printf("Query failed for kepler_node_cpu_watts: %v", err)
			return
		}

		switch v := val.(type) {
		case *model.Scalar:
			log.Printf("Query successful: kepler_node_cpu_watts = %v", v.Value)
			ch <- prometheus.MustNewConstMetric(e.metrics[0], prometheus.GaugeValue, float64(v.Value))
		case model.Vector:
			log.Printf("Query successful: kepler_node_cpu_watts returned %d samples", len(v))
			for _, sample := range v {
				log.Printf("  Sample: %v = %v", sample.Metric, sample.Value)
				ch <- prometheus.MustNewConstMetric(e.metrics[0], prometheus.GaugeValue, float64(sample.Value))
			}
		default:
			log.Printf("Unhandled metric type: %T", v)
		}
	}()
	wg.Wait()
}
