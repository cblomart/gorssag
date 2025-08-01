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
	ListTopics() ([]string, error)
	GetFeedInfo(topic string) (*models.FeedInfo, error)
	DeleteFeed(topic string) error
	Close() error

	// Storage optimization methods
	CleanupOldArticles(retention time.Duration) error
	OptimizeDatabase() error
	GetDatabaseStats() (map[string]interface{}, error)
	RemoveDuplicateArticles() error
	CompressOldArticles() error
	GetFeedStats() (map[string]interface{}, error) // New method for feed statistics
}
