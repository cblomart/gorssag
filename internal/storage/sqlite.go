package storage

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"gorssag/internal/config"
	"gorssag/internal/models"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pemistahl/lingua-go"
)

// getGoroutineID returns a unique identifier for the current goroutine
func getGoroutineID() uint64 {
	// Use a more reliable method to get goroutine ID
	// Check for potential integer overflow
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines < 0 {
		// Fallback to a safe value if overflow occurs
		return 0
	}
	// Use a combination of goroutine count and current time for uniqueness
	return uint64(numGoroutines) + uint64(time.Now().UnixNano()%1000)
}

type SQLiteStorage struct {
	db             *sql.DB
	config         *config.Config
	detector       lingua.LanguageDetector
	mutex          sync.RWMutex
	topicMutexes   map[string]*sync.Mutex
	topicMutexLock sync.Mutex
}

func NewSQLiteStorage(dataDir string, cfg *config.Config) (*SQLiteStorage, error) {
	// Ensure data directory exists with secure permissions (0750)
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "rss_aggregator.db")
	log.Printf("Initializing database at: %s", dbPath)

	// Check if database exists and validate schema
	needsRecreation := false

	// Check for force recreation environment variable
	if os.Getenv("FORCE_DB_RECREATE") == "true" {
		log.Printf("Force database recreation requested via environment variable")
		needsRecreation = true
	} else if _, err := os.Stat(dbPath); err == nil {
		// Database exists, validate schema
		log.Printf("Database exists, validating schema...")
		if !validateSchema(dbPath) {
			log.Printf("Database schema validation failed, will recreate database")
			needsRecreation = true
		}
	} else {
		// Database doesn't exist, will create it
		log.Printf("Database doesn't exist, will create new database")
		needsRecreation = true
	}

	// If recreation is needed, remove existing database
	if needsRecreation {
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove existing database: %v", err)
		}
		log.Printf("Creating new database with proper schema")
	} else {
		log.Printf("Using existing database with valid schema")
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_synchronous=NORMAL&_cache_size=10000&_temp_store=MEMORY&_timeout=30000&_busy_timeout=30000&_mmap_size=268435456")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %v", err)
	}

	// Set additional PRAGMA settings for better performance and reliability
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456",
		"PRAGMA busy_timeout = 30000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("Warning: failed to set %s: %v", pragma, err)
		}
	}

	// Create tables with proper indexing
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	// Initialize language detector with common RSS languages
	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(
			lingua.English, lingua.German, lingua.French, lingua.Spanish,
			lingua.Chinese, lingua.Russian, lingua.Italian, lingua.Portuguese,
			lingua.Dutch, lingua.Swedish, lingua.Danish, lingua.Finnish,
			lingua.Polish, lingua.Czech, lingua.Hungarian, lingua.Romanian,
		).
		Build()

	return &SQLiteStorage{
		db:           db,
		config:       cfg,
		detector:     detector,
		topicMutexes: make(map[string]*sync.Mutex),
	}, nil
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
		article_id TEXT UNIQUE NOT NULL, -- UUID for consistent article identification
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
		language TEXT DEFAULT 'en', -- New column for article language
		FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
	);

	-- Table for storing compressed content separately
	CREATE TABLE IF NOT EXISTS compressed_content (
		article_id TEXT PRIMARY KEY,
		compressed_content BLOB NOT NULL,
		compressed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles(article_id) ON DELETE CASCADE
	);

	-- Search index table for efficient full-text search with multi-language support
	CREATE TABLE IF NOT EXISTS search_index (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id TEXT NOT NULL,
		search_term TEXT NOT NULL,
		field_type TEXT NOT NULL, -- 'title', 'description', 'content', 'author', 'source'
		language TEXT NOT NULL, -- 'en', 'zh', 'de', 'fr', 'es', etc.
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles(article_id) ON DELETE CASCADE
	);

	-- Index for efficient search with language support
	CREATE INDEX IF NOT EXISTS idx_search_index_article_term ON search_index(article_id, search_term);
	CREATE INDEX IF NOT EXISTS idx_search_index_term_lang ON search_index(search_term, language);
	CREATE INDEX IF NOT EXISTS idx_search_index_field ON search_index(field_type);
	CREATE INDEX IF NOT EXISTS idx_search_index_language ON search_index(language);

	-- Article-Topic membership table for many-to-many relationships
	CREATE TABLE IF NOT EXISTS article_topics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		article_id TEXT NOT NULL,
		topic_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES articles(article_id) ON DELETE CASCADE,
		FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
		UNIQUE(article_id, topic_id) -- Prevent duplicate assignments
	);

	-- Indexes for efficient topic membership queries
	CREATE INDEX IF NOT EXISTS idx_article_topics_article ON article_topics(article_id);
	CREATE INDEX IF NOT EXISTS idx_article_topics_topic ON article_topics(topic_id);`

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

// validateSchema checks if the database has the correct schema
func validateSchema(dbPath string) bool {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Failed to open database for schema validation: %v", err)
		return false
	}
	defer db.Close()

	// Check if required tables exist
	requiredTables := []string{"topics", "articles", "compressed_content", "search_index"}
	for _, table := range requiredTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		err := db.QueryRow(query, table).Scan(&count)
		if err != nil || count == 0 {
			log.Printf("Missing required table: %s", table)
			return false
		}
	}

	// Check if articles table has all required columns
	requiredArticleColumns := []string{
		"id", "article_id", "topic_id", "title", "link", "description",
		"content", "author", "source", "categories", "published_at",
		"created_at", "updated_at", "language",
	}
	for _, column := range requiredArticleColumns {
		var count int
		query := "SELECT COUNT(*) FROM pragma_table_info('articles') WHERE name=?"
		err := db.QueryRow(query, column).Scan(&count)
		if err != nil || count == 0 {
			log.Printf("Missing required column in articles table: %s", column)
			return false
		}
	}

	// Check if topics table has required columns
	topicColumns := []string{"id", "name", "created_at", "updated_at"}
	for _, column := range topicColumns {
		var count int
		query := "SELECT COUNT(*) FROM pragma_table_info('topics') WHERE name=?"
		err := db.QueryRow(query, column).Scan(&count)
		if err != nil || count == 0 {
			log.Printf("Missing required column in topics table: %s", column)
			return false
		}
	}

	// Check if compressed_content table has required columns
	compressedColumns := []string{"article_id", "compressed_content", "compressed_at"}
	for _, column := range compressedColumns {
		var count int
		query := "SELECT COUNT(*) FROM pragma_table_info('compressed_content') WHERE name=?"
		err := db.QueryRow(query, column).Scan(&count)
		if err != nil || count == 0 {
			log.Printf("Missing required column in compressed_content table: %s", column)
			return false
		}
	}

	// Check if search_index table has required columns
	searchColumns := []string{"article_id", "search_term", "field_type", "language"}
	for _, column := range searchColumns {
		var count int
		query := "SELECT COUNT(*) FROM pragma_table_info('search_index') WHERE name=?"
		err := db.QueryRow(query, column).Scan(&count)
		if err != nil || count == 0 {
			log.Printf("Missing required column in search_index table: %s", column)
			return false
		}
	}

	log.Printf("Database schema validation passed")
	return true
}

// CleanupOldArticles removes articles older than the specified retention period
func (s *SQLiteStorage) CleanupOldArticles(retentionPeriod time.Duration) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	cutoffTime := time.Now().Add(-retentionPeriod)

	// Delete old articles
	result, err := s.db.Exec("DELETE FROM articles WHERE published_at < ?", cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old articles: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old articles (older than %v)", rowsAffected, retentionPeriod)
	}

	return nil
}

func (s *SQLiteStorage) SaveFeed(topic string, feed *models.AggregatedFeed) error {

	log.Printf("SaveFeed: [THREAD-%d] Starting to save %d articles for topic '%s'", getGoroutineID(), len(feed.Articles), topic)

	log.Printf("SaveFeed: [THREAD-%d] Beginning database transaction for topic '%s'", getGoroutineID(), topic)
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to begin transaction for topic '%s': %v", getGoroutineID(), topic, err)
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Track if transaction was committed
	committed := false
	defer func() {
		if !committed {
			log.Printf("SaveFeed: [THREAD-%d] Rolling back transaction for topic '%s'", getGoroutineID(), topic)
			if err := tx.Rollback(); err != nil {
				log.Printf("Warning: failed to rollback transaction: %v", err)
			}
		}
	}()

	// Get or create topic
	log.Printf("SaveFeed: [THREAD-%d] Getting or creating topic '%s'", getGoroutineID(), topic)
	topicID, err := s.getOrCreateTopic(tx, topic)
	if err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to get or create topic '%s': %v", getGoroutineID(), topic, err)
		return err
	}
	log.Printf("SaveFeed: [THREAD-%d] Using topic ID %d for topic '%s'", getGoroutineID(), topicID, topic)

	// Delete existing articles for this topic
	log.Printf("SaveFeed: [THREAD-%d] Deleting existing articles for topic ID %d", getGoroutineID(), topicID)
	if _, err := tx.Exec("DELETE FROM articles WHERE topic_id = ?", topicID); err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to delete existing articles for topic ID %d: %v", getGoroutineID(), topicID, err)
		return fmt.Errorf("failed to delete existing articles: %v", err)
	}
	log.Printf("SaveFeed: [THREAD-%d] Successfully deleted existing articles for topic ID %d", getGoroutineID(), topicID)

	// Insert new articles
	log.Printf("SaveFeed: [THREAD-%d] Preparing insert statement for topic '%s'", getGoroutineID(), topic)
	stmt, err := tx.Prepare(`
		INSERT INTO articles (article_id, topic_id, title, link, description, content, author, source, categories, published_at, language)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to prepare insert statement for topic '%s': %v", getGoroutineID(), topic, err)
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer func() {
		log.Printf("SaveFeed: [THREAD-%d] Closing prepared statement for topic '%s'", getGoroutineID(), topic)
		if err := stmt.Close(); err != nil {
			log.Printf("Warning: failed to close prepared statement: %v", err)
		}
	}()

	// Prepare compressed content statement
	log.Printf("SaveFeed: [THREAD-%d] Preparing compressed content statement for topic '%s'", getGoroutineID(), topic)
	compressedStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO compressed_content (article_id, compressed_content, compressed_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to prepare compressed content statement for topic '%s': %v", getGoroutineID(), topic, err)
		return fmt.Errorf("failed to prepare compressed content statement: %v", err)
	}
	defer func() {
		log.Printf("SaveFeed: [THREAD-%d] Closing compressed content statement for topic '%s'", getGoroutineID(), topic)
		if err := compressedStmt.Close(); err != nil {
			log.Printf("Warning: failed to close compressed content statement: %v", err)
		}
	}()

	threeDaysAgo := time.Now().AddDate(0, 0, -3)

	log.Printf("SaveFeed: [THREAD-%d] Starting to process %d articles for topic '%s'", getGoroutineID(), len(feed.Articles), topic)

	for i, article := range feed.Articles {
		log.Printf("SaveFeed: [THREAD-%d] Processing article %d/%d: %s", getGoroutineID(), i+1, len(feed.Articles), article.ID)

		categoriesJSON, _ := json.Marshal(article.Categories)

		// Clean and optimize content
		content := cleanAndOptimizeContent(article.Content)

		var contentToStore string
		var compressedContent []byte
		var shouldCompress bool

		if s.config != nil && s.config.EnableContentCompression &&
			article.PublishedAt.Before(threeDaysAgo) && len(content) > 0 {
			// Store empty content in main table, compressed content in separate table
			contentToStore = ""
			shouldCompress = true

			// Compress content (but don't store yet - wait until after article is inserted)
			compressed, err := compressContent(content)
			if err != nil {
				log.Printf("Warning: failed to compress content for article %s: %v", article.ID, err)
				// Fall back to uncompressed content
				contentToStore = content
				shouldCompress = false
			} else {
				compressedContent = compressed
			}
		} else {
			// Keep content uncompressed for recent articles
			contentToStore = content
			shouldCompress = false
		}

		// Detect language for the article
		articleLanguage := s.detectLanguage(article.Title + " " + article.Description + " " + article.Content)

		// Insert article using prepared statement
		log.Printf("SaveFeed: [THREAD-%d] Inserting article %s into database", getGoroutineID(), article.ID)
		_, err = stmt.Exec(article.ID, topicID, article.Title, article.Link, article.Description, contentToStore, article.Author, article.Source, categoriesJSON, article.PublishedAt, articleLanguage)
		if err != nil {
			log.Printf("SaveFeed: [THREAD-%d] Failed to insert article %s: %v", getGoroutineID(), article.ID, err)
			return fmt.Errorf("failed to insert article %s: %v", article.ID, err)
		}
		log.Printf("SaveFeed: [THREAD-%d] Successfully inserted article %s", getGoroutineID(), article.ID)

		// Now store compressed content after article is inserted (to avoid FK constraint)
		if shouldCompress && len(compressedContent) > 0 {
			if _, err := compressedStmt.Exec(article.ID, compressedContent); err != nil {
				log.Printf("Warning: failed to store compressed content for article %s: %v", article.ID, err)
			}
		}

		// Update FTS5 index with the original content (not compressed)
		log.Printf("SaveFeed: [THREAD-%d] Updating search index for article %s", getGoroutineID(), article.ID)
		if err := s.updateSearchIndexWithTx(tx, article.ID, article.Title, article.Description, content, article.Author, article.Source); err != nil {
			log.Printf("SaveFeed: [THREAD-%d] Warning: failed to update FTS index for article %s: %v", getGoroutineID(), article.ID, err)
		} else {
			log.Printf("SaveFeed: [THREAD-%d] Successfully updated search index for article %s", getGoroutineID(), article.ID)
		}
	}

	// Update topic timestamp
	log.Printf("SaveFeed: [THREAD-%d] Updating topic timestamp for topic ID %d", getGoroutineID(), topicID)
	if _, err := tx.Exec("UPDATE topics SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", topicID); err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to update topic timestamp for topic ID %d: %v", getGoroutineID(), topicID, err)
		return fmt.Errorf("failed to update topic timestamp: %v", err)
	}
	log.Printf("SaveFeed: [THREAD-%d] Successfully updated topic timestamp for topic ID %d", getGoroutineID(), topicID)

	log.Printf("SaveFeed: [THREAD-%d] Committing transaction for topic '%s'", getGoroutineID(), topic)
	if err := tx.Commit(); err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to commit transaction for topic '%s': %v", getGoroutineID(), topic, err)
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	committed = true
	log.Printf("SaveFeed: [THREAD-%d] Successfully committed transaction for topic '%s'", getGoroutineID(), topic)

	// Verify the articles were actually saved by checking the database
	log.Printf("SaveFeed: [THREAD-%d] Verifying article count for topic ID %d", getGoroutineID(), topicID)
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM articles WHERE topic_id = ?", topicID).Scan(&count)
	if err != nil {
		log.Printf("SaveFeed: [THREAD-%d] Failed to verify article count for topic ID %d: %v", getGoroutineID(), topicID, err)
	} else {
		log.Printf("SaveFeed: [THREAD-%d] Verified %d articles in database for topic '%s'", getGoroutineID(), count, topic)
		if count != len(feed.Articles) {
			log.Printf("SaveFeed: [THREAD-%d] WARNING - Expected %d articles but found %d in database for topic '%s'", getGoroutineID(), len(feed.Articles), count, topic)
		}
	}

	log.Printf("SaveFeed: [THREAD-%d] Successfully saved %d articles for topic '%s'", getGoroutineID(), len(feed.Articles), topic)
	return nil
}

func (s *SQLiteStorage) LoadFeed(topic string) (*models.AggregatedFeed, error) {
	log.Printf("LoadFeed: [THREAD-%d] Starting to load feed for topic '%s'", getGoroutineID(), topic)

	// Get topic ID with timeout
	var topicID int
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("LoadFeed: Starting database queries for topic '%s'", topic)

	// Simple diagnostic: check if topics table exists
	var tableExists int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='topics'").Scan(&tableExists)
	if err != nil {
		log.Printf("LoadFeed: Failed to check if topics table exists: %v", err)
		return nil, fmt.Errorf("database error: %v", err)
	}

	log.Printf("LoadFeed: Topics table exists: %t", tableExists > 0)

	if tableExists == 0 {
		log.Printf("LoadFeed: Topics table does not exist")
		return nil, fmt.Errorf("topics table not found")
	}

	// Get the specific topic
	err = s.db.QueryRowContext(ctx, "SELECT id FROM topics WHERE name = ?", topic).Scan(&topicID)
	if err != nil {
		log.Printf("LoadFeed: Topic '%s' not found: %v", topic, err)
		return nil, fmt.Errorf("topic not found: %v", err)
	}

	log.Printf("LoadFeed: Found topic ID %d for topic '%s'", topicID, topic)

	// Query articles with LEFT JOIN to get compressed content in a single query
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	rows, err := s.db.QueryContext(ctx2, `
		SELECT 
			a.article_id, 
			a.title, 
			a.description, 
			a.content, 
			a.link, 
			a.author, 
			a.source, 
			a.published_at, 
			a.categories, 
			a.language,
			cc.compressed_content
		FROM articles a
		LEFT JOIN compressed_content cc ON a.article_id = cc.article_id
		WHERE a.topic_id = ? 
		ORDER BY a.published_at DESC
	`, topicID)
	if err != nil {
		log.Printf("LoadFeed: Failed to query articles: %v", err)
		return nil, fmt.Errorf("failed to query articles: %v", err)
	}
	defer rows.Close()

	var articles []models.Article
	articleCount := 0
	for rows.Next() {
		var article models.Article
		var content string
		var categoriesJSON sql.NullString
		var language sql.NullString
		var compressedContent sql.NullString
		err = rows.Scan(
			&article.ID,
			&article.Title,
			&article.Description,
			&content,
			&article.Link,
			&article.Author,
			&article.Source,
			&article.PublishedAt,
			&categoriesJSON,
			&language,
			&compressedContent,
		)
		if err != nil {
			log.Printf("LoadFeed: Failed to scan article %d: %v", articleCount, err)
			return nil, fmt.Errorf("failed to scan article: %v", err)
		}

		articleCount++
		if articleCount%10 == 0 {
			log.Printf("LoadFeed: Processed %d articles for topic '%s'", articleCount, topic)
		}

		// Handle content (regular or compressed)
		article.Content = content
		if article.Content == "" && compressedContent.Valid {
			// Try to decompress content
			decompressed, err := decompressContent([]byte(compressedContent.String))
			if err != nil {
				log.Printf("Warning: failed to decompress content for article %s: %v", article.ID, err)
				article.Content = "Content unavailable (decompression failed)"
			} else {
				article.Content = decompressed
			}
		} else if article.Content == "" {
			article.Content = "Content unavailable"
		}

		// Parse categories JSON
		if categoriesJSON.Valid {
			if err := json.Unmarshal([]byte(categoriesJSON.String), &article.Categories); err != nil {
				log.Printf("Warning: failed to parse categories for article %s: %v", article.ID, err)
				article.Categories = []string{}
			}
		}

		// Set language from database
		if language.Valid {
			article.Language = language.String
		} else {
			article.Language = "en" // Default to English if not found
		}

		articles = append(articles, article)
	}

	// Check for any errors during iteration
	if err := rows.Err(); err != nil {
		log.Printf("LoadFeed: Error during rows iteration: %v", err)
		return nil, fmt.Errorf("error during rows iteration: %v", err)
	}

	log.Printf("LoadFeed: Successfully loaded %d articles for topic '%s'", len(articles), topic)

	if len(articles) == 0 {
		log.Printf("LoadFeed: No articles found for topic '%s'", topic)
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
			&article.ID,
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

		// If content is empty, try to decompress from compressed_content table
		if article.Content == "" {
			var compressedData []byte
			err := s.db.QueryRow("SELECT compressed_content FROM compressed_content WHERE article_id = ?", article.ID).Scan(&compressedData)
			if err == nil && len(compressedData) > 0 {
				decompressed, err := decompressContent(compressedData)
				if err != nil {
					log.Printf("Warning: failed to decompress content for article %s: %v", article.ID, err)
					article.Content = "Content unavailable (decompression failed)"
				} else {
					article.Content = decompressed
				}
			} else {
				article.Content = "Content unavailable"
			}
		}

		// Parse categories JSON
		if categoriesJSON != "" {
			if err := json.Unmarshal([]byte(categoriesJSON), &article.Categories); err != nil {
				log.Printf("Warning: failed to parse categories for article %s: %v", article.ID, err)
				article.Categories = []string{}
			}
		}

		articles = append(articles, article)
	}

	return articles, nil
}

// GetAllArticles returns all articles across all topics without topic filtering
func (s *SQLiteStorage) GetAllArticles(query *models.ODataQuery) ([]models.Article, int, error) {
	// Build SQL query for all articles without topic filtering
	baseQuery := `
		SELECT 
			a.article_id, 
			a.title, 
			a.description, 
			a.content, 
			a.link, 
			a.author, 
			a.source, 
			a.published_at, 
			a.categories, 
			a.language,
			t.name as topic,
			cc.compressed_content
		FROM articles a
		LEFT JOIN topics t ON a.topic_id = t.id
		LEFT JOIN compressed_content cc ON a.article_id = cc.article_id
		WHERE 1=1
	`
	args := []interface{}{}

	// Add search conditions
	if len(query.Search) > 0 {
		searchConditions := make([]string, len(query.Search))
		for i, term := range query.Search {
			searchConditions[i] = "(a.title LIKE ? OR a.description LIKE ? OR a.content LIKE ? OR a.author LIKE ? OR a.source LIKE ?)"
			args = append(args, "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
		baseQuery += " AND (" + strings.Join(searchConditions, " OR ") + ")"
	}

	// Add filter conditions
	if query.Filter != "" {
		baseQuery += " AND (a.title LIKE ? OR a.description LIKE ? OR a.content LIKE ? OR a.author LIKE ?)"
		args = append(args, "%"+query.Filter+"%", "%"+query.Filter+"%", "%"+query.Filter+"%", "%"+query.Filter+"%")
	}

	// Count total articles for pagination (before LIMIT/OFFSET)
	countQuery := `
		SELECT COUNT(DISTINCT a.article_id) 
		FROM articles a
		LEFT JOIN topics t ON a.topic_id = t.id
		LEFT JOIN compressed_content cc ON a.article_id = cc.article_id
		WHERE 1=1
	`
	countArgs := []interface{}{}

	// Apply the same search conditions to count query
	if len(query.Search) > 0 {
		searchConditions := make([]string, len(query.Search))
		for i, term := range query.Search {
			searchConditions[i] = "(a.title LIKE ? OR a.description LIKE ? OR a.content LIKE ? OR a.author LIKE ? OR a.source LIKE ?)"
			countArgs = append(countArgs, "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
		countQuery += " AND (" + strings.Join(searchConditions, " OR ") + ")"
	}

	// Apply the same filter conditions to count query
	if query.Filter != "" {
		countQuery += " AND (a.title LIKE ? OR a.description LIKE ? OR a.content LIKE ? OR a.author LIKE ?)"
		countArgs = append(countArgs, "%"+query.Filter+"%", "%"+query.Filter+"%", "%"+query.Filter+"%", "%"+query.Filter+"%")
	}

	var totalCount int
	err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count articles: %v", err)
	}

	// Add ordering
	if query.OrderBy != "" {
		baseQuery += " ORDER BY " + s.parseOrderBy(query.OrderBy)
	} else {
		baseQuery += " ORDER BY a.published_at DESC"
	}

	// Add pagination - LIMIT must come before OFFSET in SQLite
	if query.Top > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, query.Top)
		if query.Skip > 0 {
			baseQuery += " OFFSET ?"
			args = append(args, query.Skip)
		}
	} else if query.Skip > 0 {
		// If only skip is specified, we need a default limit
		baseQuery += " LIMIT -1 OFFSET ?"
		args = append(args, query.Skip)
	}

	// Execute query
	rows, err := s.db.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query all articles: %v", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var content string
		var categoriesJSON sql.NullString
		var language sql.NullString
		var topic sql.NullString
		var compressedContent sql.NullString

		err = rows.Scan(
			&article.ID,
			&article.Title,
			&article.Description,
			&content,
			&article.Link,
			&article.Author,
			&article.Source,
			&article.PublishedAt,
			&categoriesJSON,
			&language,
			&topic,
			&compressedContent,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan article: %v", err)
		}

		// Handle content (regular or compressed)
		article.Content = content
		if article.Content == "" && compressedContent.Valid {
			decompressed, err := decompressContent([]byte(compressedContent.String))
			if err != nil {
				log.Printf("Warning: failed to decompress content for article %s: %v", article.ID, err)
				article.Content = "Content unavailable (decompression failed)"
			} else {
				article.Content = decompressed
			}
		} else if article.Content == "" {
			article.Content = "Content unavailable"
		}

		// Parse categories JSON
		if categoriesJSON.Valid {
			if err := json.Unmarshal([]byte(categoriesJSON.String), &article.Categories); err != nil {
				log.Printf("Warning: failed to parse categories for article %s: %v", article.ID, err)
				article.Categories = []string{}
			}
		}

		// Set language and topic
		if language.Valid {
			article.Language = language.String
		} else {
			article.Language = "en"
		}

		if topic.Valid {
			article.Topic = topic.String
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error during rows iteration: %v", err)
	}

	return articles, totalCount, nil
}

func (s *SQLiteStorage) buildODataQuery(topicID int, query *models.ODataQuery) (string, []interface{}) {
	baseQuery := `
		SELECT article_id, title, link, description, content, author, source, categories, published_at
		FROM articles 
		WHERE topic_id = ?
	`
	args := []interface{}{topicID}

	// Add search conditions using FTS5
	if len(query.Search) > 0 {
		// Use FTS5 search instead of LIKE queries
		searchArticleIDs, err := s.searchArticlesIndex(query.Search, topicID)
		if err != nil {
			log.Printf("Warning: FTS search failed, falling back to LIKE queries: %v", err)
			// Fallback to LIKE queries
			searchConditions := make([]string, len(query.Search))
			for i, term := range query.Search {
				searchConditions[i] = "(title LIKE ? OR description LIKE ? OR content LIKE ? OR author LIKE ? OR source LIKE ?)"
				args = append(args, "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%", "%"+term+"%")
			}
			baseQuery += " AND (" + join(searchConditions, " OR ") + ")"
		} else if len(searchArticleIDs) > 0 {
			// Use FTS5 results
			placeholders := make([]string, len(searchArticleIDs))
			for i := range searchArticleIDs {
				placeholders[i] = "?"
				args = append(args, searchArticleIDs[i])
			}
			baseQuery += " AND article_id IN (" + join(placeholders, ",") + ")"
		} else {
			// No search results found
			baseQuery += " AND 1=0" // Return no results
		}
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
	s.mutex.Lock()
	defer s.mutex.Unlock()

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

// OptimizeDatabase performs database maintenance operations
func (s *SQLiteStorage) OptimizeDatabase() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// VACUUM to reclaim space and optimize storage
	if _, err := s.db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("failed to vacuum database: %v", err)
	}

	// ANALYZE to update statistics for query optimization
	if _, err := s.db.Exec("ANALYZE"); err != nil {
		return fmt.Errorf("failed to analyze database: %v", err)
	}

	// Update table statistics
	if _, err := s.db.Exec("ANALYZE articles"); err != nil {
		return fmt.Errorf("failed to analyze articles table: %v", err)
	}

	if _, err := s.db.Exec("ANALYZE topics"); err != nil {
		return fmt.Errorf("failed to analyze topics table: %v", err)
	}

	log.Printf("Database optimization completed")
	return nil
}

// GetDatabaseStats returns database statistics
func (s *SQLiteStorage) GetDatabaseStats() (map[string]interface{}, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := make(map[string]interface{})

	// Get total articles count
	var totalArticles int
	err := s.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&totalArticles)
	if err != nil {
		return nil, fmt.Errorf("failed to get total articles count: %v", err)
	}
	stats["total_articles"] = totalArticles

	// Get total topics count
	var totalTopics int
	err = s.db.QueryRow("SELECT COUNT(*) FROM topics").Scan(&totalTopics)
	if err != nil {
		return nil, fmt.Errorf("failed to get total topics count: %v", err)
	}
	stats["total_topics"] = totalTopics

	// Get average content length (handle NULL values)
	var avgContentLength sql.NullFloat64
	err = s.db.QueryRow("SELECT AVG(LENGTH(content)) FROM articles WHERE content IS NOT NULL AND content != ''").Scan(&avgContentLength)
	if err != nil {
		return nil, fmt.Errorf("failed to get average content length: %v", err)
	}
	if avgContentLength.Valid {
		stats["avg_content_length"] = avgContentLength.Float64
	} else {
		stats["avg_content_length"] = 0.0
	}

	// Get compressed content count
	var compressedCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM compressed_content").Scan(&compressedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get compressed content count: %v", err)
	}
	stats["compressed_articles"] = compressedCount

	// Get uncompressed content count
	var uncompressedCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM articles WHERE content IS NOT NULL AND content != ''").Scan(&uncompressedCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get uncompressed content count: %v", err)
	}
	stats["uncompressed_articles"] = uncompressedCount

	// Get database file size
	var dbSize int64
	err = s.db.QueryRow("SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size()").Scan(&dbSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %v", err)
	}
	stats["database_size_bytes"] = dbSize

	// Get articles by topic
	rows, err := s.db.Query(`
		SELECT t.name, COUNT(a.id) as count 
		FROM topics t 
		LEFT JOIN articles a ON t.id = a.topic_id 
		GROUP BY t.id, t.name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles by topic: %v", err)
	}
	defer rows.Close()

	articlesByTopic := make(map[string]int)
	for rows.Next() {
		var topicName string
		var count int
		if err := rows.Scan(&topicName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan topic count: %v", err)
		}
		articlesByTopic[topicName] = count
	}
	stats["articles_by_topic"] = articlesByTopic

	return stats, nil
}

// RemoveDuplicateArticles removes duplicate articles based on content similarity
func (s *SQLiteStorage) RemoveDuplicateArticles() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Remove exact duplicates based on link
	result, err := s.db.Exec(`
		DELETE FROM articles 
		WHERE id NOT IN (
			SELECT MIN(id) 
			FROM articles 
			GROUP BY link
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to remove duplicate articles: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected > 0 {
		log.Printf("Removed %d duplicate articles", rowsAffected)
	}

	return nil
}

// CompressOldArticles compresses articles older than 3 days that are still uncompressed
func (s *SQLiteStorage) CompressOldArticles() error {
	// Use a longer timeout to avoid deadlocks
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if s.config == nil || !s.config.EnableContentCompression {
		return nil // Compression not enabled
	}

	threeDaysAgo := time.Now().AddDate(0, 0, -3)

	// First, check if there are any articles to compress
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM articles a 
		LEFT JOIN compressed_content c ON a.article_id = c.article_id 
		WHERE a.published_at < ? 
		AND a.content != '' 
		AND c.article_id IS NULL
	`, threeDaysAgo).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count old articles: %v", err)
	}

	if count == 0 {
		log.Printf("No old articles to compress")
		return nil
	}

	log.Printf("Found %d old articles to compress", count)

	// Find articles older than 3 days that have content but no compressed version
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.article_id, a.content 
		FROM articles a 
		LEFT JOIN compressed_content c ON a.article_id = c.article_id 
		WHERE a.published_at < ? 
		AND a.content != '' 
		AND c.article_id IS NULL
		LIMIT 1000
	`, threeDaysAgo)
	if err != nil {
		return fmt.Errorf("failed to query old articles: %v", err)
	}
	defer rows.Close()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Track if transaction was committed
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("Warning: failed to rollback transaction: %v", err)
			}
		}
	}()

	compressedStmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO compressed_content (article_id, compressed_content, compressed_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare compressed content statement: %v", err)
	}
	defer compressedStmt.Close()

	updateStmt, err := tx.Prepare(`
		UPDATE articles SET content = '' WHERE article_id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %v", err)
	}
	defer updateStmt.Close()

	compressedCount := 0
	for rows.Next() {
		var articleID, content string
		if err := rows.Scan(&articleID, &content); err != nil {
			log.Printf("Warning: failed to scan article for compression: %v", err)
			continue
		}

		if len(content) == 0 {
			continue
		}

		// Compress content
		compressed, err := compressContent(content)
		if err != nil {
			log.Printf("Warning: failed to compress content for article %s: %v", articleID, err)
			continue
		}

		// Store compressed content
		if _, err := compressedStmt.Exec(articleID, compressed); err != nil {
			log.Printf("Warning: failed to store compressed content for article %s: %v", articleID, err)
			continue
		}

		// Clear uncompressed content
		if _, err := updateStmt.Exec(articleID); err != nil {
			log.Printf("Warning: failed to clear uncompressed content for article %s: %v", articleID, err)
			continue
		}

		compressedCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit compression transaction: %v", err)
	}
	committed = true

	if compressedCount > 0 {
		log.Printf("Compressed %d old articles", compressedCount)
	}

	return nil
}

// compressContent compresses text content using gzip
func compressContent(content string) ([]byte, error) {
	if content == "" {
		return nil, nil
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)

	if _, err := gw.Write([]byte(content)); err != nil {
		return nil, fmt.Errorf("failed to compress content: %v", err)
	}

	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}

	return buf.Bytes(), nil
}

// decompressContent decompresses gzipped content
func decompressContent(compressed []byte) (string, error) {
	if len(compressed) == 0 {
		return "", nil
	}

	gr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		return "", fmt.Errorf("failed to decompress content: %v", err)
	}

	return string(decompressed), nil
}

// cleanAndOptimizeContent cleans and optimizes article content for storage
func cleanAndOptimizeContent(content string) string {
	if content == "" {
		return content
	}

	// Only remove excessive whitespace at the beginning and end
	content = strings.TrimSpace(content)

	// Remove excessive consecutive newlines (more than 3)
	content = regexp.MustCompile(`\n{4,}`).ReplaceAllString(content, "\n\n\n")

	// Remove excessive spaces (more than 2 consecutive spaces)
	content = regexp.MustCompile(` {3,}`).ReplaceAllString(content, "  ")

	return content
}

// updateSearchIndexWithTx updates the search index for an article using a transaction
func (s *SQLiteStorage) updateSearchIndexWithTx(tx *sql.Tx, articleID string, title, description, content, author, source string) error {
	// Delete existing search index entries for this article
	if _, err := tx.Exec("DELETE FROM search_index WHERE article_id = ?", articleID); err != nil {
		log.Printf("Warning: failed to delete existing search index: %v", err)
		return err
	}

	// Extract and insert search terms for each field
	fields := map[string]string{
		"title":       title,
		"description": description,
		"content":     content,
		"author":      author,
		"source":      source,
	}

	for fieldType, text := range fields {
		if text == "" {
			continue
		}

		// Extract search terms from text
		terms := s.extractSearchTerms(text)
		for lang, termList := range terms {
			for _, term := range termList {
				if len(term) > 2 { // Only index meaningful terms
					_, err := tx.Exec(`
						INSERT INTO search_index (article_id, search_term, field_type, language)
						VALUES (?, ?, ?, ?)
					`, articleID, strings.ToLower(term), fieldType, lang)
					if err != nil {
						log.Printf("Warning: failed to insert search term %s for article %s: %v", term, articleID, err)
					}
				}
			}
		}
	}

	return nil
}

// updateSearchIndex updates the search index for an article
func (s *SQLiteStorage) updateSearchIndex(articleID string, title, description, content, author, source string) error {
	// Remove mutex lock since this is called from within SaveFeed which already has the lock
	// s.mutex.Lock()
	// defer s.mutex.Unlock()

	// Delete existing search index entries for this article
	if _, err := s.db.Exec("DELETE FROM search_index WHERE article_id = ?", articleID); err != nil {
		log.Printf("Warning: failed to delete existing search index: %v", err)
		return err
	}

	// Extract and insert search terms for each field
	fields := map[string]string{
		"title":       title,
		"description": description,
		"content":     content,
		"author":      author,
		"source":      source,
	}

	for fieldType, text := range fields {
		if text == "" {
			continue
		}

		// Extract search terms from text
		terms := s.extractSearchTerms(text)
		for lang, termList := range terms {
			for _, term := range termList {
				if len(term) > 2 { // Only index meaningful terms
					_, err := s.db.Exec(`
						INSERT INTO search_index (article_id, search_term, field_type, language)
						VALUES (?, ?, ?, ?)
					`, articleID, strings.ToLower(term), fieldType, lang)
					if err != nil {
						log.Printf("Warning: failed to insert search term %s for article %s: %v", term, articleID, err)
					}
				}
			}
		}
	}

	return nil
}

// detectLanguage detects the language of the given text using the lingua-go library
func (s *SQLiteStorage) detectLanguage(text string) string {
	if text == "" {
		return "en" // Default to English
	}

	// Use the proper language detection library
	language, exists := s.detector.DetectLanguageOf(text)
	if !exists {
		return "en" // Default to English if detection fails
	}

	// Convert lingua language to our language codes
	switch language {
	case lingua.English:
		return "en"
	case lingua.German:
		return "de"
	case lingua.French:
		return "fr"
	case lingua.Spanish:
		return "es"
	case lingua.Chinese:
		return "zh"
	case lingua.Russian:
		return "ru"
	case lingua.Italian:
		return "it"
	case lingua.Portuguese:
		return "pt"
	case lingua.Dutch:
		return "nl"
	case lingua.Swedish:
		return "sv"
	case lingua.Danish:
		return "da"
	case lingua.Finnish:
		return "fi"
	case lingua.Polish:
		return "pl"
	case lingua.Czech:
		return "cs"
	case lingua.Hungarian:
		return "hu"
	case lingua.Romanian:
		return "ro"
	default:
		return "en" // Default to English for unsupported languages
	}
}

// getStopWords returns stop words for the given language
func getStopWords(language string) map[string]bool {
	switch language {
	case "zh":
		// Chinese stop words (simplified)
		return map[string]bool{
			"的": true, "了": true, "在": true, "是": true, "我": true, "有": true, "和": true, "就": true,
			"不": true, "人": true, "都": true, "一": true, "一个": true, "上": true, "也": true, "很": true,
			"到": true, "说": true, "要": true, "去": true, "你": true, "会": true, "着": true, "没有": true,
			"看": true, "好": true, "自己": true, "这": true, "那": true, "他": true, "她": true, "它": true,
		}
	case "ru":
		// Russian stop words
		return map[string]bool{
			"и": true, "в": true, "во": true, "не": true, "что": true, "он": true, "на": true, "я": true,
			"с": true, "со": true, "как": true, "а": true, "то": true, "все": true, "она": true, "так": true,
			"его": true, "но": true, "да": true, "ты": true, "к": true, "у": true, "же": true, "вы": true,
			"за": true, "бы": true, "по": true, "только": true, "ее": true, "мне": true, "было": true,
		}
	case "de":
		// German stop words
		return map[string]bool{
			"der": true, "die": true, "das": true, "und": true, "in": true, "den": true, "von": true, "zu": true,
			"mit": true, "sich": true, "des": true, "auf": true, "für": true, "ist": true, "im": true, "dem": true,
			"nicht": true, "ein": true, "eine": true, "als": true, "auch": true, "es": true, "an": true, "werden": true,
			"aus": true, "er": true, "hat": true, "daß": true, "sie": true, "nach": true, "wird": true, "bei": true,
		}
	case "fr":
		// French stop words
		return map[string]bool{
			"le": true, "la": true, "de": true, "un": true, "une": true, "et": true, "à": true, "être": true,
			"en": true, "avoir": true, "que": true, "pour": true, "dans": true, "ce": true, "il": true,
			"qui": true, "ne": true, "sur": true, "se": true, "pas": true, "plus": true, "pouvoir": true, "par": true,
			"je": true, "avec": true, "tout": true, "faire": true, "son": true, "mettre": true, "autre": true,
		}
	case "es":
		// Spanish stop words
		return map[string]bool{
			"el": true, "la": true, "de": true, "que": true, "y": true, "a": true, "los": true, "se": true,
			"del": true, "las": true, "un": true, "por": true, "con": true, "no": true, "una": true, "su": true,
			"para": true, "es": true, "al": true, "lo": true, "como": true, "más": true, "o": true, "pero": true,
			"sus": true, "le": true, "ha": true, "me": true, "si": true, "sin": true, "sobre": true, "este": true,
		}
	case "it":
		// Italian stop words
		return map[string]bool{
			"il": true, "la": true, "di": true, "da": true, "in": true, "con": true, "su": true, "per": true,
			"tra": true, "fra": true, "lo": true, "gli": true, "le": true, "un": true, "una": true,
			"e": true, "o": true, "ma": true, "se": true, "che": true, "come": true, "dove": true, "quando": true,
			"perché": true, "anche": true, "solo": true, "sempre": true, "mai": true, "già": true, "ancora": true,
		}
	case "pt":
		// Portuguese stop words
		return map[string]bool{
			"o": true, "a": true, "os": true, "as": true, "um": true, "uma": true, "e": true, "é": true,
			"de": true, "do": true, "da": true, "em": true, "para": true, "com": true, "não": true,
			"se": true, "na": true, "por": true, "mais": true, "como": true,
			"mas": true, "foi": true, "ele": true, "das": true, "tem": true, "à": true, "seu": true, "sua": true,
		}
	default:
		// English stop words (default)
		return map[string]bool{
			"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
			"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
			"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
			"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
			"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
			"should": true, "may": true, "might": true, "can": true, "this": true, "that": true,
			"these": true, "those": true, "i": true, "you": true, "he": true, "she": true,
			"it": true, "we": true, "they": true, "me": true, "him": true, "her": true,
			"us": true, "them": true, "my": true, "your": true, "his": true, "its": true,
			"our": true, "their": true, "mine": true, "yours": true, "hers": true,
			"ours": true, "theirs": true, "from": true, "as": true, "if": true, "then": true,
			"else": true, "when": true, "where": true, "why": true, "how": true, "all": true,
			"any": true, "both": true, "each": true, "few": true, "more": true, "most": true,
			"other": true, "some": true, "such": true, "no": true, "nor": true, "not": true,
			"only": true, "own": true, "same": true, "so": true, "than": true, "too": true,
			"very": true, "just": true, "now": true, "here": true, "there": true, "up": true,
			"down": true, "out": true, "off": true, "over": true, "under": true, "again": true,
			"further": true, "once": true, "about": true, "against": true,
			"between": true, "into": true, "through": true, "during": true, "before": true,
			"after": true, "above": true, "below": true, "onto": true, "upon": true, "within": true, "without": true,
			"among": true, "throughout": true, "toward": true, "towards": true,
		}
	}
}

// extractSearchTerms extracts meaningful search terms from text with language support
func (s *SQLiteStorage) extractSearchTerms(text string) map[string][]string {
	if text == "" {
		return make(map[string][]string)
	}

	// Detect language using the proper library
	language := s.detectLanguage(text)

	// Get language-specific stop words
	stopWords := getStopWords(language)

	// Language-specific text processing
	var terms []string

	switch language {
	case "zh":
		// Chinese text processing - split by characters and common patterns
		// Remove HTML tags first
		text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, " ")

		// Split Chinese text into meaningful chunks
		// This is a simplified approach - for production, consider using proper Chinese segmentation
		chinesePattern := regexp.MustCompile(`[\p{Han}]+`)
		matches := chinesePattern.FindAllString(text, -1)

		for _, match := range matches {
			if len(match) >= 2 && !stopWords[match] {
				terms = append(terms, match)
			}
		}

		// Also extract Latin words from Chinese text
		latinPattern := regexp.MustCompile(`[a-zA-Z]+`)
		latinMatches := latinPattern.FindAllString(text, -1)
		for _, match := range latinMatches {
			match = strings.ToLower(match)
			if len(match) > 2 && !stopWords[match] {
				terms = append(terms, match)
			}
		}

	default:
		// Latin-based languages (English, French, German, Spanish, etc.)
		// Convert to lowercase and remove HTML tags
		text = strings.ToLower(text)
		text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, " ")

		// Remove special characters but keep spaces
		text = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(text, " ")

		// Split into words
		words := strings.Fields(text)

		for _, word := range words {
			word = strings.TrimSpace(word)
			if len(word) > 2 && !stopWords[word] {
				terms = append(terms, word)
			}
		}
	}

	// Limit terms to avoid excessive storage
	if len(terms) > 100 {
		terms = terms[:100]
	}

	return map[string][]string{
		language: terms,
	}
}

// searchArticlesIndex performs full-text search using the search index table with multi-language support
func (s *SQLiteStorage) searchArticlesIndex(searchTerms []string, topicID int) ([]string, error) {
	if len(searchTerms) == 0 {
		return nil, nil
	}

	// Detect language of search terms
	searchText := strings.Join(searchTerms, " ")
	searchLanguage := s.detectLanguage(searchText)

	// Use a more secure approach with individual OR conditions instead of IN clause
	// This avoids SQL string concatenation while maintaining functionality
	var articleIDs []string

	// Try language-specific search first
	for _, term := range searchTerms {
		query := `
			SELECT DISTINCT a.article_id 
			FROM articles a
			JOIN search_index si ON a.article_id = si.article_id
			WHERE a.topic_id = ? AND si.language = ? AND si.search_term = ?
			ORDER BY a.published_at DESC
		`

		rows, err := s.db.Query(query, topicID, searchLanguage, strings.ToLower(term))
		if err != nil {
			return nil, fmt.Errorf("failed to search index: %v", err)
		}

		for rows.Next() {
			var articleID string
			if err := rows.Scan(&articleID); err != nil {
				if closeErr := rows.Close(); closeErr != nil {
					log.Printf("Warning: failed to close rows: %v", closeErr)
				}
				return nil, fmt.Errorf("failed to scan search result: %v", err)
			}
			articleIDs = append(articleIDs, articleID)
		}
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}

	// If no results found in the same language, try cross-language search
	if len(articleIDs) == 0 {
		for _, term := range searchTerms {
			query := `
				SELECT DISTINCT a.article_id 
				FROM articles a
				JOIN search_index si ON a.article_id = si.article_id
				WHERE a.topic_id = ? AND si.search_term = ?
				ORDER BY a.published_at DESC
			`

			rows, err := s.db.Query(query, topicID, strings.ToLower(term))
			if err != nil {
				return nil, fmt.Errorf("failed to search index cross-language: %v", err)
			}

			for rows.Next() {
				var articleID string
				if err := rows.Scan(&articleID); err != nil {
					if closeErr := rows.Close(); closeErr != nil {
						log.Printf("Warning: failed to close rows: %v", closeErr)
					}
					return nil, fmt.Errorf("failed to scan search result: %v", err)
				}
				articleIDs = append(articleIDs, articleID)
			}
			if err := rows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}

	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	var uniqueArticleIDs []string
	for _, id := range articleIDs {
		if !seen[id] {
			seen[id] = true
			uniqueArticleIDs = append(uniqueArticleIDs, id)
		}
	}

	return uniqueArticleIDs, nil
}

// GetFeedStats returns detailed statistics for each feed
func (s *SQLiteStorage) GetFeedStats() (map[string]interface{}, error) {
	query := `
		SELECT 
			source,
			COUNT(*) as article_count,
			AVG(LENGTH(content) + LENGTH(title) + LENGTH(description)) as avg_content_size,
			SUM(LENGTH(content) + LENGTH(title) + LENGTH(description)) as total_content_size,
			COUNT(CASE WHEN language != 'en' THEN 1 END) as non_english_count,
			MIN(published_at) as oldest_article,
			MAX(published_at) as newest_article
		FROM articles 
		GROUP BY source
		ORDER BY article_count DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query feed stats: %v", err)
	}
	defer rows.Close()

	var feeds []map[string]interface{}
	for rows.Next() {
		var source string
		var articleCount int
		var avgContentSize sql.NullFloat64
		var totalContentSize sql.NullFloat64
		var nonEnglishCount int
		var oldestArticle sql.NullString
		var newestArticle sql.NullString

		err := rows.Scan(&source, &articleCount, &avgContentSize, &totalContentSize, &nonEnglishCount, &oldestArticle, &newestArticle)
		if err != nil {
			log.Printf("Warning: failed to scan feed stat: %v", err)
			continue
		}

		feed := map[string]interface{}{
			"source":            source,
			"article_count":     articleCount,
			"non_english_count": nonEnglishCount,
		}

		if avgContentSize.Valid {
			feed["avg_content_size"] = int(avgContentSize.Float64)
		}
		if totalContentSize.Valid {
			feed["total_content_size"] = int(totalContentSize.Float64)
		}
		if oldestArticle.Valid {
			feed["oldest_article"] = oldestArticle.String
		}
		if newestArticle.Valid {
			feed["newest_article"] = newestArticle.String
		}

		feeds = append(feeds, feed)
	}

	return map[string]interface{}{
		"feeds": feeds,
	}, nil
}

// SaveArticles saves articles independently of topics (new approach)
func (s *SQLiteStorage) SaveArticles(articles []models.Article) error {
	if len(articles) == 0 {
		return nil
	}

	log.Printf("SaveArticles: Starting to save %d articles independently", len(articles))

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Use INSERT OR REPLACE to handle duplicates gracefully
	// Set topic_id to 1 temporarily (will be updated later by AssignArticlesToTopic)
	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO articles (article_id, topic_id, title, link, description, content, author, source, categories, published_at, language)
		VALUES (?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	successCount := 0
	for _, article := range articles {
		categoriesJSON, _ := json.Marshal(article.Categories)
		content := cleanAndOptimizeContent(article.Content)
		articleLanguage := s.detectLanguage(article.Title + " " + article.Description + " " + article.Content)

		_, err = stmt.Exec(article.ID, article.Title, article.Link, article.Description, content, article.Author, article.Source, categoriesJSON, article.PublishedAt, articleLanguage)
		if err != nil {
			log.Printf("Warning: failed to insert article %s: %v", article.ID, err)
			continue // Continue with other articles instead of failing completely
		}

		// Update search index
		if err := s.updateSearchIndexWithTx(tx, article.ID, article.Title, article.Description, content, article.Author, article.Source); err != nil {
			log.Printf("Warning: failed to update search index for article %s: %v", article.ID, err)
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("SaveArticles: Successfully saved %d/%d articles", successCount, len(articles))
	return nil
}

// AssignArticlesToTopic assigns articles to a topic after they're stored
func (s *SQLiteStorage) AssignArticlesToTopic(articleIDs []string, topic string) error {
	if len(articleIDs) == 0 {
		return nil
	}

	log.Printf("AssignArticlesToTopic: Assigning %d articles to topic '%s'", len(articleIDs), topic)

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

	// Use the new article_topics table for many-to-many relationships
	stmt, err := tx.Prepare("INSERT OR IGNORE INTO article_topics (article_id, topic_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	// Also update the legacy topic_id column for backward compatibility
	legacyStmt, err := tx.Prepare("UPDATE articles SET topic_id = ? WHERE article_id = ? AND topic_id = 1")
	if err != nil {
		return fmt.Errorf("failed to prepare legacy update statement: %v", err)
	}
	defer legacyStmt.Close()

	successCount := 0
	for _, articleID := range articleIDs {
		// Insert into membership table
		_, err = stmt.Exec(articleID, topicID)
		if err != nil {
			log.Printf("Warning: failed to assign article %s to topic %s: %v", articleID, topic, err)
			continue
		}

		// Update legacy column only if it's still set to the default (1)
		_, _ = legacyStmt.Exec(topicID, articleID)

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("AssignArticlesToTopic: Successfully assigned %d/%d articles to topic '%s'", successCount, len(articleIDs), topic)
	return nil
}

// AddArticleToTopic adds a single article to a topic
func (s *SQLiteStorage) AddArticleToTopic(articleID string, topic string) error {
	return s.AssignArticlesToTopic([]string{articleID}, topic)
}

// RemoveArticleFromTopic removes an article from a topic
func (s *SQLiteStorage) RemoveArticleFromTopic(articleID string, topic string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get topic ID
	var topicID int
	err = tx.QueryRow("SELECT id FROM topics WHERE name = ?", topic).Scan(&topicID)
	if err != nil {
		return fmt.Errorf("topic not found: %v", err)
	}

	// Remove from membership table
	_, err = tx.Exec("DELETE FROM article_topics WHERE article_id = ? AND topic_id = ?", articleID, topicID)
	if err != nil {
		return fmt.Errorf("failed to remove article from topic: %v", err)
	}

	return tx.Commit()
}

// GetArticleTopics returns all topics for an article
func (s *SQLiteStorage) GetArticleTopics(articleID string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT t.name 
		FROM topics t 
		JOIN article_topics at ON t.id = at.topic_id 
		WHERE at.article_id = ?
		ORDER BY t.name
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query article topics: %v", err)
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

// GetTopicArticles returns articles for a topic using the membership table
func (s *SQLiteStorage) GetTopicArticles(topic string, query *models.ODataQuery) ([]models.Article, int, error) {
	// This method uses the new article_topics table instead of the legacy topic_id column
	baseQuery := `
		SELECT 
			a.article_id, 
			a.title, 
			a.description, 
			a.content, 
			a.link, 
			a.author, 
			a.source, 
			a.published_at, 
			a.categories, 
			a.language,
			cc.compressed_content
		FROM articles a
		JOIN article_topics at ON a.article_id = at.article_id
		JOIN topics t ON at.topic_id = t.id
		LEFT JOIN compressed_content cc ON a.article_id = cc.article_id
		WHERE t.name = ?
	`

	// Count query for pagination
	countQuery := `
		SELECT COUNT(DISTINCT a.article_id)
		FROM articles a
		JOIN article_topics at ON a.article_id = at.article_id
		JOIN topics t ON at.topic_id = t.id
		WHERE t.name = ?
	`

	args := []interface{}{topic}
	countArgs := []interface{}{topic}

	// Add search conditions if specified
	if len(query.Search) > 0 {
		searchConditions := make([]string, len(query.Search))
		for i, term := range query.Search {
			searchConditions[i] = "(a.title LIKE ? OR a.description LIKE ? OR a.content LIKE ?)"
			searchTerm := "%" + term + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
			countArgs = append(countArgs, searchTerm, searchTerm, searchTerm)
		}
		searchClause := " AND (" + strings.Join(searchConditions, " OR ") + ")"
		baseQuery += searchClause
		countQuery += searchClause
	}

	// Get total count
	var totalCount int
	err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count articles: %v", err)
	}

	// Add ordering
	if query.OrderBy != "" {
		if strings.Contains(strings.ToLower(query.OrderBy), "desc") {
			baseQuery += " ORDER BY a.published_at DESC"
		} else {
			baseQuery += " ORDER BY a.published_at ASC"
		}
	} else {
		baseQuery += " ORDER BY a.published_at DESC" // Default ordering
	}

	// Add pagination - LIMIT must come before OFFSET in SQLite
	if query.Top > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, query.Top)
		if query.Skip > 0 {
			baseQuery += " OFFSET ?"
			args = append(args, query.Skip)
		}
	} else if query.Skip > 0 {
		// If only skip is specified, we need a default limit
		baseQuery += " LIMIT -1 OFFSET ?"
		args = append(args, query.Skip)
	}

	// Execute query
	rows, err := s.db.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query topic articles: %v", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var compressedContent []byte
		var categoriesJSON string

		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Description,
			&article.Content,
			&article.Link,
			&article.Author,
			&article.Source,
			&article.PublishedAt,
			&categoriesJSON,
			&article.Language,
			&compressedContent,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan article: %v", err)
		}

		// Handle compressed content
		if len(compressedContent) > 0 && article.Content == "" {
			if decompressed, err := decompressContent(compressedContent); err == nil {
				article.Content = decompressed
			}
		}

		// Parse categories JSON
		if categoriesJSON != "" {
			json.Unmarshal([]byte(categoriesJSON), &article.Categories)
		}

		articles = append(articles, article)
	}

	return articles, totalCount, nil
}

// GetCombinedFilters combines filters from multiple topics
func (s *SQLiteStorage) GetCombinedFilters(topics []string) ([]string, bool) {
	// This method would ideally read topic configurations from database
	// For now, we'll implement the logic in the aggregator layer
	return nil, false
}
