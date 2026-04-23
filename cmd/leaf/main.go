package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/eco-digit/leaf/internal/cache"
	"github.com/eco-digit/leaf/internal/config"
	"github.com/eco-digit/leaf/internal/embodied"
	"github.com/eco-digit/leaf/internal/infrastructure"
	"github.com/eco-digit/leaf/internal/intensity"
	"github.com/eco-digit/leaf/internal/model"
	"github.com/eco-digit/leaf/internal/orchestrator"
	"github.com/eco-digit/leaf/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	collectOnce := flag.Bool("collect-once", false, "Run one calculation cycle, print results, and exit")
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

	embodiedRS, err := embodied.Calculate(infra, time.Now().Truncate(time.Hour))
	if err != nil {
		log.Fatalf("calculate embodied: %v", err)
	}
	if err := embodied.Validate(embodiedRS); err != nil {
		log.Fatalf("validate embodied: %v", err)
	}
	c.Update(embodiedRS)
	log.Printf("seeded cache with %d embodied impact records", len(embodiedRS))

	interval, err := parseInterval(cfg.Orchestrator.ReportingInterval)
	if err != nil {
		log.Fatalf("orchestrator interval: %v", err)
	}

	ttl, err := parseInterval(cfg.Intensity.TTL)
	if err != nil {
		log.Fatalf("intensity ttl: %v", err)
	}
	if ttl == 0 {
		ttl = interval
	}

	orch, err := orchestrator.New(
		cfg.Prometheus.URL, cfg.Prometheus.Username, cfg.Prometheus.Password,
		intensity.Config{TTL: ttl, Zone: cfg.Intensity.ElectricityMaps.Zone, Country: cfg.Intensity.MixData.Country},
		cfg.Intensity.ElectricityMaps.APIKey, cfg.Intensity.ElectricityMaps.BaseURL,
		cfg.Intensity.MixData.Path,
		interval,
		infra,
		embodiedRS,
		c,
	)
	if err != nil {
		log.Fatalf("orchestrator: %v", err)
	}

	if *collectOnce {
		result, err := orch.RunCycle()
		if err != nil {
			log.Fatalf("collect-once: %v", err)
		}
		printCycleResult(result)
		return
	}

	addr := cfg.Server.Addr
	if addr == "" {
		addr = ":9010"
	}
	log.Printf("starting Leaf on %s", addr)
	srv := server.New(c, addr)

	go orch.Start(context.Background())

	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func parseInterval(s string) (time.Duration, error) {
	if s == "" {
		return time.Hour, nil
	}
	return time.ParseDuration(s)
}

// printCycleResult prints a human-readable summary of one calculation cycle.
func printCycleResult(result orchestrator.CycleResult) {
	fmt.Printf("\n ### Intensity factors ###\n")
	fmt.Printf("  GWP  %12g %-25s (source: %s)\n", result.Factors.GWP.Value, result.Factors.GWP.Unit, result.Factors.GWP.Source)
	fmt.Printf("  ADP  %12g %-25s (source: %s)\n", result.Factors.ADP.Value, result.Factors.ADP.Unit, result.Factors.ADP.Source)
	fmt.Printf("  CED  %12g %-25s (source: %s)\n", result.Factors.CED.Value, result.Factors.CED.Unit, result.Factors.CED.Source)

	fmt.Printf("\n## Provider impact results ##\n")
	printResultSet(result.RS)

	if len(result.Warnings) > 0 {
		fmt.Printf("\n## Warnings (%d) ##\n", len(result.Warnings))
		for _, w := range result.Warnings {
			fmt.Printf("  ! %s\n", w)
		}
	}
	fmt.Println()
}

func printResultSet(rs model.ResultSet) {
	type row struct {
		key   string
		label string
		value float64
		unit  string
	}

	var rows []row
	for _, r := range rs {
		var label string
		switch r.Subject {
		case model.SubjectDevice:
			label = fmt.Sprintf("device  %-20s %-10s %-12s", r.Device, r.Component, r.Category)
		case model.SubjectProvider:
			label = fmt.Sprintf("provider%-20s %-10s %-12s", "", r.Component, r.Category)
		default:
			continue
		}
		key := fmt.Sprintf("%s/%s/%s/%s/%s", r.Subject, r.ImpactPhase, r.Component, r.Category, r.Device)
		rows = append(rows, row{key: key, label: label + " [" + string(r.ImpactPhase) + "]", value: r.Value, unit: r.Unit})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].key < rows[j].key })

	for _, r := range rows {
		fmt.Printf("  %s  %12g %s\n", r.label, r.value, r.unit)
	}
}
