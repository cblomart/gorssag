package models

import (
	"testing"
	"time"
)

func TestArticle_Fields(t *testing.T) {
	article := Article{
		ID:          "test-id",
		Title:       "Test Title",
		Link:        "https://example.com/test",
		Description: "Test description",
		Content:     "Test content",
		Author:      "Test Author",
		Source:      "Test Source",
		PublishedAt: time.Now(),
		Categories:  []string{"test", "category"},
		Topic:       "test-topic",
		Language:    "en",
	}

	// Test that all fields are set correctly
	if article.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", article.ID)
	}

	if article.Title != "Test Title" {
		t.Errorf("Expected Title 'Test Title', got '%s'", article.Title)
	}

	if article.Link != "https://example.com/test" {
		t.Errorf("Expected Link 'https://example.com/test', got '%s'", article.Link)
	}

	if article.Description != "Test description" {
		t.Errorf("Expected Description 'Test description', got '%s'", article.Description)
	}

	if article.Content != "Test content" {
		t.Errorf("Expected Content 'Test content', got '%s'", article.Content)
	}

	if article.Author != "Test Author" {
		t.Errorf("Expected Author 'Test Author', got '%s'", article.Author)
	}

	if article.Source != "Test Source" {
		t.Errorf("Expected Source 'Test Source', got '%s'", article.Source)
	}

	if len(article.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(article.Categories))
	}

	if article.Topic != "test-topic" {
		t.Errorf("Expected Topic 'test-topic', got '%s'", article.Topic)
	}

	if article.Language != "en" {
		t.Errorf("Expected Language 'en', got '%s'", article.Language)
	}
}

func TestAggregatedFeed_Fields(t *testing.T) {
	articles := []Article{
		{
			ID:    "article-1",
			Title: "Article 1",
		},
		{
			ID:    "article-2",
			Title: "Article 2",
		},
	}

	feed := AggregatedFeed{
		Topic:    "test-topic",
		Articles: articles,
		Count:    len(articles),
		Updated:  time.Now(),
	}

	// Test that all fields are set correctly
	if feed.Topic != "test-topic" {
		t.Errorf("Expected Topic 'test-topic', got '%s'", feed.Topic)
	}

	if len(feed.Articles) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(feed.Articles))
	}

	if feed.Count != 2 {
		t.Errorf("Expected Count 2, got %d", feed.Count)
	}
}

func TestFeedInfo_Fields(t *testing.T) {
	info := FeedInfo{
		Topic:        "test-topic",
		FileSize:     1024,
		LastModified: time.Now(),
		ArticleCount: 5,
	}

	// Test that all fields are set correctly
	if info.Topic != "test-topic" {
		t.Errorf("Expected Topic 'test-topic', got '%s'", info.Topic)
	}

	if info.FileSize != 1024 {
		t.Errorf("Expected FileSize 1024, got %d", info.FileSize)
	}

	if info.ArticleCount != 5 {
		t.Errorf("Expected ArticleCount 5, got %d", info.ArticleCount)
	}
}

func TestFeedStatus_Fields(t *testing.T) {
	status := FeedStatus{
		URL:               "https://example.com/feed",
		Topic:             "test-topic",
		LastPolled:        time.Now(),
		LastError:         "test error",
		IsDisabled:        false,
		DisabledReason:    "",
		ArticlesCount:     5,
		ErrorCount:        2,
		ConsecutiveErrors: 1,
		LastSuccess:       time.Now(),
		NextRetry:         time.Now(),
		RetryCount:        3,
		IsContentIssue:    false,
		UserAgent:         "test-agent",
		TestedUserAgents:  []string{"agent1", "agent2"},
	}

	// Test that all fields are set correctly
	if status.URL != "https://example.com/feed" {
		t.Errorf("Expected URL 'https://example.com/feed', got '%s'", status.URL)
	}

	if status.Topic != "test-topic" {
		t.Errorf("Expected Topic 'test-topic', got '%s'", status.Topic)
	}

	if status.LastError != "test error" {
		t.Errorf("Expected LastError 'test error', got '%s'", status.LastError)
	}

	if status.ArticlesCount != 5 {
		t.Errorf("Expected ArticlesCount 5, got %d", status.ArticlesCount)
	}

	if status.ErrorCount != 2 {
		t.Errorf("Expected ErrorCount 2, got %d", status.ErrorCount)
	}

	if status.ConsecutiveErrors != 1 {
		t.Errorf("Expected ConsecutiveErrors 1, got %d", status.ConsecutiveErrors)
	}

	if status.RetryCount != 3 {
		t.Errorf("Expected RetryCount 3, got %d", status.RetryCount)
	}

	if status.UserAgent != "test-agent" {
		t.Errorf("Expected UserAgent 'test-agent', got '%s'", status.UserAgent)
	}

	if len(status.TestedUserAgents) != 2 {
		t.Errorf("Expected 2 tested user agents, got %d", len(status.TestedUserAgents))
	}
}

func TestODataQuery_Fields(t *testing.T) {
	now := time.Now()
	query := ODataQuery{
		Filter:   "title eq 'test'",
		OrderBy:  "published_at desc",
		Select:   []string{"title", "author"},
		Search:   []string{"test", "search"},
		Top:      10,
		Skip:     5,
		DateFrom: &now,
		DateTo:   &now,
		Source:   "test-source",
		Author:   "test-author",
		Category: "test-category",
	}

	// Test that all fields are set correctly
	if query.Filter != "title eq 'test'" {
		t.Errorf("Expected Filter 'title eq 'test'', got '%s'", query.Filter)
	}

	if query.OrderBy != "published_at desc" {
		t.Errorf("Expected OrderBy 'published_at desc', got '%s'", query.OrderBy)
	}

	if len(query.Select) != 2 {
		t.Errorf("Expected 2 select fields, got %d", len(query.Select))
	}

	if len(query.Search) != 2 {
		t.Errorf("Expected 2 search terms, got %d", len(query.Search))
	}

	if query.Top != 10 {
		t.Errorf("Expected Top 10, got %d", query.Top)
	}

	if query.Skip != 5 {
		t.Errorf("Expected Skip 5, got %d", query.Skip)
	}

	if query.Source != "test-source" {
		t.Errorf("Expected Source 'test-source', got '%s'", query.Source)
	}

	if query.Author != "test-author" {
		t.Errorf("Expected Author 'test-author', got '%s'", query.Author)
	}

	if query.Category != "test-category" {
		t.Errorf("Expected Category 'test-category', got '%s'", query.Category)
	}
}

func TestFilterCriteria_Fields(t *testing.T) {
	criteria := FilterCriteria{
		Field:    "title",
		Operator: "eq",
		Value:    "test",
	}

	// Test that all fields are set correctly
	if criteria.Field != "title" {
		t.Errorf("Expected Field 'title', got '%s'", criteria.Field)
	}

	if criteria.Operator != "eq" {
		t.Errorf("Expected Operator 'eq', got '%s'", criteria.Operator)
	}

	if criteria.Value != "test" {
		t.Errorf("Expected Value 'test', got '%s'", criteria.Value)
	}
}
