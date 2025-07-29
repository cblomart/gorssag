package odata

import (
	"testing"
	"time"

	"gorssag/internal/models"
)

func TestFilterParser_ParseComparison(t *testing.T) {
	parser := NewFilterParser()

	tests := []struct {
		name     string
		filter   string
		expected *FilterExpression
	}{
		{
			name:   "equals operator",
			filter: "title eq 'AI'",
			expected: &FilterExpression{
				Operator: "eq",
				Field:    "title",
				Value:    "AI",
			},
		},
		{
			name:   "not equals operator",
			filter: "author ne 'John Doe'",
			expected: &FilterExpression{
				Operator: "ne",
				Field:    "author",
				Value:    "John Doe",
			},
		},
		{
			name:   "greater than operator",
			filter: "published_at gt '2023-01-01T00:00:00Z'",
			expected: &FilterExpression{
				Operator: "gt",
				Field:    "published_at",
				Value:    "2023-01-01T00:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.filter)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if result.Operator != tt.expected.Operator {
				t.Errorf("Operator = %v, want %v", result.Operator, tt.expected.Operator)
			}
			if result.Field != tt.expected.Field {
				t.Errorf("Field = %v, want %v", result.Field, tt.expected.Field)
			}
			if result.Value != tt.expected.Value {
				t.Errorf("Value = %v, want %v", result.Value, tt.expected.Value)
			}
		})
	}
}

func TestFilterParser_ParseFunctions(t *testing.T) {
	parser := NewFilterParser()

	tests := []struct {
		name     string
		filter   string
		expected *FilterExpression
	}{
		{
			name:   "startswith function",
			filter: "startswith(title, 'AI')",
			expected: &FilterExpression{
				Function: "startswith",
				Field:    "title",
				Value:    "AI",
			},
		},
		{
			name:   "endswith function",
			filter: "endswith(title, 'News')",
			expected: &FilterExpression{
				Function: "endswith",
				Field:    "title",
				Value:    "News",
			},
		},
		{
			name:   "contains function",
			filter: "contains(description, 'technology')",
			expected: &FilterExpression{
				Function: "contains",
				Field:    "description",
				Value:    "technology",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.filter)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if result.Function != tt.expected.Function {
				t.Errorf("Function = %v, want %v", result.Function, tt.expected.Function)
			}
			if result.Field != tt.expected.Field {
				t.Errorf("Field = %v, want %v", result.Field, tt.expected.Field)
			}
			if result.Value != tt.expected.Value {
				t.Errorf("Value = %v, want %v", result.Value, tt.expected.Value)
			}
		})
	}
}

func TestFilterParser_ParseLogicalOperators(t *testing.T) {
	parser := NewFilterParser()

	filter := "title eq 'AI' and author eq 'John Doe'"
	result, err := parser.Parse(filter)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Operator != "and" {
		t.Errorf("Operator = %v, want 'and'", result.Operator)
	}

	if result.Left == nil || result.Right == nil {
		t.Error("Expected left and right expressions to be parsed")
	}

	if result.Left.Operator != "eq" || result.Left.Field != "title" {
		t.Error("Left expression not parsed correctly")
	}

	if result.Right.Operator != "eq" || result.Right.Field != "author" {
		t.Error("Right expression not parsed correctly")
	}
}

func TestFilterParser_Evaluate(t *testing.T) {
	parser := NewFilterParser()

	article := models.Article{
		Title:       "AI Technology News",
		Description: "Latest developments in artificial intelligence",
		Content:     "This article discusses the latest AI breakthroughs",
		Author:      "John Doe",
		Source:      "Tech News",
		PublishedAt: time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
		Categories:  []string{"technology", "ai"},
	}

	tests := []struct {
		name     string
		filter   string
		expected bool
	}{
		{
			name:     "equals match",
			filter:   "title eq 'AI Technology News'",
			expected: true,
		},
		{
			name:     "equals no match",
			filter:   "title eq 'Wrong Title'",
			expected: false,
		},
		{
			name:     "startswith match",
			filter:   "startswith(title, 'AI')",
			expected: true,
		},
		{
			name:     "startswith no match",
			filter:   "startswith(title, 'Wrong')",
			expected: false,
		},
		{
			name:     "contains match",
			filter:   "contains(description, 'artificial intelligence')",
			expected: true,
		},
		{
			name:     "and operator both true",
			filter:   "title eq 'AI Technology News' and author eq 'John Doe'",
			expected: true,
		},
		{
			name:     "and operator one false",
			filter:   "title eq 'AI Technology News' and author eq 'Wrong Author'",
			expected: false,
		},
		{
			name:     "or operator one true",
			filter:   "title eq 'Wrong Title' or author eq 'John Doe'",
			expected: true,
		},
		{
			name:     "or operator both false",
			filter:   "title eq 'Wrong Title' or author eq 'Wrong Author'",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.Parse(tt.filter)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result, err := parser.Evaluate(expr, article)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("Evaluate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterParser_GetFieldValue(t *testing.T) {
	parser := NewFilterParser()

	article := models.Article{
		Title:       "Test Title",
		Description: "Test Description",
		Content:     "Test Content",
		Author:      "Test Author",
		Source:      "Test Source",
		PublishedAt: time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
		Categories:  []string{"test", "category"},
	}

	tests := []struct {
		field string
		want  string
	}{
		{"title", "Test Title"},
		{"description", "Test Description"},
		{"content", "Test Content"},
		{"author", "Test Author"},
		{"source", "Test Source"},
		{"published_at", "2023-06-15T10:00:00Z"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := parser.getFieldValue(tt.field, article)
			if got != tt.want {
				t.Errorf("getFieldValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
