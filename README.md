# GoRSSag - RSS Aggregator

A Go-based RSS aggregator service that combines multiple RSS feeds per topic and provides a REST API with comprehensive OData support for filtering, searching, and querying articles.

## Features

- **RSS Feed Aggregation**: Combine multiple RSS feeds per topic
- **Topic-Level Filtering**: Filter articles at the RSS source level using full-text terms
- **JSON API**: RESTful API with JSON responses
- **Advanced OData Support**: Full OData query capabilities including filtering, searching, sorting, and pagination
- **Global Search**: Search across all article fields with OR logic
- **Field Selection**: Choose which fields to return using `$select`
- **Persistent Storage**: SQLite database with optimized indexing for fast queries
- **Memory Caching**: Hot data cached in memory for fast access
- **Background RSS Polling**: Continuous feed updates to ensure data freshness
- **Parallel Feed Fetching**: Concurrent RSS feed retrieval for optimal performance
- **Modern Web Interface**: Single Page Application (SPA) for user-friendly browsing
- **Interactive API Documentation**: Swagger UI for developer testing
- **Production Security**: Rate limiting, input validation, security headers, and CORS protection
- **Docker Support**: Containerized deployment with Docker and Docker Compose
- **Environment Configuration**: All settings configurable via environment variables

## Quick Start

### Using Docker Compose

```bash
# Clone the repository
git clone <repository-url>
cd gorssag

# Start the service
docker-compose up -d

# Check if it's running
curl http://localhost:8080/health
```

### Using Docker

```bash
# Build the image
docker build -t gorssag .

# Run with custom configuration
docker run -p 8080:8080 \
  -e PORT=8080 \
  -e CACHE_TTL=30m \
  -e DATA_DIR=/app/data \
  -e POLL_INTERVAL=15m \
  -e FEED_TOPIC_TECH="https://feeds.feedburner.com/TechCrunch,https://rss.cnn.com/rss/edition_technology.rss" \
  -e FEED_TOPIC_NEWS="https://feeds.bbci.co.uk/news/rss.xml,https://rss.cnn.com/rss/edition.rss" \
  -v rss_data:/app/data \
  gorssag
```

### Local Development

```bash
# Download dependencies
go mod download

# Run the application
go run main.go
```

## Configuration

All configuration is done via environment variables:

### Server Configuration
- `PORT`: Server port (default: 8080)
- `CACHE_TTL`: Cache time-to-live for hot data (default: 15m)
- `DATA_DIR`: Directory for persistent storage (default: ./data)
- `LOG_LEVEL`: Logging level (default: info)
- `POLL_INTERVAL`: Background polling interval (default: 15m)

### Web Interface Configuration
- `ENABLE_SPA`: Enable the Single Page Application interface (default: true)
- `ENABLE_SWAGGER`: Enable the Swagger API documentation (default: true)

### RSS Feed Configuration
Configure feeds using `FEED_TOPIC_*` environment variables with optional filtering:

```bash
# Format: FEED_TOPIC_<TOPIC_NAME>=url1,url2,url3|filter1,filter2,filter3
# If no filters specified: FEED_TOPIC_<TOPIC_NAME>=url1,url2,url3

# Example with filters (only articles containing these terms will be stored)
FEED_TOPIC_TECH=https://feeds.feedburner.com/TechCrunch,https://rss.cnn.com/rss/edition_technology.rss|AI,artificial intelligence,machine learning,blockchain

# Example without filters (all articles will be stored)
FEED_TOPIC_NEWS=https://feeds.bbci.co.uk/news/rss.xml,https://rss.cnn.com/rss/edition.rss

# Multiple filters (OR logic - articles matching any filter term will be included)
FEED_TOPIC_PROGRAMMING=https://blog.golang.org/feed.atom,https://feeds.feedburner.com/oreilly/go|Go,golang,programming,development
```

### Topic-Level Filtering
Each topic can be configured with full-text filters that are applied at the RSS source level. Only articles containing the specified terms will be stored and served.

**Filter Features:**
- **Full-Text Search**: Filters search across title, description, content, author, and categories
- **OR Logic**: Articles matching any filter term are included
- **Case Insensitive**: Filter matching is case-insensitive
- **Optional**: Topics can be configured without filters to include all articles
- **Real-Time**: Filters are applied during RSS polling and storage

**Example Use Cases:**
- **Tech Topic**: Filter for "AI", "machine learning", "blockchain" to focus on emerging technologies
- **News Topic**: Filter for "technology", "innovation" to get tech-focused news
- **Programming Topic**: Filter for "Go", "golang", "programming" to focus on development content

### Security Configuration
The RSS Aggregator includes comprehensive security protections for production environments:

```bash
# Rate Limiting
ENABLE_RATE_LIMIT=true              # Enable rate limiting (default: true)
RATE_LIMIT_PER_SECOND=10.0          # Requests per second per IP (default: 10.0)
RATE_LIMIT_BURST=20                 # Burst limit per IP (default: 20)

# CORS Configuration
ENABLE_CORS=true                    # Enable CORS protection (default: true)
ALLOWED_ORIGINS=*                   # Comma-separated list of allowed origins

# Security Headers
ENABLE_SECURITY_HEADERS=true        # Enable security headers (default: true)
MAX_REQUEST_SIZE=10485760           # Maximum request size in bytes (default: 10MB)
ENABLE_REQUEST_ID=true              # Enable request ID tracking (default: true)
```

**Security Features:**
- **Rate Limiting**: Per-IP rate limiting to prevent abuse
- **Input Validation**: OData query parameter validation and sanitization
- **Security Headers**: XSS protection, content type sniffing prevention, frame denial
- **CORS Protection**: Configurable cross-origin resource sharing
- **Request Size Limits**: Protection against large payload attacks
- **Request Tracking**: Unique request IDs for monitoring and debugging
- **IP Address Detection**: Proper handling of forwarded headers behind proxies

## API Endpoints

### Health Check
```
GET /health
```

### Get Available Topics
```
GET /api/v1/topics
```

### Get Aggregated Feed
```
GET /api/v1/feeds/{topic}
```

### Get Feed Information
```
GET /api/v1/feeds/{topic}/info
```

### Refresh Feed
```
POST /api/v1/feeds/{topic}/refresh
```

### Poller Control Endpoints

#### Get Poller Status
```
GET /api/v1/poller/status
```

#### Force Poll Topic
```
POST /api/v1/poller/force-poll/{topic}
```

#### Get Last Polled Times
```
GET /api/v1/poller/last-polled
```

## OData Query Capabilities

### Filtering (`$filter`)
- **Comparison Operators**: `eq`, `ne`, `gt`, `ge`, `lt`, `le`
- **String Functions**: `startswith()`, `endswith()`, `contains()`
- **Logical Operators**: `and`, `or`
- **Supported Fields**: `title`, `description`, `content`, `author`, `source`, `published_at`

### Global Search (`$search`)
- Search across all article fields with OR logic
- Multiple comma-separated terms
- Searches: title, description, content, author, source, categories

### Sorting (`$orderby`)
- Sort by: `title`, `author`, `source`, `published_at`
- Directions: `asc`, `desc`

### Field Selection (`$select`)
- Select specific fields to return in the response
- Comma-separated field names
- Supported fields: `title`, `link`, `description`, `content`, `author`, `source`, `categories`, `published_at`
- If not specified, all fields are returned (default behavior)

### Pagination
- `$top`: Limit results
- `$skip`: Skip results

#### Examples

```bash
# Advanced filtering with logical operators
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=startswith(title, 'AI') and published_at gt '2023-01-01T00:00:00Z'"

# Global search across all fields
curl "http://localhost:8080/api/v1/feeds/tech?\$search=AI,machine learning,artificial intelligence"

# Select specific fields only
curl "http://localhost:8080/api/v1/feeds/tech?\$select=title,author,published_at"

# Complex query combining filter, search, pagination, and field selection
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=contains(description, 'technology')&\$search=AI&\$orderby=published_at desc&\$top=10&\$select=title,link,author"

# Get feed information
curl http://localhost:8080/api/v1/feeds/tech/info

# Refresh feed data
curl -X POST http://localhost:8080/api/v1/feeds/tech/refresh

# Check poller status
curl http://localhost:8080/api/v1/poller/status

# Force poll a specific topic
curl -X POST http://localhost:8080/api/v1/poller/force-poll/tech

# Get last polled times for all topics
curl http://localhost:8080/api/v1/poller/last-polled
```

## Performance Features

### Background RSS Polling
- **Continuous Fetching**: RSS feeds are automatically polled at configurable intervals
- **Request Minimization**: Reduces external requests by pre-fetching data
- **Configurable Intervals**: Set polling frequency via `POLL_INTERVAL` environment variable
- **Smart Polling**: Avoids polling the same topic too frequently (minimum 5-minute gap)
- **Error Resilience**: Individual feed failures don't affect other feeds
- **Force Polling**: Manual trigger to refresh specific topics via API

### Parallel Feed Fetching
- **Concurrent RSS Polling**: All RSS feeds are fetched simultaneously using goroutines
- **Timeout Protection**: 30-second timeout prevents hanging on slow feeds
- **Error Resilience**: Individual feed failures don't affect other feeds
- **Latest Articles**: Ensures we get the most recent articles from all sources

### Data Storage
The application uses a two-tier storage approach:

1. **Persistent Storage**: RSS feeds are stored in SQLite database with optimized indexing
2. **Memory Cache**: Hot data is cached in memory for fast access

### SQLite Storage
- **Optimized for OData**: Full SQL query support with proper indexing
- **Full-Text Search**: Indexed text search across all fields using LIKE queries
- **Indexed Queries**: B-tree indexes on common query fields (topic_id, published_at, author, source)
- **Composite Indexes**: Optimized for complex queries (topic + date, author + source)
- **Transaction Support**: ACID compliance for data integrity
- **Performance**: 10-100x faster than in-memory filtering for complex queries

### Storage Benefits
- **Persistence**: Data survives container restarts
- **Performance**: Hot data served from memory, optimized queries from SQLite
- **Scalability**: Can handle large amounts of RSS data efficiently
- **Reliability**: ACID transactions ensure data integrity
- **Efficiency**: Only loads required data, not entire datasets

## Project Structure

```
gorssag/
├── main.go                 # Application entry point
├── go.mod                  # Go module file
├── Dockerfile              # Docker configuration
├── docker-compose.yml      # Docker Compose configuration
├── README.md               # Comprehensive documentation
├── API.md                  # Detailed API documentation
├── .github/workflows/      # GitHub Actions workflows
└── internal/
    ├── api/                # HTTP API handlers
    ├── aggregator/         # RSS feed aggregation logic (parallel fetching)
    ├── cache/              # Memory caching layer
    ├── config/             # Environment-based configuration
    ├── models/             # Data models
    ├── odata/              # OData parser and evaluator
    ├── poller/             # Background RSS polling system
    └── storage/            # Persistent storage layer
```

## Development

### Prerequisites

- Go 1.21 or later
- Docker (optional)
- Docker Compose (optional)

### Building

```bash
go build -o gorssag .
```

### Running Tests

```bash
go test -v ./...
```

## Docker Pipeline

The project includes GitHub Actions workflows for:

1. **Testing**: Runs tests on every push and pull request
2. **Docker Build**: Builds and pushes Docker images to GitHub Container Registry

### Manual Docker Build

```bash
# Build image
docker build -t gorssag .

# Run container with custom feeds and polling
docker run -p 8080:8080 \
  -e POLL_INTERVAL=10m \
  -e FEED_TOPIC_MYCUSTOM="https://example.com/feed1,https://example.com/feed2" \
  -v $(pwd)/data:/app/data \
  gorssag
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License. 

## Web Interfaces

### Single Page Application (SPA)
The RSS Aggregator includes a modern, responsive web interface for browsing and searching RSS feeds.

**Features:**
- **Topic Browser**: View all available RSS topics with article counts
- **Article Search**: Search across all articles with advanced filtering
- **Real-time Updates**: Refresh feeds and view latest articles
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Modern UI**: Clean, intuitive interface with smooth animations

**Access:** `http://localhost:8080/`

### Swagger API Documentation
Interactive API documentation for developers to test and explore the RSS Aggregator API.

**Features:**
- **Interactive Testing**: Test API endpoints directly from the browser
- **Request/Response Examples**: See example requests and responses
- **Parameter Documentation**: Detailed parameter descriptions and types
- **Authentication Support**: Test with API keys if configured
- **Export Capabilities**: Export API specifications

**Access:** `http://localhost:8080/swagger/`

### Disabling Interfaces
Both interfaces can be disabled via environment variables:

```bash
# Disable SPA interface
ENABLE_SPA=false

# Disable Swagger documentation
ENABLE_SWAGGER=false

# Disable both (API-only mode)
ENABLE_SPA=false
ENABLE_SWAGGER=false
``` 