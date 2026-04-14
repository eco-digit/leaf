package main

import (
	"flag"
	"log"
	"time"

	"github.com/OSBA-eco-digit/leaf/internal/cache"
	"github.com/OSBA-eco-digit/leaf/internal/config"
	"github.com/OSBA-eco-digit/leaf/internal/embodied"
	"github.com/OSBA-eco-digit/leaf/internal/infrastructure"
	"github.com/OSBA-eco-digit/leaf/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
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
