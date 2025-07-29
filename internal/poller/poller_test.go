package poller

import (
	"testing"
	"time"

	"gorssag/internal/aggregator"
	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/storage"
)

func TestPoller_New(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{"AI", "blockchain"},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	storageManager, _ := storage.NewStorage("./testdata")
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute)

	if p == nil {
		t.Error("Expected poller to be created, got nil")
	}
}

func TestPoller_IsPolling(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	storageManager, _ := storage.NewStorage("./testdata")
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute)

	// Initially should not be polling
	if p.IsPolling() {
		t.Error("Expected poller to not be polling initially")
	}

	// Start polling
	p.Start()
	if !p.IsPolling() {
		t.Error("Expected poller to be polling after start")
	}

	// Stop polling
	p.Stop()
	if p.IsPolling() {
		t.Error("Expected poller to not be polling after stop")
	}
}

func TestPoller_GetLastPolledTime(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	storageManager, _ := storage.NewStorage("./testdata")
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute)

	// Initially should be empty
	lastPolled := p.GetLastPolledTime()
	if len(lastPolled) != 0 {
		t.Errorf("Expected empty last polled times, got %d", len(lastPolled))
	}

	// Force poll a topic
	err := p.ForcePoll("tech")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should now have a last polled time
	lastPolled = p.GetLastPolledTime()
	if len(lastPolled) == 0 {
		t.Error("Expected non-zero last polled times after force poll")
	}

	// Check if the time is recent
	if lastPolled["tech"].IsZero() {
		t.Error("Expected last polled time to be recent")
	}
}

func TestPoller_ForcePoll_InvalidTopic(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	storageManager, _ := storage.NewStorage("./testdata")
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute)

	// Try to force poll an invalid topic
	err := p.ForcePoll("invalid-topic")
	if err == nil {
		t.Error("Expected error for invalid topic, got nil")
	}
}
