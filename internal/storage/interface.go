package storage

import "gorssag/internal/models"

// Storage defines the interface for different storage backends
type Storage interface {
	SaveFeed(topic string, feed *models.AggregatedFeed) error
	LoadFeed(topic string) (*models.AggregatedFeed, error)
	QueryArticles(topic string, query *models.ODataQuery) ([]models.Article, error)
	ListTopics() ([]string, error)
	GetFeedInfo(topic string) (*models.FeedInfo, error)
	DeleteFeed(topic string) error
	Close() error
}
