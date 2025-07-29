package poller

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorssag/internal/aggregator"
	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/storage"

	"github.com/mmcdole/gofeed"
)

type Poller struct {
	aggregator   *aggregator.Aggregator
	cacheManager *cache.Manager
	storage      storage.Storage
	feeds        map[string]config.TopicConfig
	parser       *gofeed.Parser
	pollInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	lastPolled   map[string]time.Time
	isPolling    bool
}

func New(agg *aggregator.Aggregator, cacheManager *cache.Manager, storage storage.Storage, feeds map[string]config.TopicConfig, pollInterval time.Duration) *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Poller{
		aggregator:   agg,
		cacheManager: cacheManager,
		storage:      storage,
		feeds:        feeds,
		parser:       gofeed.NewParser(),
		pollInterval: pollInterval,
		ctx:          ctx,
		cancel:       cancel,
		lastPolled:   make(map[string]time.Time),
	}
}

func (p *Poller) Start() {
	p.mu.Lock()
	if p.isPolling {
		p.mu.Unlock()
		return
	}
	p.isPolling = true
	p.mu.Unlock()

	log.Printf("Starting RSS feed poller with interval: %v", p.pollInterval)

	p.wg.Add(1)
	go p.pollLoop()
}

func (p *Poller) Stop() {
	p.mu.Lock()
	if !p.isPolling {
		p.mu.Unlock()
		return
	}
	p.isPolling = false
	p.mu.Unlock()

	log.Println("Stopping RSS feed poller...")
	p.cancel()
	p.wg.Wait()
	log.Println("RSS feed poller stopped")
}

func (p *Poller) pollLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	// Poll immediately on start
	p.pollAllFeeds()

	for {
		select {
		case <-ticker.C:
			p.pollAllFeeds()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Poller) pollAllFeeds() {
	log.Println("Starting background feed polling...")

	var wg sync.WaitGroup
	for topic := range p.feeds {
		wg.Add(1)
		go func(topicName string) {
			defer wg.Done()
			p.pollTopicFeeds(topicName)
		}(topic)
	}

	wg.Wait()
	log.Println("Background feed polling completed")
}

func (p *Poller) pollTopicFeeds(topic string) {
	log.Printf("Polling feeds for topic: %s", topic)

	topicConfig, exists := p.feeds[topic]
	if !exists {
		log.Printf("Topic '%s' not found in configuration", topic)
		return
	}

	// Fetch articles from all feeds for this topic
	articles, err := p.fetchFeedsParallel(topicConfig.URLs)
	if err != nil {
		log.Printf("Error fetching feeds for topic '%s': %v", topic, err)
		p.lastPolled[topic] = time.Now()
		return
	}

	// Filter articles based on topic configuration
	filteredArticles := p.filterArticlesForTopic(articles, topicConfig.Filters)

	if len(filteredArticles) == 0 {
		log.Printf("No articles found for topic: %s", topic)
		p.lastPolled[topic] = time.Now()
		return
	}

	// Create aggregated feed
	feed := &models.AggregatedFeed{
		Topic:    topic,
		Articles: filteredArticles,
		Count:    len(filteredArticles),
		Updated:  time.Now(),
	}

	// Save to storage
	if err := p.storage.SaveFeed(topic, feed); err != nil {
		log.Printf("Error saving feed for topic '%s': %v", topic, err)
	} else {
		log.Printf("Saved %d articles for topic: %s", len(filteredArticles), topic)
	}

	// Update cache
	cacheKey := fmt.Sprintf("feed:%s", topic)
	p.cacheManager.Set(cacheKey, feed, 0)

	// Update last polled time
	p.lastPolled[topic] = time.Now()
}

func (p *Poller) filterArticlesForTopic(articles []models.Article, filters []string) []models.Article {
	if len(filters) == 0 {
		// No filters specified, return all articles
		return articles
	}

	var filteredArticles []models.Article

	for _, article := range articles {
		if p.articleMatchesFilters(article, filters) {
			filteredArticles = append(filteredArticles, article)
		}
	}

	return filteredArticles
}

func (p *Poller) articleMatchesFilters(article models.Article, filters []string) bool {
	// Create a combined text field for searching
	articleText := strings.ToLower(strings.Join([]string{
		article.Title,
		article.Description,
		article.Content,
		article.Author,
		strings.Join(article.Categories, " "),
	}, " "))

	// Check if any filter term matches (OR logic)
	for _, filter := range filters {
		if strings.Contains(articleText, strings.ToLower(filter)) {
			return true
		}
	}

	return false
}

func (p *Poller) fetchFeedsParallel(feedURLs []string) ([]models.Article, error) {
	var wg sync.WaitGroup
	results := make(chan aggregator.FeedResult, len(feedURLs))

	// Start goroutines for each feed URL
	for _, url := range feedURLs {
		wg.Add(1)
		go func(feedURL string) {
			defer wg.Done()
			articles, err := p.fetchFeed(feedURL)
			results <- aggregator.FeedResult{
				URL:      feedURL,
				Articles: articles,
				Error:    err,
			}
		}(url)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results with timeout
	timeout := time.After(30 * time.Second)
	var allArticles []models.Article

	for {
		select {
		case result, ok := <-results:
			if !ok {
				return allArticles, nil
			}
			if result.Error != nil {
				log.Printf("Error polling feed %s: %v", result.URL, result.Error)
			} else {
				allArticles = append(allArticles, result.Articles...)
			}
		case <-timeout:
			log.Printf("Timeout waiting for feed results")
			return allArticles, nil
		}
	}
}

func (p *Poller) fetchFeed(url string) ([]models.Article, error) {
	feed, err := p.parser.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("http error: %v", err)
	}

	var articles []models.Article
	for _, item := range feed.Items {
		// Safely get author name
		authorName := ""
		if item.Author != nil {
			authorName = item.Author.Name
		}

		article := models.Article{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Content,
			Author:      authorName,
			Source:      feed.Title,
			Categories:  []string{},
			PublishedAt: time.Now(),
		}

		// Extract categories
		for _, category := range item.Categories {
			article.Categories = append(article.Categories, category)
		}

		// Parse published date
		if item.PublishedParsed != nil {
			article.PublishedAt = *item.PublishedParsed
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (p *Poller) GetLastPolledTime() map[string]time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]time.Time)
	for topic, time := range p.lastPolled {
		result[topic] = time
	}
	return result
}

func (p *Poller) IsPolling() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isPolling
}

func (p *Poller) ForcePoll(topic string) error {
	log.Printf("Force polling topic: %s", topic)

	if _, exists := p.feeds[topic]; !exists {
		return fmt.Errorf("topic '%s' not found", topic)
	}

	p.pollTopicFeeds(topic)
	return nil
}
