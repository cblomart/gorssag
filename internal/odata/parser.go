package odata

import (
	"fmt"
	"strings"
	"time"

	"gorssag/internal/models"
)

type FilterParser struct{}

type FilterExpression struct {
	Operator  string
	Field     string
	Value     string
	Left      *FilterExpression
	Right     *FilterExpression
	Function  string
	Arguments []string
}

func NewFilterParser() *FilterParser {
	return &FilterParser{}
}

func (p *FilterParser) Parse(filter string) (*FilterExpression, error) {
	if filter == "" {
		return nil, nil
	}

	// Remove extra whitespace
	filter = strings.TrimSpace(filter)

	return p.parseExpression(filter)
}

func (p *FilterParser) parseExpression(expr string) (*FilterExpression, error) {
	expr = strings.TrimSpace(expr)

	// Check for logical operators (and, or) - case insensitive
	lowerExpr := strings.ToLower(expr)
	if strings.Contains(lowerExpr, " and ") {
		return p.parseLogicalOperator(expr, "and")
	}
	if strings.Contains(lowerExpr, " or ") {
		return p.parseLogicalOperator(expr, "or")
	}

	// Check for comparison operators
	if strings.Contains(expr, " eq ") {
		return p.parseComparison(expr, "eq")
	}
	if strings.Contains(expr, " ne ") {
		return p.parseComparison(expr, "ne")
	}
	if strings.Contains(expr, " gt ") {
		return p.parseComparison(expr, "gt")
	}
	if strings.Contains(expr, " ge ") {
		return p.parseComparison(expr, "ge")
	}
	if strings.Contains(expr, " lt ") {
		return p.parseComparison(expr, "lt")
	}
	if strings.Contains(expr, " le ") {
		return p.parseComparison(expr, "le")
	}

	// Check for functions
	if strings.HasPrefix(expr, "startswith(") {
		return p.parseFunction(expr, "startswith")
	}
	if strings.HasPrefix(expr, "endswith(") {
		return p.parseFunction(expr, "endswith")
	}
	if strings.HasPrefix(expr, "contains(") {
		return p.parseFunction(expr, "contains")
	}

	return nil, fmt.Errorf("unable to parse expression: %s", expr)
}

func (p *FilterParser) parseLogicalOperator(expr string, op string) (*FilterExpression, error) {
	// Find the position of the logical operator (case insensitive)
	lowerExpr := strings.ToLower(expr)
	opIndex := strings.Index(lowerExpr, " "+op+" ")
	if opIndex == -1 {
		return nil, fmt.Errorf("invalid logical expression: %s", expr)
	}

	// Split on the actual operator (preserving case)
	leftPart := expr[:opIndex]
	rightPart := expr[opIndex+len(" "+op+" "):]

	left, err := p.parseExpression(leftPart)
	if err != nil {
		return nil, err
	}

	right, err := p.parseExpression(rightPart)
	if err != nil {
		return nil, err
	}

	return &FilterExpression{
		Operator: op,
		Left:     left,
		Right:    right,
	}, nil
}

func (p *FilterParser) parseComparison(expr string, op string) (*FilterExpression, error) {
	parts := strings.Split(expr, " "+op+" ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid comparison expression: %s", expr)
	}

	field := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Remove quotes from value
	value = strings.Trim(value, "'\"")

	return &FilterExpression{
		Operator: op,
		Field:    field,
		Value:    value,
	}, nil
}

func (p *FilterParser) parseFunction(expr string, funcName string) (*FilterExpression, error) {
	// Extract arguments from function call
	// e.g., startswith(title, 'AI') -> title, 'AI'
	argsStart := strings.Index(expr, "(")
	argsEnd := strings.LastIndex(expr, ")")

	if argsStart == -1 || argsEnd == -1 {
		return nil, fmt.Errorf("invalid function call: %s", expr)
	}

	argsStr := expr[argsStart+1 : argsEnd]
	args := p.parseFunctionArguments(argsStr)

	if len(args) != 2 {
		return nil, fmt.Errorf("function %s expects 2 arguments, got %d", funcName, len(args))
	}

	return &FilterExpression{
		Function:  funcName,
		Field:     strings.TrimSpace(args[0]),
		Value:     strings.Trim(args[1], "'\""),
		Arguments: args,
	}, nil
}

func (p *FilterParser) parseFunctionArguments(argsStr string) []string {
	var args []string
	var currentArg strings.Builder
	var inQuotes bool
	var quoteChar byte

	for i := 0; i < len(argsStr); i++ {
		char := argsStr[i]

		if !inQuotes && (char == '\'' || char == '"') {
			inQuotes = true
			quoteChar = char
			continue
		}

		if inQuotes && char == quoteChar {
			inQuotes = false
			continue
		}

		if !inQuotes && char == ',' {
			args = append(args, strings.TrimSpace(currentArg.String()))
			currentArg.Reset()
			continue
		}

		currentArg.WriteByte(char)
	}

	// Add the last argument
	if currentArg.Len() > 0 {
		args = append(args, strings.TrimSpace(currentArg.String()))
	}

	return args
}

func (p *FilterParser) Evaluate(expr *FilterExpression, article models.Article) (bool, error) {
	if expr == nil {
		return true, nil
	}

	// Handle logical operators
	if expr.Operator == "and" {
		left, err := p.Evaluate(expr.Left, article)
		if err != nil {
			return false, err
		}
		if !left {
			return false, nil
		}

		right, err := p.Evaluate(expr.Right, article)
		if err != nil {
			return false, err
		}
		return right, nil
	}

	if expr.Operator == "or" {
		left, err := p.Evaluate(expr.Left, article)
		if err != nil {
			return false, err
		}
		if left {
			return true, nil
		}

		right, err := p.Evaluate(expr.Right, article)
		if err != nil {
			return false, err
		}
		return right, nil
	}

	// Handle comparison operators
	if expr.Operator != "" && expr.Field != "" {
		return p.evaluateComparison(expr, article)
	}

	// Handle functions
	if expr.Function != "" {
		return p.evaluateFunction(expr, article)
	}

	return false, fmt.Errorf("invalid filter expression")
}

func (p *FilterParser) evaluateComparison(expr *FilterExpression, article models.Article) (bool, error) {
	fieldValue := p.getFieldValue(expr.Field, article)

	switch expr.Operator {
	case "eq":
		return fieldValue == expr.Value, nil
	case "ne":
		return fieldValue != expr.Value, nil
	case "gt":
		return p.compareValues(fieldValue, expr.Value) > 0, nil
	case "ge":
		return p.compareValues(fieldValue, expr.Value) >= 0, nil
	case "lt":
		return p.compareValues(fieldValue, expr.Value) < 0, nil
	case "le":
		return p.compareValues(fieldValue, expr.Value) <= 0, nil
	default:
		return false, fmt.Errorf("unsupported comparison operator: %s", expr.Operator)
	}
}

func (p *FilterParser) evaluateFunction(expr *FilterExpression, article models.Article) (bool, error) {
	fieldValue := p.getFieldValue(expr.Field, article)
	searchValue := strings.ToLower(expr.Value)

	switch expr.Function {
	case "startswith":
		return strings.HasPrefix(strings.ToLower(fieldValue), searchValue), nil
	case "endswith":
		return strings.HasSuffix(strings.ToLower(fieldValue), searchValue), nil
	case "contains":
		return strings.Contains(strings.ToLower(fieldValue), searchValue), nil
	default:
		return false, fmt.Errorf("unsupported function: %s", expr.Function)
	}
}

func (p *FilterParser) getFieldValue(field string, article models.Article) string {
	switch strings.ToLower(field) {
	case "title":
		return article.Title
	case "description":
		return article.Description
	case "content":
		return article.Content
	case "author":
		return article.Author
	case "source":
		return article.Source
	case "published_at":
		return article.PublishedAt.Format(time.RFC3339)
	default:
		return ""
	}
}

func (p *FilterParser) compareValues(a, b string) int {
	// Try to parse as dates first
	timeA, errA := time.Parse(time.RFC3339, a)
	timeB, errB := time.Parse(time.RFC3339, b)

	if errA == nil && errB == nil {
		if timeA.Before(timeB) {
			return -1
		}
		if timeA.After(timeB) {
			return 1
		}
		return 0
	}

	// Fallback to string comparison
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}
