package main

import (
	"flag"
	"log"

	"github.com/OSBA-eco-digit/leaf/internal/cache"
	"github.com/OSBA-eco-digit/leaf/internal/config"
	"github.com/OSBA-eco-digit/leaf/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	c := cache.New()

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
