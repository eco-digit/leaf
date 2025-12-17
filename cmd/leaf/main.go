package main

import (
	"flag"
	"github.com/OSBA-eco-digit/leaf/internal/config"
	"github.com/OSBA-eco-digit/leaf/internal/exporter"
	"github.com/OSBA-eco-digit/leaf/internal/promclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client, err := promclient.NewClient(cfg.Prometheus.URL, cfg.Prometheus.Username, cfg.Prometheus.Password)
	if err != nil {
		log.Fatalf("Failed to create Prometheus client: %v", err)
	}

	leafExporter := exporter.NewLeafExporter(client)
	prometheus.MustRegister(leafExporter)

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Starting Leaf exporter on :9010 ...")
	log.Fatal(http.ListenAndServe("localhost:9010", nil))
}
