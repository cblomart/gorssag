// Package main RSS Aggregator API
//
// RSS Aggregator is a high-performance service that combines multiple RSS feeds per topic,
// provides aggregated feeds in JSON format, and supports advanced OData querying capabilities.
//
//	Schemes: http, https
//	Host: localhost:8080
//	BasePath: /api/v1
//	Version: 1.0.0
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Security:
//	- api_key:
//
// swagger:meta
package main

import "github.com/swaggo/swag"

// @title RSS Aggregator API
// @version 1.0
// @description A high-performance RSS aggregator service with OData support
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey api_key
// @in header
// @name Authorization
// @description API key for authentication

func init() {
	swag.Register(swag.Name, &swag.Spec{
		InfoInstanceName: "swagger",
		SwaggerTemplate:  docTemplate,
	})
}

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "title": "RSS Aggregator API",
        "description": "A high-performance RSS aggregator service with OData support",
        "version": "1.0.0",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "MIT",
            "url": "https://opensource.org/licenses/MIT"
        }
    },
    "host": "localhost:8080",
    "basePath": "/api/v1",
    "schemes": ["http", "https"],
    "consumes": ["application/json"],
    "produces": ["application/json"],
    "paths": {
        "/health": {
            "get": {
                "description": "Health check endpoint",
                "summary": "Health Check",
                "operationId": "healthCheck",
                "responses": {
                    "200": {
                        "description": "Service is healthy",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "status": {
                                    "type": "string",
                                    "example": "ok"
                                },
                                "timestamp": {
                                    "type": "string",
                                    "format": "date-time"
                                },
                                "poller_active": {
                                    "type": "boolean"
                                }
                            }
                        }
                    }
                }
            }
        },
        "/topics": {
            "get": {
                "description": "Get all available RSS feed topics",
                "summary": "List Topics",
                "operationId": "getTopics",
                "responses": {
                    "200": {
                        "description": "List of available topics",
                        "schema": {
                            "type": "array",
                            "items": {
                                "type": "string"
                            },
                            "example": ["tech", "news", "programming"]
                        }
                    },
                    "500": {
                        "description": "Internal server error"
                    }
                }
            }
        },
        "/feeds/{topic}": {
            "get": {
                "description": "Get aggregated RSS feed for a specific topic with optional OData query parameters",
                "summary": "Get Aggregated Feed",
                "operationId": "getAggregatedFeed",
                "parameters": [
                    {
                        "name": "topic",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Topic name"
                    },
                    {
                        "name": "$filter",
                        "in": "query",
                        "required": false,
                        "type": "string",
                        "description": "OData filter expression (e.g., title eq 'AI' and author eq 'John Doe')"
                    },
                    {
                        "name": "$search",
                        "in": "query",
                        "required": false,
                        "type": "string",
                        "description": "Search terms (comma-separated)"
                    },
                    {
                        "name": "$orderby",
                        "in": "query",
                        "required": false,
                        "type": "string",
                        "description": "Sort order (e.g., published_at desc, title asc)"
                    },
                    {
                        "name": "$top",
                        "in": "query",
                        "required": false,
                        "type": "integer",
                        "description": "Maximum number of results"
                    },
                    {
                        "name": "$skip",
                        "in": "query",
                        "required": false,
                        "type": "integer",
                        "description": "Number of results to skip"
                    },
                    {
                        "name": "$select",
                        "in": "query",
                        "required": false,
                        "type": "string",
                        "description": "Fields to include (comma-separated)"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Aggregated feed data",
                        "schema": {
                            "$ref": "#/definitions/AggregatedFeed"
                        }
                    },
                    "400": {
                        "description": "Bad request - invalid OData query"
                    },
                    "404": {
                        "description": "Topic not found"
                    },
                    "500": {
                        "description": "Internal server error"
                    }
                }
            }
        },
        "/feeds/{topic}/info": {
            "get": {
                "description": "Get metadata about a specific topic's feed",
                "summary": "Get Feed Info",
                "operationId": "getFeedInfo",
                "parameters": [
                    {
                        "name": "topic",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Topic name"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Feed metadata",
                        "schema": {
                            "$ref": "#/definitions/FeedInfo"
                        }
                    },
                    "404": {
                        "description": "Topic not found"
                    },
                    "500": {
                        "description": "Internal server error"
                    }
                }
            }
        },
        "/feeds/{topic}/refresh": {
            "post": {
                "description": "Force refresh of a specific topic's feed",
                "summary": "Refresh Feed",
                "operationId": "refreshFeed",
                "parameters": [
                    {
                        "name": "topic",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Topic name"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Feed refreshed successfully",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "message": {
                                    "type": "string",
                                    "example": "Feed refreshed successfully"
                                },
                                "articles_count": {
                                    "type": "integer"
                                }
                            }
                        }
                    },
                    "404": {
                        "description": "Topic not found"
                    },
                    "500": {
                        "description": "Internal server error"
                    }
                }
            }
        },
        "/poller/status": {
            "get": {
                "description": "Get background poller status",
                "summary": "Get Poller Status",
                "operationId": "getPollerStatus",
                "responses": {
                    "200": {
                        "description": "Poller status",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "active": {
                                    "type": "boolean"
                                },
                                "last_polled": {
                                    "type": "object",
                                    "additionalProperties": {
                                        "type": "string",
                                        "format": "date-time"
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        "/poller/force-poll/{topic}": {
            "post": {
                "description": "Force poll a specific topic",
                "summary": "Force Poll Topic",
                "operationId": "forcePollTopic",
                "parameters": [
                    {
                        "name": "topic",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Topic name"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Topic polled successfully",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "message": {
                                    "type": "string",
                                    "example": "Topic polled successfully"
                                },
                                "articles_count": {
                                    "type": "integer"
                                }
                            }
                        }
                    },
                    "404": {
                        "description": "Topic not found"
                    },
                    "500": {
                        "description": "Internal server error"
                    }
                }
            }
        },
        "/poller/last-polled": {
            "get": {
                "description": "Get last polled times for all topics",
                "summary": "Get Last Polled Times",
                "operationId": "getLastPolledTimes",
                "responses": {
                    "200": {
                        "description": "Last polled times",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string",
                                "format": "date-time"
                            }
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "Article": {
            "type": "object",
            "properties": {
                "title": {
                    "type": "string",
                    "description": "Article title"
                },
                "link": {
                    "type": "string",
                    "description": "Article URL"
                },
                "description": {
                    "type": "string",
                    "description": "Article description"
                },
                "content": {
                    "type": "string",
                    "description": "Article content"
                },
                "author": {
                    "type": "string",
                    "description": "Article author"
                },
                "source": {
                    "type": "string",
                    "description": "RSS source name"
                },
                "categories": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Article categories"
                },
                "published_at": {
                    "type": "string",
                    "format": "date-time",
                    "description": "Publication date"
                }
            }
        },
        "AggregatedFeed": {
            "type": "object",
            "properties": {
                "topic": {
                    "type": "string",
                    "description": "Topic name"
                },
                "articles": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/Article"
                    },
                    "description": "List of articles"
                },
                "count": {
                    "type": "integer",
                    "description": "Number of articles"
                },
                "updated": {
                    "type": "string",
                    "format": "date-time",
                    "description": "Last update time"
                }
            }
        },
        "FeedInfo": {
            "type": "object",
            "properties": {
                "topic": {
                    "type": "string",
                    "description": "Topic name"
                },
                "file_size": {
                    "type": "integer",
                    "description": "File size in bytes"
                },
                "last_modified": {
                    "type": "string",
                    "format": "date-time",
                    "description": "Last modification time"
                },
                "article_count": {
                    "type": "integer",
                    "description": "Number of articles"
                }
            }
        }
    },
    "tags": [
        {
            "name": "Health",
            "description": "Health check endpoints"
        },
        {
            "name": "Topics",
            "description": "Topic management endpoints"
        },
        {
            "name": "Feeds",
            "description": "RSS feed endpoints"
        },
        {
            "name": "Poller",
            "description": "Background poller endpoints"
        }
    ]
}`
