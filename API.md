# GoRSSag API Documentation

This document provides detailed information about the GoRSSag RSS Aggregator API endpoints and their usage.

## Base URL

All API endpoints are prefixed with `/api/v1/`.

## Health Check

### GET /health

Returns the health status of the service.

**Response:**
```json
{
  "status": "healthy",
  "service": "rss-aggregator",
  "poller_active": true
}
```

## Topics

### GET /api/v1/topics

Returns a list of available RSS feed topics.

**Response:**
```json
{
  "topics": ["tech", "news", "programming"],
  "count": 3
}
```

## Feeds

### GET /api/v1/feeds/{topic}

Returns aggregated RSS feed data for the specified topic.

**Parameters:**
- `topic` (path): The topic name (e.g., "tech", "news")

**Query Parameters (OData):**
- `$filter`: Filter expression
- `$search`: Search terms (comma-separated)
- `$orderby`: Sort expression
- `$top`: Limit number of results
- `$skip`: Skip number of results
- `$select`: Select specific fields

**Example:**
```bash
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=contains(title, 'AI')&\$top=10"
```

**Response:**
```json
{
  "topic": "tech",
  "articles": [
    {
      "title": "AI Breakthrough in Machine Learning",
      "link": "https://example.com/ai-article",
      "description": "New developments in AI technology",
      "content": "Full article content...",
      "author": "John Doe",
      "source": "Tech News",
      "categories": ["AI", "Technology"],
      "published_at": "2023-01-15T10:30:00Z"
    }
  ],
  "count": 1,
  "updated": "2023-01-15T10:35:00Z"
}
```

### GET /api/v1/feeds/{topic}/info

Returns metadata about the specified feed topic.

**Parameters:**
- `topic` (path): The topic name

**Response:**
```json
{
  "topic": "tech",
  "file_size": 1024,
  "last_modified": "2023-01-15T10:35:00Z",
  "article_count": 25
}
```

### POST /api/v1/feeds/{topic}/refresh

Forces a refresh of the specified feed topic.

**Parameters:**
- `topic` (path): The topic name

**Response:**
```json
{
  "message": "Feed refreshed successfully",
  "topic": "tech"
}
```

## Poller Control

### GET /api/v1/poller/status

Returns the current status of the background RSS poller.

**Response:**
```json
{
  "is_polling": true,
  "status": "active"
}
```

### POST /api/v1/poller/force-poll/{topic}

Forces an immediate poll of the specified topic, bypassing the normal polling schedule.

**Parameters:**
- `topic` (path): The topic name to force poll

**Response:**
```json
{
  "message": "Force poll initiated successfully",
  "topic": "tech"
}
```

**Error Response (404):**
```json
{
  "error": "topic 'invalid-topic' not found"
}
```

### GET /api/v1/poller/last-polled

Returns the last polled time for all available topics.

**Response:**
```json
{
  "last_polled": {
    "tech": "2023-01-15T10:30:00Z",
    "news": "2023-01-15T10:25:00Z",
    "programming": "2023-01-15T10:20:00Z"
  }
}
```

## OData Filtering

The API supports OData query parameters for advanced filtering and querying.

### $filter Parameter

Supports complex filtering expressions with comparison operators, string functions, and logical operators.

#### Comparison Operators
- `eq`: Equal
- `ne`: Not equal
- `gt`: Greater than
- `ge`: Greater than or equal
- `lt`: Less than
- `le`: Less than or equal

#### String Functions
- `startswith(field, value)`: Check if field starts with value
- `endswith(field, value)`: Check if field ends with value
- `contains(field, value)`: Check if field contains value

#### Logical Operators
- `and`: Logical AND
- `or`: Logical OR

#### Supported Fields
- `title`: Article title
- `description`: Article description
- `content`: Article content
- `author`: Article author
- `source`: RSS feed source
- `published_at`: Publication date

#### Examples

```bash
# Simple equality filter
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=author eq 'John Doe'"

# String function filter
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=startswith(title, 'AI')"

# Complex logical filter
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=contains(title, 'AI') and published_at gt '2023-01-01T00:00:00Z'"

# OR operator
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=author eq 'John Doe' or author eq 'Jane Smith'"
```

### $search Parameter

Performs global search across all article fields using OR logic.

**Format:** Comma-separated search terms

**Example:**
```bash
curl "http://localhost:8080/api/v1/feeds/tech?\$search=AI,machine learning,artificial intelligence"
```

This will find articles containing any of the terms "AI", "machine learning", or "artificial intelligence" in any field.

### $orderby Parameter

Sorts results by specified field and direction.

**Format:** `field direction` (direction can be `asc` or `desc`)

**Supported Fields:**
- `title`
- `author`
- `source`
- `published_at`

**Examples:**
```bash
# Sort by publication date (newest first)
curl "http://localhost:8080/api/v1/feeds/tech?\$orderby=published_at desc"

# Sort by title alphabetically
curl "http://localhost:8080/api/v1/feeds/tech?\$orderby=title asc"
```

### $top and $skip Parameters

Used for pagination.

**Examples:**
```bash
# Get first 10 articles
curl "http://localhost:8080/api/v1/feeds/tech?\$top=10"

# Skip first 20 articles and get next 10
curl "http://localhost:8080/api/v1/feeds/tech?\$skip=20&\$top=10"
```

### $select Parameter

Selects specific fields to include in the response. If not specified, all fields are returned.

**Format:** Comma-separated field names

**Supported Fields:**
- `title`: Article title
- `link`: Article URL
- `description`: Article description
- `content`: Article content
- `author`: Article author
- `source`: RSS feed source
- `categories`: Article categories
- `published_at`: Publication date

**Examples:**
```bash
# Select only title and author
curl "http://localhost:8080/api/v1/feeds/tech?\$select=title,author"

# Select title, link, and publication date
curl "http://localhost:8080/api/v1/feeds/tech?\$select=title,link,published_at"

# Select all text fields
curl "http://localhost:8080/api/v1/feeds/tech?\$select=title,description,content,author"

# Combine with other OData parameters
curl "http://localhost:8080/api/v1/feeds/tech?\$filter=contains(title, 'AI')&\$select=title,author&\$top=5"
```

**Response with field selection:**
```json
{
  "topic": "tech",
  "articles": [
    {
      "title": "AI Breakthrough in Machine Learning",
      "author": "John Doe"
    },
    {
      "title": "New Developments in AI",
      "author": "Jane Smith"
    }
  ],
  "count": 2,
  "updated": "2023-01-15T10:35:00Z"
}
```

**Note:** When using `$select`, only the specified fields are included in the response. All other fields are omitted (empty strings, zero values, or empty arrays).

## Error Responses

### Invalid Filter Expression

When an invalid filter expression is provided:

```json
{
  "error": "Invalid filter expression: syntax error at position 10"
}
```

### Topic Not Found

When requesting a non-existent topic:

```json
{
  "error": "Topic 'invalid-topic' not found"
}
```

### Server Error

When an internal server error occurs:

```json
{
  "error": "Failed to fetch RSS feeds"
}
```

## Caching

The API uses a two-tier caching strategy:

1. **Memory Cache**: Hot data is cached in memory for fast access
2. **Persistent Storage**: All feed data is stored on disk for persistence

Cache behavior:
- Memory cache TTL is configurable via `CACHE_TTL` environment variable
- Background polling keeps data fresh automatically
- Manual refresh endpoints bypass cache
- Cache is automatically invalidated when new data is fetched

## Rate Limiting

Currently, no rate limiting is implemented. However, the background polling system helps reduce load on external RSS feeds by pre-fetching data at regular intervals.

## Background Polling

The service includes a background RSS poller that:

- Automatically fetches RSS feeds at configurable intervals
- Minimizes external requests by pre-fetching data
- Provides API endpoints to monitor and control polling
- Handles errors gracefully without affecting other feeds
- Supports manual force polling for immediate updates

Polling can be configured via the `POLL_INTERVAL` environment variable (default: 15 minutes). 