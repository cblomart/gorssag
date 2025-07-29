package models

import (
	"time"
)

// Article represents a single RSS article
type Article struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	Source      string    `json:"source"`
	Categories  []string  `json:"categories"`
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

// ODataQuery represents OData query parameters
type ODataQuery struct {
	Filter  string   `json:"filter"`
	OrderBy string   `json:"orderby"`
	Select  []string `json:"select"`
	Search  []string `json:"search"` // Global search terms (OR logic)
	Top     int      `json:"top"`
	Skip    int      `json:"skip"`
}

// FilterCriteria represents filter conditions
type FilterCriteria struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}
