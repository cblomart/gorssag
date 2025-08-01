// Copyright (c) 2024 cblomart
// Licensed under the MIT License

package main

import (
	"context"
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
	storageManager, err := storage.NewStorage(cfg.DataDir, cfg)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// Clean up old articles based on retention policy
	log.Printf("Cleaning up articles older than %v", cfg.ArticleRetention)
	if err := storageManager.CleanupOldArticles(cfg.ArticleRetention); err != nil {
		log.Printf("Warning: failed to cleanup old articles: %v", err)
	}

	// Compress old articles that are still uncompressed (run in background)
	log.Printf("Starting background compression of old articles for storage optimization")
	go func() {
		if err := storageManager.CompressOldArticles(); err != nil {
			log.Printf("Warning: failed to compress old articles: %v", err)
		} else {
			log.Printf("Background compression of old articles completed")
		}
	}()

	// Initialize RSS aggregator
	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)

	// Perform initial centralized feed polling to establish status
	log.Printf("Starting initial centralized feed polling...")
	err = agg.PollAllFeeds()
	if err != nil {
		log.Printf("Warning: some feeds failed during initial polling: %v", err)
	}
	log.Printf("Initial centralized feed polling completed")

	// Initialize background poller
	backgroundPoller := poller.New(agg, cacheManager, storageManager, cfg.Feeds, cfg.PollInterval, cfg.ArticleRetention, cfg)

	// Start background polling
	backgroundPoller.Start()

	// Initialize API server
	server := api.NewServer(agg, backgroundPoller, cfg)

	log.Printf("Starting RSS Aggregator server on port %d", cfg.Port)
	log.Printf("Data directory: %s", cfg.DataDir)
	log.Printf("Cache TTL: %v", cfg.CacheTTL)
	log.Printf("Article retention: %v", cfg.ArticleRetention)
	log.Printf("Background polling interval: %v", cfg.PollInterval)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start signal handler in goroutine
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping services...")
		backgroundPoller.Stop()
		cancel() // Cancel the context to stop the server
	}()

	// Start the server with context for graceful shutdown
	if err := server.StartWithContext(ctx); err != nil && err != context.Canceled {
		log.Fatal("Failed to start server:", err)
	}
}
