package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorssag/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dataDir string) (*SQLiteStorage, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "rss_aggregator.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_synchronous=NORMAL&_cache_size=10000&_temp_store=MEMORY")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %v", err)
	}

	// Create tables with proper indexing
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &SQLiteStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
	// Create topics table
	topicsTable := `
	CREATE TABLE IF NOT EXISTS topics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create articles table with comprehensive indexing
	articlesTable := `
	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		topic_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		link TEXT NOT NULL,
		description TEXT,
		content TEXT,
		author TEXT,
		source TEXT NOT NULL,
		categories TEXT, -- JSON array
		published_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
	);`

	// Create indexes for fast OData queries
	indexes := []string{
		// Primary indexes for common queries
		"CREATE INDEX IF NOT EXISTS idx_articles_topic_id ON articles(topic_id);",
		"CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_articles_author ON articles(author);",
		"CREATE INDEX IF NOT EXISTS idx_articles_source ON articles(source);",

		// Text search indexes using LIKE for basic text search
		"CREATE INDEX IF NOT EXISTS idx_articles_title_like ON articles(title);",
		"CREATE INDEX IF NOT EXISTS idx_articles_description_like ON articles(description);",
		"CREATE INDEX IF NOT EXISTS idx_articles_content_like ON articles(content);",

		// Composite indexes for complex queries
		"CREATE INDEX IF NOT EXISTS idx_articles_topic_published ON articles(topic_id, published_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_articles_author_source ON articles(author, source);",
	}

	// Execute table creation
	if _, err := db.Exec(topicsTable); err != nil {
		return fmt.Errorf("failed to create topics table: %v", err)
	}

	if _, err := db.Exec(articlesTable); err != nil {
		return fmt.Errorf("failed to create articles table: %v", err)
	}

	// Execute index creation
	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %v", err)
		}
	}

	return nil
}

func (s *SQLiteStorage) SaveFeed(topic string, feed *models.AggregatedFeed) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get or create topic
	topicID, err := s.getOrCreateTopic(tx, topic)
	if err != nil {
		return err
	}

	// Delete existing articles for this topic
	if _, err := tx.Exec("DELETE FROM articles WHERE topic_id = ?", topicID); err != nil {
		return fmt.Errorf("failed to delete existing articles: %v", err)
	}

	// Insert new articles
	stmt, err := tx.Prepare(`
		INSERT INTO articles (topic_id, title, link, description, content, author, source, categories, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	for _, article := range feed.Articles {
		categoriesJSON, _ := json.Marshal(article.Categories)

		_, err := stmt.Exec(
			topicID,
			article.Title,
			article.Link,
			article.Description,
			article.Content,
			article.Author,
			article.Source,
			string(categoriesJSON),
			article.PublishedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert article: %v", err)
		}
	}

	// Update topic timestamp
	if _, err := tx.Exec("UPDATE topics SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", topicID); err != nil {
		return fmt.Errorf("failed to update topic timestamp: %v", err)
	}

	return tx.Commit()
}

func (s *SQLiteStorage) LoadFeed(topic string) (*models.AggregatedFeed, error) {
	// Get topic ID
	var topicID int
	err := s.db.QueryRow("SELECT id FROM topics WHERE name = ?", topic).Scan(&topicID)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %v", err)
	}

	// Query articles with ordering
	rows, err := s.db.Query(`
		SELECT title, link, description, content, author, source, categories, published_at
		FROM articles 
		WHERE topic_id = ? 
		ORDER BY published_at DESC
	`, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %v", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var categoriesJSON string

		err := rows.Scan(
			&article.Title,
			&article.Link,
			&article.Description,
			&article.Content,
			&article.Author,
			&article.Source,
			&categoriesJSON,
			&article.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %v", err)
		}

		// Parse categories JSON
		if categoriesJSON != "" {
			json.Unmarshal([]byte(categoriesJSON), &article.Categories)
		}

		articles = append(articles, article)
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles found for topic '%s'", topic)
	}

	return &models.AggregatedFeed{
		Topic:    topic,
		Articles: articles,
		Count:    len(articles),
		Updated:  time.Now(),
	}, nil
}

func (s *SQLiteStorage) QueryArticles(topic string, query *models.ODataQuery) ([]models.Article, error) {
	// Get topic ID
	var topicID int
	err := s.db.QueryRow("SELECT id FROM topics WHERE name = ?", topic).Scan(&topicID)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %v", err)
	}

	// Build SQL query with OData support
	sqlQuery, args := s.buildODataQuery(topicID, query)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %v", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var categoriesJSON string

		err := rows.Scan(
			&article.Title,
			&article.Link,
			&article.Description,
			&article.Content,
			&article.Author,
			&article.Source,
			&categoriesJSON,
			&article.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %v", err)
		}

		// Parse categories JSON
		if categoriesJSON != "" {
			json.Unmarshal([]byte(categoriesJSON), &article.Categories)
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (s *SQLiteStorage) buildODataQuery(topicID int, query *models.ODataQuery) (string, []interface{}) {
	baseQuery := `
		SELECT title, link, description, content, author, source, categories, published_at
		FROM articles 
		WHERE topic_id = ?
	`
	args := []interface{}{topicID}

	// Add search conditions
	if len(query.Search) > 0 {
		searchConditions := make([]string, len(query.Search))
		for i, term := range query.Search {
			searchConditions[i] = "(title LIKE ? OR description LIKE ? OR content LIKE ? OR author LIKE ? OR source LIKE ?)"
			args = append(args, "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
		baseQuery += " AND (" + join(searchConditions, " OR ") + ")"
	}

	// Add filter conditions (basic support - full OData parsing would be more complex)
	if query.Filter != "" {
		// For now, support basic text search in filter
		baseQuery += " AND (title LIKE ? OR description LIKE ? OR content LIKE ? OR author LIKE ?)"
		args = append(args, "%"+query.Filter+"%", "%"+query.Filter+"%", "%"+query.Filter+"%", "%"+query.Filter+"%")
	}

	// Add ordering
	if query.OrderBy != "" {
		baseQuery += " ORDER BY " + s.parseOrderBy(query.OrderBy)
	} else {
		baseQuery += " ORDER BY published_at DESC"
	}

	// Add pagination
	if query.Top > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, query.Top)
	}

	if query.Skip > 0 {
		baseQuery = "SELECT * FROM (" + baseQuery + ") LIMIT -1 OFFSET ?"
		args = append(args, query.Skip)
	}

	return baseQuery, args
}

func (s *SQLiteStorage) parseOrderBy(orderBy string) string {
	// Simple order by parsing - can be extended for more complex cases
	switch orderBy {
	case "title asc":
		return "title ASC"
	case "title desc":
		return "title DESC"
	case "author asc":
		return "author ASC"
	case "author desc":
		return "author DESC"
	case "source asc":
		return "source ASC"
	case "source desc":
		return "source DESC"
	case "published_at asc":
		return "published_at ASC"
	case "published_at desc":
		return "published_at DESC"
	default:
		return "published_at DESC"
	}
}

func (s *SQLiteStorage) getOrCreateTopic(tx *sql.Tx, topic string) (int, error) {
	var topicID int

	// Try to get existing topic
	err := tx.QueryRow("SELECT id FROM topics WHERE name = ?", topic).Scan(&topicID)
	if err == nil {
		return topicID, nil
	}

	// Create new topic
	result, err := tx.Exec("INSERT INTO topics (name) VALUES (?)", topic)
	if err != nil {
		return 0, fmt.Errorf("failed to create topic: %v", err)
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get topic ID: %v", err)
	}

	return int(newID), nil
}

func (s *SQLiteStorage) ListTopics() ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM topics ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to query topics: %v", err)
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %v", err)
		}
		topics = append(topics, topic)
	}

	return topics, nil
}

func (s *SQLiteStorage) GetFeedInfo(topic string) (*models.FeedInfo, error) {
	var topicID int
	var updatedAt time.Time
	var articleCount int

	// Get topic info
	err := s.db.QueryRow("SELECT id, updated_at FROM topics WHERE name = ?", topic).Scan(&topicID, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("topic not found: %v", err)
	}

	// Get article count
	err = s.db.QueryRow("SELECT COUNT(*) FROM articles WHERE topic_id = ?", topicID).Scan(&articleCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles: %v", err)
	}

	return &models.FeedInfo{
		Topic:        topic,
		FileSize:     0, // Not applicable for SQLite
		LastModified: updatedAt,
		ArticleCount: articleCount,
	}, nil
}

func (s *SQLiteStorage) DeleteFeed(topic string) error {
	_, err := s.db.Exec("DELETE FROM topics WHERE name = ?", topic)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %v", err)
	}
	return nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
