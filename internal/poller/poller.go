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

		// Convert description to markdown
		description := convertHTMLToMarkdown(item.Description)

		// Convert content to markdown, fallback to description if content is empty
		content := convertHTMLToMarkdown(item.Content)

		// Multiple fallback strategies to ensure we always have content
		if content == "" {
			if description != "" {
				// Use description as content
				content = description
			} else if item.Title != "" {
				// Use title as minimal content if both content and description are empty
				content = "# " + item.Title + "\n\n*No additional content available.*"
			} else {
				// Last resort: generic content
				content = "*Content not available*"
			}
		}

		article := models.Article{
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

	// Store original HTML for fallback
	originalHTML := html

	// Remove HTML tags and convert to Markdown
	text := html

	// Handle CDATA sections first
	text = regexp.MustCompile(`<!\[CDATA\[(.*?)\]\]>`).ReplaceAllString(text, "$1")

	// Convert images - handle both img tags and img elements within links
	// First, handle standalone img tags
	text = regexp.MustCompile(`<img[^>]*src="([^"]*)"[^>]*alt="([^"]*)"[^>]*>`).ReplaceAllStringFunc(text, func(match string) string {
		// Extract alt text and use it if meaningful, otherwise skip the image
		altMatch := regexp.MustCompile(`<img[^>]*src="([^"]*)"[^>]*alt="([^"]*)"[^>]*>`).FindStringSubmatch(match)
		if len(altMatch) >= 3 {
			altText := strings.TrimSpace(altMatch[2])
			altTextLower := strings.ToLower(altText)
			// Only include alt text if it's meaningful and not just a generic description
			// Skip if it's empty, generic, or just describes the image
			if altText != "" &&
				altText != "image" &&
				altText != "Image" &&
				!strings.Contains(altTextLower, "image") &&
				!strings.Contains(altTextLower, "presentation") &&
				!strings.Contains(altTextLower, "banner") &&
				!strings.Contains(altTextLower, "header") &&
				!strings.Contains(altTextLower, "logo") &&
				!strings.Contains(altTextLower, "icon") &&
				!strings.Contains(altTextLower, "tech") &&
				!strings.Contains(altTextLower, "modern") &&
				!strings.Contains(altTextLower, "futuristic") &&
				!strings.Contains(altTextLower, "business") &&
				!strings.Contains(altTextLower, "company") &&
				len(altText) < 50 { // More restrictive length check
				return "**" + altText + "**"
			}
		}
		// Remove image if no meaningful alt text
		return ""
	})
	text = regexp.MustCompile(`<img[^>]*alt="([^"]*)"[^>]*src="([^"]*)"[^>]*>`).ReplaceAllStringFunc(text, func(match string) string {
		// Extract alt text and use it if meaningful, otherwise skip the image
		altMatch := regexp.MustCompile(`<img[^>]*alt="([^"]*)"[^>]*src="([^"]*)"[^>]*>`).FindStringSubmatch(match)
		if len(altMatch) >= 3 {
			altText := strings.TrimSpace(altMatch[1])
			altTextLower := strings.ToLower(altText)
			// Only include alt text if it's meaningful and not just a generic description
			// Skip if it's empty, generic, or just describes the image
			if altText != "" &&
				altText != "image" &&
				altText != "Image" &&
				!strings.Contains(altTextLower, "image") &&
				!strings.Contains(altTextLower, "presentation") &&
				!strings.Contains(altTextLower, "banner") &&
				!strings.Contains(altTextLower, "header") &&
				!strings.Contains(altTextLower, "logo") &&
				!strings.Contains(altTextLower, "icon") &&
				!strings.Contains(altTextLower, "tech") &&
				!strings.Contains(altTextLower, "modern") &&
				!strings.Contains(altTextLower, "futuristic") &&
				!strings.Contains(altTextLower, "business") &&
				!strings.Contains(altTextLower, "company") &&
				len(altText) < 50 { // More restrictive length check
				return "**" + altText + "**"
			}
		}
		// Remove image if no meaningful alt text
		return ""
	})
	text = regexp.MustCompile(`<img[^>]*src="([^"]*)"[^>]*>`).ReplaceAllString(text, "")

	// Handle images within links (like Blogger's image links) - remove them entirely
	text = regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>\s*<img[^>]*src="([^"]*)"[^>]*alt="([^"]*)"[^>]*>\s*</a>`).ReplaceAllStringFunc(text, func(match string) string {
		// Extract alt text and use it if meaningful, otherwise skip the image link
		altMatch := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>\s*<img[^>]*src="([^"]*)"[^>]*alt="([^"]*)"[^>]*>\s*</a>`).FindStringSubmatch(match)
		if len(altMatch) >= 4 {
			altText := strings.TrimSpace(altMatch[3])
			altTextLower := strings.ToLower(altText)
			// Only include alt text if it's meaningful and not just a generic description
			// Skip if it's empty, generic, or just describes the image
			if altText != "" &&
				altText != "image" &&
				altText != "Image" &&
				!strings.Contains(altTextLower, "image") &&
				!strings.Contains(altTextLower, "presentation") &&
				!strings.Contains(altTextLower, "banner") &&
				!strings.Contains(altTextLower, "header") &&
				!strings.Contains(altTextLower, "logo") &&
				!strings.Contains(altTextLower, "icon") &&
				!strings.Contains(altTextLower, "tech") &&
				!strings.Contains(altTextLower, "modern") &&
				!strings.Contains(altTextLower, "futuristic") &&
				!strings.Contains(altTextLower, "business") &&
				!strings.Contains(altTextLower, "company") &&
				len(altText) < 50 { // More restrictive length check
				return "**" + altText + "**"
			}
		}
		// Remove image link if no meaningful alt text
		return ""
	})
	text = regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>\s*<img[^>]*src="([^"]*)"[^>]*>\s*</a>`).ReplaceAllString(text, "")

	// Remove any remaining empty image links that might be decorative
	text = regexp.MustCompile(`\[\]\([^)]*\)`).ReplaceAllString(text, "")

	// Remove any remaining image links that are just domain names (like blogger.googleusercontent.com)
	text = regexp.MustCompile(`\[[^\]]*\.googleusercontent\.com\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\.blogger\.com\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\.wordpress\.com\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\.medium\.com\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\.substack\.com\]\([^)]*\)`).ReplaceAllString(text, "")

	// More general pattern to remove any image links that contain common image hosting domains
	text = regexp.MustCompile(`\[[^\]]*googleusercontent[^\]]*\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*blogger[^\]]*\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*wordpress[^\]]*\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*medium[^\]]*\]\([^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*substack[^\]]*\]\([^)]*\)`).ReplaceAllString(text, "")

	// Final cleanup: remove any link that contains image hosting domains in the URL
	text = regexp.MustCompile(`\[[^\]]*\]\([^)]*googleusercontent[^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\]\([^)]*blogger[^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\]\([^)]*wordpress[^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\]\([^)]*medium[^)]*\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[[^\]]*\]\([^)]*substack[^)]*\)`).ReplaceAllString(text, "")

	// Specific cleanup for the exact pattern we're seeing
	text = regexp.MustCompile(`\[blogger\.googleusercontent\.com\]\([^)]*\)`).ReplaceAllString(text, "")

	// Convert common HTML elements to Markdown
	text = regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`).ReplaceAllString(text, "# $1\n\n")
	text = regexp.MustCompile(`<h2[^>]*>(.*?)</h2>`).ReplaceAllString(text, "## $1\n\n")
	text = regexp.MustCompile(`<h3[^>]*>(.*?)</h3>`).ReplaceAllString(text, "### $1\n\n")
	text = regexp.MustCompile(`<h4[^>]*>(.*?)</h4>`).ReplaceAllString(text, "#### $1\n\n")
	text = regexp.MustCompile(`<h5[^>]*>(.*?)</h5>`).ReplaceAllString(text, "##### $1\n\n")
	text = regexp.MustCompile(`<h6[^>]*>(.*?)</h6>`).ReplaceAllString(text, "###### $1\n\n")

	// Convert strong and bold tags
	text = regexp.MustCompile(`<strong[^>]*>(.*?)</strong>`).ReplaceAllString(text, "**$1**")
	text = regexp.MustCompile(`<b[^>]*>(.*?)</b>`).ReplaceAllString(text, "**$1**")

	// Convert emphasis and italic tags
	text = regexp.MustCompile(`<em[^>]*>(.*?)</em>`).ReplaceAllString(text, "*$1*")
	text = regexp.MustCompile(`<i[^>]*>(.*?)</i>`).ReplaceAllString(text, "*$1*")

	// Convert links
	text = regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`).ReplaceAllStringFunc(text, func(match string) string {
		// Extract href and text content
		hrefMatch := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`).FindStringSubmatch(match)
		if len(hrefMatch) >= 3 {
			href := hrefMatch[1]
			linkText := cleanText(hrefMatch[2])

			// If link text is empty or just whitespace, skip the link entirely
			if strings.TrimSpace(linkText) == "" {
				return ""
			}

			// If link text is just the URL or a domain, use a descriptive name
			if linkText == href || strings.Contains(linkText, "://") {
				// Extract domain name for better readability
				if strings.Contains(href, "://") {
					parts := strings.Split(href, "://")
					if len(parts) > 1 {
						domain := strings.Split(parts[1], "/")[0]
						linkText = domain
					} else {
						linkText = "Link"
					}
				} else {
					linkText = "Link"
				}
			}

			return fmt.Sprintf("[%s](%s)", linkText, href)
		}
		return match
	})

	// Fix double brackets in links (convert [[text]](url) to [text](url))
	text = regexp.MustCompile(`\[\[([^\]]*)\]\]\(([^)]*)\)`).ReplaceAllString(text, "[$1]($2)")

	// Convert unordered lists with better spacing
	text = regexp.MustCompile(`<ul[^>]*>(.*?)</ul>`).ReplaceAllStringFunc(text, func(match string) string {
		content := regexp.MustCompile(`<ul[^>]*>(.*?)</ul>`).ReplaceAllString(match, "$1")
		content = regexp.MustCompile(`<li[^>]*>(.*?)</li>`).ReplaceAllString(content, "- $1\n")
		return "\n" + content + "\n"
	})

	// Convert ordered lists with better spacing
	text = regexp.MustCompile(`<ol[^>]*>(.*?)</ol>`).ReplaceAllStringFunc(text, func(match string) string {
		content := regexp.MustCompile(`<ol[^>]*>(.*?)</ol>`).ReplaceAllString(match, "$1")
		content = regexp.MustCompile(`<li[^>]*>(.*?)</li>`).ReplaceAllStringFunc(content, func(liMatch string) string {
			liContent := regexp.MustCompile(`<li[^>]*>(.*?)</li>`).ReplaceAllString(liMatch, "$1")
			return "1. " + liContent + "\n"
		})
		return "\n" + content + "\n"
	})

	// Convert paragraphs with better spacing
	text = regexp.MustCompile(`<p[^>]*>(.*?)</p>`).ReplaceAllStringFunc(text, func(match string) string {
		content := regexp.MustCompile(`<p[^>]*>(.*?)</p>`).ReplaceAllString(match, "$1")
		content = cleanText(content)
		if content != "" {
			return content + "\n\n"
		}
		return ""
	})

	// Convert line breaks
	text = regexp.MustCompile(`<br[^>]*>`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`<br/>`).ReplaceAllString(text, "\n")

	// Convert blockquotes
	text = regexp.MustCompile(`<blockquote[^>]*>(.*?)</blockquote>`).ReplaceAllStringFunc(text, func(match string) string {
		content := regexp.MustCompile(`<blockquote[^>]*>(.*?)</blockquote>`).ReplaceAllString(match, "$1")
		content = cleanText(content)
		if content != "" {
			return "\n> " + content + "\n\n"
		}
		return ""
	})

	// Convert pre and code blocks
	text = regexp.MustCompile(`<pre[^>]*>(.*?)</pre>`).ReplaceAllStringFunc(text, func(match string) string {
		content := regexp.MustCompile(`<pre[^>]*>(.*?)</pre>`).ReplaceAllString(match, "$1")
		content = cleanText(content)
		if content != "" {
			return "\n```\n" + content + "\n```\n\n"
		}
		return ""
	})
	text = regexp.MustCompile(`<code[^>]*>(.*?)</code>`).ReplaceAllString(text, "`$1`")

	// Convert divs with better spacing
	text = regexp.MustCompile(`<div[^>]*>(.*?)</div>`).ReplaceAllStringFunc(text, func(match string) string {
		content := regexp.MustCompile(`<div[^>]*>(.*?)</div>`).ReplaceAllString(match, "$1")
		content = cleanText(content)
		if content != "" {
			return content + "\n\n"
		}
		return ""
	})

	// Convert spans (just clean the content)
	text = regexp.MustCompile(`<span[^>]*>(.*?)</span>`).ReplaceAllString(text, "$1")

	// Remove any remaining HTML tags
	text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&mdash;", "—")
	text = strings.ReplaceAll(text, "&ndash;", "–")
	text = strings.ReplaceAll(text, "&hellip;", "...")
	text = strings.ReplaceAll(text, "&ldquo;", "\"")
	text = strings.ReplaceAll(text, "&rdquo;", "\"")
	text = strings.ReplaceAll(text, "&lsquo;", "'")
	text = strings.ReplaceAll(text, "&rsquo;", "'")

	// Clean up whitespace and ensure proper paragraph breaks
	text = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(text, "\n\n")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Add proper line breaks after sentences and improve readability
	text = regexp.MustCompile(`([.!?])\s+([A-Z])`).ReplaceAllString(text, "$1\n\n$2")

	// Clean up excessive line breaks
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// Ensure proper spacing around headers
	text = regexp.MustCompile(`([^\n])\n(#+\s)`).ReplaceAllString(text, "$1\n\n$2")

	// If the result is still empty after all processing, try to extract any text content
	if text == "" {
		// Last resort: extract any text that might be left
		text = regexp.MustCompile(`[^<>]+`).FindString(originalHTML)
		if text != "" {
			text = strings.TrimSpace(text)
		}
	}

	// Final cleanup: remove any remaining empty links and clean up whitespace
	text = regexp.MustCompile(`\[\]\([^)]*\)`).ReplaceAllString(text, "")    // Remove empty links
	text = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(text, "\n\n") // Normalize multiple newlines
	text = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(text, "\n\n")      // Ensure proper paragraph breaks
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")             // Normalize whitespace within lines
	text = strings.TrimSpace(text)

	if text == "" && originalHTML != "" {
		// Last resort: if we still have nothing but original HTML had content, return a cleaned version
		text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(originalHTML, "")
		text = strings.TrimSpace(text)
	}

	return text
}

// cleanText removes HTML tags from text content
func cleanText(html string) string {
	if html == "" {
		return ""
	}

	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
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
