package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/eco-digit/leaf/internal/cache"
	"github.com/eco-digit/leaf/internal/collector"
	"github.com/eco-digit/leaf/internal/config"
	"github.com/eco-digit/leaf/internal/embodied"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/promclient"
	"github.com/eco-digit/leaf/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	// DEV: Needed for testing
	collectOnce := flag.Bool("collect-once", false, "Run one collection cycle, print results, and exit")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	infra, err := infrastructure.Load(cfg.Infrastructure.InfraPath, cfg.Infrastructure.ProfilePath)
	if err != nil {
		log.Fatalf("load infrastructure: %v", err)
	}

	c := cache.New()

	// Pass cache with static embodied metrics computed at startupl.
	rs, err := embodied.Calculate(infra, time.Now().Truncate(time.Hour))
	if err != nil {
		log.Fatalf("calculate embodied: %v", err)
	}
	if err := embodied.Validate(rs); err != nil {
		log.Fatalf("validate embodied: %v", err)
	}
	c.Update(rs)
	log.Printf("Seeded cache with %d embodied impact records", len(rs))

	// DEV: Fill hte cache with prom metrics once
	if *collectOnce {
		runCollectOnce(cfg, infra)
		return
	}

	addr := cfg.Server.Addr
	if addr == "" {
		addr = ":9010"
	}

	log.Printf("Starting Leaf on %s", addr)
	srv := server.New(c, addr)
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// runCollectOnce performs a single collection cycle, prints a summary of what was collected
func runCollectOnce(cfg *config.Config, infra *infrastructure.Infrastructure) {
	client, err := promclient.NewClient(cfg.Prometheus.URL, cfg.Prometheus.Username, cfg.Prometheus.Password)
	if err != nil {
		log.Fatalf("collect-once: %v", err)
	}

	window := cfg.Orchestrator.ReportingInterval
	if window == "" {
		window = "1h"
	}

	raw, err := collector.Collect(client, infra, window, time.Now())
	if err != nil {
		log.Fatalf("collect-once: %v", err)
	}

	fmt.Printf("\n=== Collection results (window=%s) ===\n\n", window)

	fmt.Printf("Devices (%d registered):\n", len(infra.Devices))
	for _, dev := range infra.Devices {
		d := raw.Devices[dev.ID]
		if len(d.Metrics) == 0 && len(d.VMMetrics) == 0 {
			fmt.Printf("  %-20s  [no data]\n", dev.ID)
			continue
		}
		fmt.Printf("  %-20s\n", dev.ID)
		for src, val := range d.Metrics {
			fmt.Printf("    %-30s = %.4f\n", src, val)
		}
		for src, vms := range d.VMMetrics {
			fmt.Printf("    %-30s = %d VMs\n", src, len(vms))
			for vmID, val := range vms {
				fmt.Printf("      %-28s = %.4f\n", vmID, val)
			}
		}
	}

	if len(raw.Racks) > 0 {
		fmt.Printf("\nRack/infrastructure metrics (%d instances):\n", len(raw.Racks))
		for instance, rack := range raw.Racks {
			fmt.Printf("  %s\n", instance)
			for src, val := range rack.Metrics {
				fmt.Printf("    %-30s = %.4f\n", src, val)
			}
		}
	}

	if len(raw.VMInfos) > 0 {
		fmt.Printf("\nVM metadata (%d VMs from libvirt):\n", len(raw.VMInfos))
		for _, vm := range raw.VMInfos {
			fmt.Printf("  %-36s  project=%s (%s)  flavor=%s  vcpus=%.0f  mem=%.0f GB\n",
				vm.VMID, vm.ProjectID, vm.ProjectName, vm.FlavorName, vm.VCPUs, vm.MemoryGB)
		}
	}

	if len(raw.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(raw.Warnings))
		for _, w := range raw.Warnings {
			fmt.Printf("  ! %s\n", w)
		}
	}

	fmt.Println()
}
