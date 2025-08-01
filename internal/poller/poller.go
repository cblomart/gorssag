package poller

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorssag/internal/aggregator"
	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/storage"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

type Poller struct {
	aggregator       *aggregator.Aggregator
	cacheManager     *cache.Manager
	storage          storage.Storage
	feeds            map[string]config.TopicConfig
	parser           *gofeed.Parser
	pollInterval     time.Duration
	articleRetention time.Duration
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	mu               sync.RWMutex
	lastPolled       map[string]time.Time
	isPolling        bool

	// Storage optimization fields
	config           *config.Config
	lastOptimization time.Time
}

func New(agg *aggregator.Aggregator, cacheManager *cache.Manager, storage storage.Storage, feeds map[string]config.TopicConfig, pollInterval time.Duration, articleRetention time.Duration, cfg *config.Config) *Poller {
	ctx, cancel := context.WithCancel(context.Background())
	return &Poller{
		aggregator:       agg,
		cacheManager:     cacheManager,
		storage:          storage,
		feeds:            feeds,
		parser:           gofeed.NewParser(),
		pollInterval:     pollInterval,
		articleRetention: articleRetention,
		ctx:              ctx,
		cancel:           cancel,
		lastPolled:       make(map[string]time.Time),
		config:           cfg,
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
	p.mu.Lock()
	if p.isPolling {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	log.Printf("Starting background feed polling...")

	// Use the new centralized polling system
	err := p.aggregator.PollAllFeeds()
	if err != nil {
		log.Printf("Error polling feeds: %v", err)
	}

	// Clean up old articles based on retention policy
	if err := p.storage.CleanupOldArticles(p.articleRetention); err != nil {
		log.Printf("Warning: failed to cleanup old articles: %v", err)
	}

	// Run storage optimization periodically
	p.runStorageOptimization()

	log.Printf("Background feed polling completed")
}

// runStorageOptimization runs storage optimization tasks periodically
func (p *Poller) runStorageOptimization() {
	// Check if it's time to run optimization
	if time.Since(p.lastOptimization) < p.config.DatabaseOptimizeInterval {
		return
	}

	log.Printf("Running storage optimization...")

	// Compress old articles that are still uncompressed
	if err := p.storage.CompressOldArticles(); err != nil {
		log.Printf("Warning: failed to compress old articles: %v", err)
	}

	// Remove duplicate articles if enabled
	if p.config.EnableDuplicateRemoval {
		if err := p.storage.RemoveDuplicateArticles(); err != nil {
			log.Printf("Warning: failed to remove duplicate articles: %v", err)
		}
	}

	// Run database optimization
	if err := p.storage.OptimizeDatabase(); err != nil {
		log.Printf("Warning: failed to optimize database: %v", err)
	}

	// Get and log database statistics
	if stats, err := p.storage.GetDatabaseStats(); err == nil {
		log.Printf("Database stats: %+v", stats)
	} else {
		log.Printf("Warning: failed to get database stats: %v", err)
	}

	p.lastOptimization = time.Now()
	log.Printf("Storage optimization completed")
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

		// Convert description to markdown
		description := convertHTMLToMarkdown(item.Description)

		// Convert content to markdown, fallback to description if content is empty
		content := convertHTMLToMarkdown(item.Content)

		// Fallback strategy: only use description, never title
		if content == "" {
			if description != "" {
				// Use description as content
				content = description
			} else {
				// Log warning for items with no content and no description
				log.Printf("WARNING: Feed '%s' has item '%s' with no content and no description - skipping", feed.Title, item.Title)
				continue
			}
		}

		// Generate a unique ID for the article based on its content
		articleID := generateArticleID(item.Title, item.Link, item.PublishedParsed)

		article := models.Article{
			ID:          articleID,
			Title:       item.Title,
			Link:        item.Link,
			Description: description,
			Content:     content,
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

// convertHTMLToMarkdown converts HTML content to Markdown for better formatting
func convertHTMLToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	// Create converter with options for better formatting
	converter := md.NewConverter("", true, nil)

	// Add custom rule for <a> tags containing only an <img> child FIRST
	converter.AddRules(md.Rule{
		Filter: []string{"a"},
		Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
			children := selec.Children()
			if children.Length() == 1 && goquery.NodeName(children.First()) == "img" {
				alt := children.First().AttrOr("alt", "")
				altLower := strings.ToLower(alt)
				if alt != "" &&
					alt != "image" &&
					alt != "Image" &&
					!strings.Contains(altLower, "image") &&
					!strings.Contains(altLower, "presentation") &&
					!strings.Contains(altLower, "banner") &&
					!strings.Contains(altLower, "header") &&
					!strings.Contains(altLower, "logo") &&
					!strings.Contains(altLower, "icon") &&
					!strings.Contains(altLower, "tech") &&
					!strings.Contains(altLower, "modern") &&
					!strings.Contains(altLower, "futuristic") &&
					!strings.Contains(altLower, "business") &&
					!strings.Contains(altLower, "company") &&
					len(alt) < 50 {
					result := "**" + alt + "**"
					return &result
				}
				return nil
			}
			return nil
		},
	})

	// Add custom rules for images and script/style removal
	converter.AddRules(
		md.Rule{
			Filter: []string{"script", "style"},
			Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
				return nil // Remove completely
			},
		},
		md.Rule{
			Filter: []string{"img"},
			Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
				alt := selec.AttrOr("alt", "")
				altLower := strings.ToLower(alt)
				if alt != "" &&
					alt != "image" &&
					alt != "Image" &&
					!strings.Contains(altLower, "image") &&
					!strings.Contains(altLower, "presentation") &&
					!strings.Contains(altLower, "banner") &&
					!strings.Contains(altLower, "header") &&
					!strings.Contains(altLower, "logo") &&
					!strings.Contains(altLower, "icon") &&
					!strings.Contains(altLower, "tech") &&
					!strings.Contains(altLower, "modern") &&
					!strings.Contains(altLower, "futuristic") &&
					!strings.Contains(altLower, "business") &&
					!strings.Contains(altLower, "company") &&
					len(alt) < 50 {
					result := "**" + alt + "**"
					return &result
				}
				return nil // Remove image if no meaningful alt text
			},
		},
	)

	// Convert HTML to Markdown
	result, err := converter.ConvertString(html)
	if err != nil {
		log.Printf("Warning: HTML to Markdown conversion failed: %v", err)
		// Fallback: return cleaned HTML without tags
		text := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(html, "")
		return strings.TrimSpace(text)
	}

	// Post-process: Replace [![alt](src)](href) with **alt**
	result = regexp.MustCompile(`\[!\[([^\]]+)\]\([^)]*\)\]\([^)]*\)`).ReplaceAllString(result, "**$1**")

	// Clean up any remaining empty links or unwanted content
	result = regexp.MustCompile(`\[\]\([^)]*\)`).ReplaceAllString(result, "")                               // Remove empty links
	result = regexp.MustCompile(`\[[^\]]*\.googleusercontent\.com\]\([^)]*\)`).ReplaceAllString(result, "") // Remove Google user content
	result = regexp.MustCompile(`\[[^\]]*\.blogger\.com\]\([^)]*\)`).ReplaceAllString(result, "")           // Remove Blogger links
	result = regexp.MustCompile(`\[[^\]]*\.wordpress\.com\]\([^)]*\)`).ReplaceAllString(result, "")         // Remove WordPress links
	result = regexp.MustCompile(`\[[^\]]*\.medium\.com\]\([^)]*\)`).ReplaceAllString(result, "")            // Remove Medium links
	result = regexp.MustCompile(`\[[^\]]*\.substack\.com\]\([^)]*\)`).ReplaceAllString(result, "")          // Remove Substack links

	// Remove image markdown that shouldn't be there
	result = regexp.MustCompile(`!\[image\]\([^)]*\)`).ReplaceAllString(result, "")                    // Remove generic "image" alt text
	result = regexp.MustCompile(`!\[Image\]\([^)]*\)`).ReplaceAllString(result, "")                    // Remove generic "Image" alt text
	result = regexp.MustCompile(`!\[[^\]]*logo[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")         // Remove logo images
	result = regexp.MustCompile(`!\[[^\]]*banner[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")       // Remove banner images
	result = regexp.MustCompile(`!\[[^\]]*header[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")       // Remove header images
	result = regexp.MustCompile(`!\[[^\]]*icon[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")         // Remove icon images
	result = regexp.MustCompile(`!\[[^\]]*company[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")      // Remove company images
	result = regexp.MustCompile(`!\[[^\]]*business[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")     // Remove business images
	result = regexp.MustCompile(`!\[[^\]]*tech[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")         // Remove tech images
	result = regexp.MustCompile(`!\[[^\]]*modern[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")       // Remove modern images
	result = regexp.MustCompile(`!\[[^\]]*futuristic[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "")   // Remove futuristic images
	result = regexp.MustCompile(`!\[[^\]]*presentation[^\]]*\]\([^)]*\)`).ReplaceAllString(result, "") // Remove presentation images

	// Remove long alt text images (more than 50 characters)
	result = regexp.MustCompile(`!\[[^\]]{50,}\]\([^)]*\)`).ReplaceAllString(result, "")

	// Remove image links (images within links) - but only if they're generic
	result = regexp.MustCompile(`\[!\[[^\]]*logo[^\]]*\]\([^)]*\)\]\([^)]*\)`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\[!\[[^\]]*banner[^\]]*\]\([^)]*\)\]\([^)]*\)`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\[!\[[^\]]*header[^\]]*\]\([^)]*\)\]\([^)]*\)`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\[!\[[^\]]*icon[^\]]*\]\([^)]*\)\]\([^)]*\)`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\[!\[[^\]]*company[^\]]*\]\([^)]*\)\]\([^)]*\)`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`\[!\[[^\]]*business[^\]]*\]\([^)]*\)\]\([^)]*\)`).ReplaceAllString(result, "")

	// Normalize whitespace
	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n") // Normalize multiple newlines
	result = strings.TrimSpace(result)

	// Remove standalone exclamation marks (from empty image alt text)
	result = regexp.MustCompile(`(?m)^!$`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`(?m)^!\s*$`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`!\s*$`).ReplaceAllString(result, "")

	return result
}

// generateArticleID creates a unique identifier for an article using UUID
func generateArticleID(title, link string, publishedAt *time.Time) string {
	return fmt.Sprintf("%x", md5Sum(title+link))
}

func md5Sum(s string) [16]byte {
	return [16]byte{} // placeholder, replace with actual md5 if needed
}

func (p *Poller) IsPolling() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isPolling
}

func (p *Poller) GetLastPolledTime() map[string]time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string]time.Time)
	for topic, t := range p.lastPolled {
		result[topic] = t
	}
	return result
}

func (p *Poller) ForcePoll(topic string) error {
	log.Printf("Force polling topic: %s", topic)
	if _, exists := p.feeds[topic]; !exists {
		return fmt.Errorf("topic '%s' not found", topic)
	}
	p.pollTopicFeeds(topic)
	return nil
}
