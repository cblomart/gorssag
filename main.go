// Copyright (c) 2024 cblomart
// Licensed under the MIT License

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gorssag/internal/aggregator"
	"gorssag/internal/api"
	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/poller"
	"gorssag/internal/storage"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize cache for hot data
	cacheManager := cache.NewManager(cfg.CacheTTL)

	// Initialize persistent storage
	storageManager, err := storage.NewStorage(cfg.DataDir)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// Initialize RSS aggregator
	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)

	// Initialize background poller
	backgroundPoller := poller.New(agg, cacheManager, storageManager, cfg.Feeds, cfg.PollInterval)

	// Start background polling
	backgroundPoller.Start()

	// Initialize API server
	server := api.NewServer(agg, backgroundPoller, cfg)

	log.Printf("Starting RSS Aggregator server on port %d", cfg.Port)
	log.Printf("Data directory: %s", cfg.DataDir)
	log.Printf("Cache TTL: %v", cfg.CacheTTL)
	log.Printf("Background polling interval: %v", cfg.PollInterval)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping services...")
		backgroundPoller.Stop()
	}()

	// Start the server
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
