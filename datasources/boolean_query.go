// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package datasources

import (
	"fmt"
	"strings"
)

const (
	// Boolean operators for query parsing
	OperatorAND = "AND"
	OperatorOR  = "OR"
	OperatorNOT = "NOT"
)

// BooleanNode represents a node in a boolean expression tree
type BooleanNode struct {
	Operator string       // "AND", "OR", "NOT", or "" for leaf nodes
	Left     *BooleanNode // Left child for binary operators
	Right    *BooleanNode // Right child for binary/unary operators
	Value    string       // Keyword value for leaf nodes
}

// ParseBooleanQuery parses a boolean search query into an expression tree
// Supports: AND, OR, NOT, parentheses, quoted phrases
// Example: "(mobile OR web) AND (bug OR issue)"
func ParseBooleanQuery(query string) (*BooleanNode, error) {
	tokens := tokenizeQuery(query)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty query")
	}

	parser := &queryParser{tokens: tokens, pos: 0}
	node, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}

	if parser.pos < len(tokens) {
		return nil, fmt.Errorf("unexpected tokens after expression: %v", tokens[parser.pos:])
	}

	return node, nil
}

// EvaluateBoolean evaluates a boolean expression tree against text content
func EvaluateBoolean(node *BooleanNode, text string) bool {
	if node == nil {
		return false
	}

	textLower := strings.ToLower(text)

	// Leaf node - check if keyword exists in text
	if node.Value != "" {
		return strings.Contains(textLower, strings.ToLower(node.Value))
	}

	// Operator node
	switch node.Operator {
	case OperatorAND:
		return EvaluateBoolean(node.Left, text) && EvaluateBoolean(node.Right, text)
	case OperatorOR:
		return EvaluateBoolean(node.Left, text) || EvaluateBoolean(node.Right, text)
	case OperatorNOT:
		return !EvaluateBoolean(node.Right, text)
	default:
		return false
	}
}

// queryParser handles parsing of tokenized boolean queries
type queryParser struct {
	tokens []string
	pos    int
	depth  int // Current recursion depth to prevent stack exhaustion
}

// parseExpression parses an OR expression (lowest precedence)
func (p *queryParser) parseExpression() (*BooleanNode, error) {
	if p.depth >= MaxBooleanQueryDepth {
		return nil, fmt.Errorf("query too deeply nested (max depth: %d)", MaxBooleanQueryDepth)
	}
	p.depth++
	defer func() { p.depth-- }()

	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.tokens) && strings.ToUpper(p.tokens[p.pos]) == OperatorOR {
		p.pos++ // consume OR
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &BooleanNode{
			Operator: OperatorOR,
			Left:     left,
			Right:    right,
		}
	}

	return left, nil
}

// parseTerm parses an AND expression (higher precedence than OR)
func (p *queryParser) parseTerm() (*BooleanNode, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.tokens) && strings.ToUpper(p.tokens[p.pos]) == OperatorAND {
		p.pos++ // consume AND
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &BooleanNode{
			Operator: OperatorAND,
			Left:     left,
			Right:    right,
		}
	}

	return left, nil
}

// parseFactor parses a NOT expression or primary expression (highest precedence)
func (p *queryParser) parseFactor() (*BooleanNode, error) {
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	// Handle NOT operator
	if strings.ToUpper(p.tokens[p.pos]) == OperatorNOT {
		p.pos++ // consume NOT
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		return &BooleanNode{
			Operator: OperatorNOT,
			Right:    right,
		}, nil
	}

	// Handle parentheses
	if p.tokens[p.pos] == "(" {
		p.pos++ // consume (
		node, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos] != ")" {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++ // consume )
		return node, nil
	}

	// Handle keyword (leaf node)
	value := p.tokens[p.pos]
	p.pos++
	return &BooleanNode{Value: value}, nil
}

// tokenizeQuery splits a query into tokens (keywords, operators, parentheses)
func tokenizeQuery(query string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(query); i++ {
		ch := query[i]

		switch ch {
		case '"':
			if inQuotes {
				// End of quoted phrase
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				inQuotes = false
			} else {
				// Start of quoted phrase
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				inQuotes = true
			}
		case '(', ')':
			if inQuotes {
				current.WriteByte(ch)
			} else {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				tokens = append(tokens, string(ch))
			}
		case ' ', '\t', '\n', '\r':
			if inQuotes {
				current.WriteByte(ch)
			} else if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// ExtractKeywords extracts all leaf keywords from a boolean expression tree
// Useful for fallback simple keyword matching when boolean evaluation isn't needed
func ExtractKeywords(node *BooleanNode) []string {
	if node == nil {
		return nil
	}

	if node.Value != "" {
		return []string{node.Value}
	}

	var keywords []string
	if node.Left != nil {
		keywords = append(keywords, ExtractKeywords(node.Left)...)
	}
	if node.Right != nil {
		keywords = append(keywords, ExtractKeywords(node.Right)...)
	}

	return keywords
}

// FilterDocsByBooleanQuery filters documents based on a boolean query
// Returns only documents where title+content matches the boolean expression
func FilterDocsByBooleanQuery(docs []Doc, topic string) []Doc {
	if topic == "" {
		return docs
	}

	queryNode, err := ParseBooleanQuery(topic)
	if err != nil {
		return docs
	}

	var filtered []Doc
	for _, doc := range docs {
		searchText := doc.Title + " " + doc.Content
		if EvaluateBoolean(queryNode, searchText) {
			filtered = append(filtered, doc)
		}
	}

	return filtered
}

// SimplifyBooleanQueryToKeywords converts a boolean query to simple keywords
// for APIs that don't support boolean operators (AND, OR, NOT, parentheses)
// Returns the original query if it's not a boolean query or if parsing fails
func SimplifyBooleanQueryToKeywords(query string) string {
	if query == "" {
		return query
	}

	queryLower := strings.ToLower(query)
	hasBooleanOperators := strings.Contains(queryLower, " and ") ||
		strings.Contains(queryLower, " or ") ||
		strings.Contains(queryLower, " not ") ||
		strings.Contains(query, "(") ||
		strings.Contains(query, ")")

	if !hasBooleanOperators {
		return query
	}

	queryNode, err := ParseBooleanQuery(query)
	if err != nil {
		return query
	}

	keywords := ExtractKeywords(queryNode)
	if len(keywords) == 0 {
		return query
	}

	return strings.Join(keywords, " ")
}
