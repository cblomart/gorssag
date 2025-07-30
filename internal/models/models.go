package models

import (
	"time"
)

// Article represents a single RSS article
type Article struct {
	ID          string    `json:"id"` // Unique identifier for the article
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	Source      string    `json:"source"`
	Categories  []string  `json:"categories"`
	Topic       string    `json:"topic,omitempty"` // Topic this article belongs to
}

// AggregatedFeed represents an aggregated RSS feed for a topic
type AggregatedFeed struct {
	Topic    string    `json:"topic"`
	Articles []Article `json:"articles"`
	Count    int       `json:"count"`
	Updated  time.Time `json:"updated"`
}

// FeedInfo represents metadata about a stored feed
type FeedInfo struct {
	Topic        string    `json:"topic"`
	FileSize     int64     `json:"file_size"`
	LastModified time.Time `json:"last_modified"`
	ArticleCount int       `json:"article_count"`
}

// FeedStatus represents the status of a feed
type FeedStatus struct {
	URL               string    `json:"url"`
	Topic             string    `json:"topic"`
	LastPolled        time.Time `json:"last_polled"`
	LastError         string    `json:"last_error,omitempty"`
	IsDisabled        bool      `json:"is_disabled"`
	DisabledReason    string    `json:"disabled_reason,omitempty"`
	ArticlesCount     int       `json:"articles_count"`
	ErrorCount        int       `json:"error_count"`
	ConsecutiveErrors int       `json:"consecutive_errors"`
	LastSuccess       time.Time `json:"last_success,omitempty"`
	NextRetry         time.Time `json:"next_retry,omitempty"`
	RetryCount        int       `json:"retry_count"`
	IsContentIssue    bool      `json:"is_content_issue"`             // True if disabled due to content quality
	UserAgent         string    `json:"user_agent,omitempty"`         // Working User-Agent for this feed
	TestedUserAgents  []string  `json:"tested_user_agents,omitempty"` // List of User-Agents already tested
}

// ODataQuery represents OData query parameters
type ODataQuery struct {
	Filter   string     `json:"filter"`
	OrderBy  string     `json:"orderby"`
	Select   []string   `json:"select"`
	Search   []string   `json:"search"` // Global search terms (OR logic)
	Top      int        `json:"top"`
	Skip     int        `json:"skip"`
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	Source   string     `json:"source,omitempty"`
	Author   string     `json:"author,omitempty"`
	Category string     `json:"category,omitempty"`
}

// FilterCriteria represents filter conditions
type FilterCriteria struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}
