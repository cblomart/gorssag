package storage

import (
	"gorssag/internal/models"
	"time"
)

// Storage defines the interface for different storage backends
type Storage interface {
	SaveFeed(topic string, feed *models.AggregatedFeed) error
	LoadFeed(topic string) (*models.AggregatedFeed, error)
	QueryArticles(topic string, query *models.ODataQuery) ([]models.Article, error)
	GetAllArticles(query *models.ODataQuery) ([]models.Article, int, error) // New method for all articles across all topics
	ListTopics() ([]string, error)
	GetFeedInfo(topic string) (*models.FeedInfo, error)
	DeleteFeed(topic string) error
	Close() error

	// New feed-centric storage methods
	SaveArticles(articles []models.Article) error                  // Save articles without topic assignment
	AssignArticlesToTopic(articleIDs []string, topic string) error // Assign articles to topics after storage
	GetCombinedFilters(topics []string) ([]string, bool)           // Get combined filters for multiple topics

	// Enhanced topic membership methods
	AddArticleToTopic(articleID string, topic string) error                                 // Add a single article to a topic
	RemoveArticleFromTopic(articleID string, topic string) error                            // Remove article from topic
	GetArticleTopics(articleID string) ([]string, error)                                    // Get all topics for an article
	GetTopicArticles(topic string, query *models.ODataQuery) ([]models.Article, int, error) // Get articles for a topic using membership table

	// Storage optimization methods
	CleanupOldArticles(retention time.Duration) error
	OptimizeDatabase() error
	GetDatabaseStats() (map[string]interface{}, error)
	RemoveDuplicateArticles() error
	CompressOldArticles() error
	GetFeedStats() (map[string]interface{}, error) // New method for feed statistics
}
