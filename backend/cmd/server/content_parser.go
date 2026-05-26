package main

import (
	"log"
	"regexp"
	"strings"
	"time"
)

// ContentBlock represents a parsed content block with type and metadata
type ContentBlock struct {
	Type       string                 `json:"type"`
	Content    string                 `json:"content"`
	Language   string                 `json:"language,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Confidence float64                `json:"confidence"`
}

// ParsedContent represents the complete parsed response from an AI provider
type ParsedContent struct {
	Blocks     []ContentBlock         `json:"blocks"`
	Metadata   map[string]interface{} `json:"metadata"`
	Provider   string                 `json:"provider"`
	Confidence float64                `json:"confidence"`
	ParsedAt   time.Time              `json:"parsedAt"`
}

// ParsingStrategy interface for different provider parsing strategies
type ParsingStrategy interface {
	Parse(content string) (*ParsedContent, error)
	GetConfidence() float64
	GetProvider() string
}

// ContentParserService provides multi-provider content parsing
type ContentParserService struct {
	strategies map[string]ParsingStrategy
}

// NewContentParserService creates a new content parser service
func NewContentParserService() *ContentParserService {
	service := &ContentParserService{
		strategies: make(map[string]ParsingStrategy),
	}

	// Register strategies
	service.strategies["openai"] = &OpenAIParsingStrategy{}
	service.strategies["claude"] = &ClaudeParsingStrategy{}
	service.strategies["gemini"] = &GeminiParsingStrategy{}
	service.strategies["universal"] = &UniversalParsingStrategy{}

	return service
}

// ParseContent parses content using the appropriate strategy
func (cps *ContentParserService) ParseContent(content string, provider string) (*ParsedContent, error) {
	strategy, exists := cps.strategies[provider]
	if !exists {
		// Fallback to universal strategy
		strategy = cps.strategies["universal"]
		log.Printf("⚠️ Unknown provider %s, using universal strategy", provider)
	}

	return strategy.Parse(content)
}

// OpenAIParsingStrategy implements parsing for OpenAI responses
type OpenAIParsingStrategy struct{}

func (o *OpenAIParsingStrategy) Parse(content string) (*ParsedContent, error) {
	blocks := []ContentBlock{}
	confidence := 0.95 // High confidence for OpenAI structured responses

	// Parse using OpenAI-specific patterns
	blocks = append(blocks, o.parseCodeBlocks(content)...)
	blocks = append(blocks, o.parseMathBlocks(content)...)
	blocks = append(blocks, o.parseListBlocks(content)...)
	blocks = append(blocks, o.parseTextBlocks(content, blocks)...)

	return &ParsedContent{
		Blocks:     blocks,
		Provider:   "openai",
		Confidence: confidence,
		ParsedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"strategy": "openai_structured",
			"version":  "1.0",
		},
	}, nil
}

func (o *OpenAIParsingStrategy) parseCodeBlocks(content string) []ContentBlock {
	blocks := []ContentBlock{}

	// Parse standard markdown code blocks with triple backticks (```language ... ```)
	// This is the primary method since modern OpenAI models (GPT-4, GPT-4o-mini)
	// always return properly formatted markdown
	codeRegex := regexp.MustCompile("```([^\\n`]*?)\\n([\\s\\S]*?)```")
	matches := codeRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		language := strings.TrimSpace(match[1])
		code := strings.TrimSpace(match[2])

		blocks = append(blocks, ContentBlock{
			Type:       "code",
			Content:    code,
			Language:   language,
			Confidence: 0.98,
			Metadata: map[string]interface{}{
				"format": "standard_backticks",
			},
		})
	}

	// Parse legacy "Code:" header format (rare edge case)
	codeHeaderRegex := regexp.MustCompile("(?i)^Code:\\s*$\\n([\\s\\S]*?)(?:\\n\\n|\\n[A-Z]|$)")
	headerMatches := codeHeaderRegex.FindAllStringSubmatch(content, -1)

	for _, match := range headerMatches {
		code := strings.TrimSpace(match[1])

		blocks = append(blocks, ContentBlock{
			Type:       "code",
			Content:    code,
			Language:   "python",
			Confidence: 0.85,
			Metadata: map[string]interface{}{
				"format": "header_format",
			},
		})
	}

	// Fallback: Use heuristic detection ONLY if no markdown blocks found
	// This prevents duplicate detection when responses already have proper formatting
	if len(blocks) == 0 {
		log.Printf("[OpenAI] No markdown code blocks found, using heuristic detection")
		blocks = append(blocks, o.detectHeuristicCodeBlocks(content)...)
	}

	return blocks
}

func (o *OpenAIParsingStrategy) parseMathBlocks(content string) []ContentBlock {
	blocks := []ContentBlock{}

	// LaTeX math blocks
	mathRegex := regexp.MustCompile("\\$\\$([\\s\\S]*?)\\$\\$")
	matches := mathRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		math := strings.TrimSpace(match[1])
		blocks = append(blocks, ContentBlock{
			Type:       "math",
			Content:    math,
			Confidence: 0.95,
			Metadata: map[string]interface{}{
				"format": "latex",
			},
		})
	}

	return blocks
}

func (o *OpenAIParsingStrategy) parseListBlocks(content string) []ContentBlock {
	blocks := []ContentBlock{}

	// Numbered lists
	listRegex := regexp.MustCompile("(?m)^\\d+\\.\\s+.*(?:\\n\\d+\\.\\s+.*)*")
	matches := listRegex.FindAllString(content, -1)

	for _, match := range matches {
		blocks = append(blocks, ContentBlock{
			Type:       "list",
			Content:    strings.TrimSpace(match),
			Confidence: 0.90,
			Metadata: map[string]interface{}{
				"list_type": "ordered",
			},
		})
	}

	// Bullet lists
	bulletRegex := regexp.MustCompile("(?m)^[-*+]\\s+.*(?:\\n[-*+]\\s+.*)*")
	bulletMatches := bulletRegex.FindAllString(content, -1)

	for _, match := range bulletMatches {
		blocks = append(blocks, ContentBlock{
			Type:       "list",
			Content:    strings.TrimSpace(match),
			Confidence: 0.90,
			Metadata: map[string]interface{}{
				"list_type": "unordered",
			},
		})
	}

	return blocks
}

func (o *OpenAIParsingStrategy) parseTextBlocks(content string, existingBlocks []ContentBlock) []ContentBlock {
	// Extract text that doesn't belong to other blocks
	remainingContent := content

	// Remove already parsed content
	for _, block := range existingBlocks {
		remainingContent = strings.ReplaceAll(remainingContent, block.Content, "")
	}

	// Clean up and split into paragraphs
	paragraphs := strings.Split(strings.TrimSpace(remainingContent), "\n\n")
	blocks := []ContentBlock{}

	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) != "" {
			blocks = append(blocks, ContentBlock{
				Type:       "text",
				Content:    strings.TrimSpace(paragraph),
				Confidence: 0.80,
				Metadata: map[string]interface{}{
					"format": "paragraph",
				},
			})
		}
	}

	return blocks
}

// detectHeuristicCodeBlocks identifies code blocks without explicit markdown formatting
func (o *OpenAIParsingStrategy) detectHeuristicCodeBlocks(content string) []ContentBlock {
	log.Printf("[OpenAI] 🔍 Running heuristic code detection on content length: %d", len(content))
	blocks := []ContentBlock{}
	lines := strings.Split(content, "\n")

	var currentCodeBlock []string
	var blockStartIdx int
	inCodeBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Strong code indicators
		isCodeLine := o.isStrongCodeLine(trimmed)

		if isCodeLine && !inCodeBlock {
			// Start new code block
			inCodeBlock = true
			blockStartIdx = i
			currentCodeBlock = []string{line}
		} else if inCodeBlock {
			// Check if we should continue or end the code block
			if o.shouldContinueCodeBlock(trimmed, line) {
				currentCodeBlock = append(currentCodeBlock, line)
			} else {
				// End code block
				if len(currentCodeBlock) >= 1 { // At least 1 line for a valid code block
					codeContent := strings.TrimSpace(strings.Join(currentCodeBlock, "\n"))
					if len(codeContent) > 10 { // Reduced minimum length check
						log.Printf("[OpenAI] 🔍 Found heuristic code block (%d lines, %d chars)", len(currentCodeBlock), len(codeContent))
						blocks = append(blocks, ContentBlock{
							Type:       "code",
							Content:    codeContent,
							Language:   o.detectLanguage(codeContent),
							Confidence: 0.75,
							Metadata: map[string]interface{}{
								"format": "heuristic_detection",
								"line_start": blockStartIdx,
								"line_end": i - 1,
							},
						})
					}
				}
				inCodeBlock = false
				currentCodeBlock = nil

				// Check if this line starts a new code block
				if isCodeLine {
					inCodeBlock = true
					blockStartIdx = i
					currentCodeBlock = []string{line}
				}
			}
		}
	}

	// Handle code block at end of content
	if inCodeBlock && len(currentCodeBlock) >= 1 {
		codeContent := strings.TrimSpace(strings.Join(currentCodeBlock, "\n"))
		if len(codeContent) > 10 {
			blocks = append(blocks, ContentBlock{
				Type:       "code",
				Content:    codeContent,
				Language:   o.detectLanguage(codeContent),
				Confidence: 0.75,
				Metadata: map[string]interface{}{
					"format": "heuristic_detection",
					"line_start": blockStartIdx,
					"line_end": len(lines) - 1,
				},
			})
		}
	}

	log.Printf("[OpenAI] 🔍 Heuristic detection found %d code blocks", len(blocks))
	return blocks
}

// isStrongCodeLine identifies lines that strongly indicate code
func (o *OpenAIParsingStrategy) isStrongCodeLine(line string) bool {
	// Python/Go imports
	if regexp.MustCompile(`^(import|from)\s+\w+`).MatchString(line) {
		return true
	}

	// Function definitions (Python, Go, JS, etc.)
	if regexp.MustCompile(`^(def|func|function|const\s+\w+\s*=\s*function|const\s+\w+\s*=\s*\()\s+\w+\s*[\(\=]`).MatchString(line) {
		return true
	}

	// Class definitions
	if regexp.MustCompile(`^(class|type\s+\w+\s+struct|interface\s+\w+)`).MatchString(line) {
		return true
	}

	// Variable assignments and declarations
	if regexp.MustCompile(`^\w+\s*[:\=]\s*[\w\"\'\[\{]|^(let|const|var)\s+\w+`).MatchString(line) {
		return true
	}

	// Control structures
	if regexp.MustCompile(`^(if|for|while|try|catch|switch|case)\s+`).MatchString(line) {
		return true
	}

	// Return statements
	if regexp.MustCompile(`^return\s+`).MatchString(line) {
		return true
	}

	// Function calls that look like code
	if regexp.MustCompile(`^\w+\.\w+\s*\(|^\w+\s*\(`).MatchString(line) {
		return true
	}

	// Array/object literals
	if regexp.MustCompile(`^\s*[\[\{]\s*$|^\s*[\[\{].*[\]\}]\s*[,;]?\s*$`).MatchString(line) {
		return true
	}

	// Method chaining (common in modern code)
	if regexp.MustCompile(`^\s*\.\w+\(`).MatchString(line) {
		return true
	}

	return false
}

// shouldContinueCodeBlock determines if a line should be part of an ongoing code block
func (o *OpenAIParsingStrategy) shouldContinueCodeBlock(trimmed, fullLine string) bool {
	// Empty lines in code blocks are OK
	if trimmed == "" {
		return true
	}

	// Indented lines (likely part of code structure)
	if strings.HasPrefix(fullLine, "    ") || strings.HasPrefix(fullLine, "\t") || strings.HasPrefix(fullLine, "  ") {
		return true
	}

	// Comments
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
		return true
	}

	// Common code patterns - be more generous
	if regexp.MustCompile(`^\w+\s*[\(\[\{]|^\s*[\)\]\}]|^\w+\s*[=:]|^return\s|^if\s|^for\s|^while\s|^else\s|^elif\s|^except\s|^try\s|^finally\s`).MatchString(trimmed) {
		return true
	}

	// Variable assignments, function calls, method chaining
	if regexp.MustCompile(`^\w+\s*[=:]\s*|^\w+\.\w+|^\.\w+\(|^\w+\s*\(`).MatchString(trimmed) {
		return true
	}

	// String literals, numbers, operators
	if regexp.MustCompile(`^[\"\'].*[\"\']$|^\d+\.?\d*$|^[+\-*/=<>!&|]+$`).MatchString(trimmed) {
		return true
	}

	// Brackets, braces, semicolons - typical code endings
	if regexp.MustCompile(`^[\[\{\(\)\]\}];?$|.*[;,\{\}\[\]]$`).MatchString(trimmed) {
		return true
	}

	// Stop at clear explanatory text (starts with uppercase and is sentence-like)
	if regexp.MustCompile(`^[A-Z][a-z]{4,}.*[\.!?]$`).MatchString(trimmed) && len(trimmed) > 20 {
		return false
	}

	// Stop at markdown headers
	if strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, " ") {
		return false
	}

	// Stop at clear explanatory sentences (multiple words starting with capital)
	if regexp.MustCompile(`^[A-Z][a-z]+\s+[a-z]+.*[a-z]$`).MatchString(trimmed) && len(trimmed) > 25 {
		return false
	}

	// Default to continuing if we're not sure
	return true
}

// detectLanguage attempts to identify the programming language
func (o *OpenAIParsingStrategy) detectLanguage(code string) string {
	// Python indicators
	if regexp.MustCompile(`import\s+(pandas|numpy|polars|dask)|def\s+\w+\s*\(|\.str\.|\.with_columns`).MatchString(code) {
		return "python"
	}

	// Go indicators
	if regexp.MustCompile(`func\s+\w+\s*\(|type\s+\w+\s+struct|package\s+\w+`).MatchString(code) {
		return "go"
	}

	// JavaScript/TypeScript indicators
	if regexp.MustCompile(`function\s+\w+|const\s+\w+\s*=|let\s+\w+\s*=|=>\s*\{`).MatchString(code) {
		return "javascript"
	}

	return "python" // Default assumption for data science context
}

func (o *OpenAIParsingStrategy) GetConfidence() float64 {
	return 0.95
}

func (o *OpenAIParsingStrategy) GetProvider() string {
	return "openai"
}

// ClaudeParsingStrategy implements parsing for Claude responses
type ClaudeParsingStrategy struct{}

func (c *ClaudeParsingStrategy) Parse(content string) (*ParsedContent, error) {
	blocks := []ContentBlock{}
	confidence := 0.85 // Good confidence for Claude responses

	// Claude often uses well-structured responses
	blocks = append(blocks, c.parseCodeBlocks(content)...)
	blocks = append(blocks, c.parseThinkingBlocks(content)...)
	blocks = append(blocks, c.parseTextBlocks(content, blocks)...)

	return &ParsedContent{
		Blocks:     blocks,
		Provider:   "claude",
		Confidence: confidence,
		ParsedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"strategy": "claude_structured",
			"version":  "1.0",
		},
	}, nil
}

func (c *ClaudeParsingStrategy) parseCodeBlocks(content string) []ContentBlock {
	blocks := []ContentBlock{}

	// Standard code blocks
	codeRegex := regexp.MustCompile("```([^\\n`]*?)\\n([\\s\\S]*?)```")
	matches := codeRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		language := strings.TrimSpace(match[1])
		code := strings.TrimSpace(match[2])

		blocks = append(blocks, ContentBlock{
			Type:       "code",
			Content:    code,
			Language:   language,
			Confidence: 0.95,
			Metadata: map[string]interface{}{
				"format": "claude_backticks",
			},
		})
	}

	return blocks
}

func (c *ClaudeParsingStrategy) parseThinkingBlocks(content string) []ContentBlock {
	blocks := []ContentBlock{}

	// Claude often uses <thinking> tags
	thinkingRegex := regexp.MustCompile("(?s)<thinking>(.*?)</thinking>")
	matches := thinkingRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		thinking := strings.TrimSpace(match[1])
		blocks = append(blocks, ContentBlock{
			Type:       "thinking",
			Content:    thinking,
			Confidence: 0.98,
			Metadata: map[string]interface{}{
				"format": "claude_thinking",
			},
		})
	}

	return blocks
}

func (c *ClaudeParsingStrategy) parseTextBlocks(content string, existingBlocks []ContentBlock) []ContentBlock {
	// Similar to OpenAI but with Claude-specific adaptations
	remainingContent := content

	// Remove thinking blocks and code blocks
	thinkingRegex := regexp.MustCompile("(?s)<thinking>.*?</thinking>")
	remainingContent = thinkingRegex.ReplaceAllString(remainingContent, "")

	codeRegex := regexp.MustCompile("```[\\s\\S]*?```")
	remainingContent = codeRegex.ReplaceAllString(remainingContent, "")

	paragraphs := strings.Split(strings.TrimSpace(remainingContent), "\n\n")
	blocks := []ContentBlock{}

	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) != "" {
			blocks = append(blocks, ContentBlock{
				Type:       "text",
				Content:    strings.TrimSpace(paragraph),
				Confidence: 0.85,
				Metadata: map[string]interface{}{
					"format": "claude_paragraph",
				},
			})
		}
	}

	return blocks
}

func (c *ClaudeParsingStrategy) GetConfidence() float64 {
	return 0.85
}

func (c *ClaudeParsingStrategy) GetProvider() string {
	return "claude"
}

// GeminiParsingStrategy implements parsing for Gemini responses
type GeminiParsingStrategy struct{}

func (g *GeminiParsingStrategy) Parse(content string) (*ParsedContent, error) {
	blocks := []ContentBlock{}
	confidence := 0.80 // Moderate confidence for Gemini responses

	// Gemini responses can be less structured
	blocks = append(blocks, g.parseCodeBlocks(content)...)
	blocks = append(blocks, g.parseTextBlocks(content, blocks)...)

	return &ParsedContent{
		Blocks:     blocks,
		Provider:   "gemini",
		Confidence: confidence,
		ParsedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"strategy": "gemini_heuristic",
			"version":  "1.0",
		},
	}, nil
}

func (g *GeminiParsingStrategy) parseCodeBlocks(content string) []ContentBlock {
	blocks := []ContentBlock{}

	// Standard code blocks
	codeRegex := regexp.MustCompile("```([^\\n`]*?)\\n([\\s\\S]*?)```")
	matches := codeRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		language := strings.TrimSpace(match[1])
		code := strings.TrimSpace(match[2])

		blocks = append(blocks, ContentBlock{
			Type:       "code",
			Content:    code,
			Language:   language,
			Confidence: 0.90,
			Metadata: map[string]interface{}{
				"format": "gemini_backticks",
			},
		})
	}

	return blocks
}

func (g *GeminiParsingStrategy) parseTextBlocks(content string, existingBlocks []ContentBlock) []ContentBlock {
	remainingContent := content

	// Remove code blocks
	codeRegex := regexp.MustCompile("```[\\s\\S]*?```")
	remainingContent = codeRegex.ReplaceAllString(remainingContent, "")

	paragraphs := strings.Split(strings.TrimSpace(remainingContent), "\n\n")
	blocks := []ContentBlock{}

	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) != "" {
			blocks = append(blocks, ContentBlock{
				Type:       "text",
				Content:    strings.TrimSpace(paragraph),
				Confidence: 0.75,
				Metadata: map[string]interface{}{
					"format": "gemini_paragraph",
				},
			})
		}
	}

	return blocks
}

func (g *GeminiParsingStrategy) GetConfidence() float64 {
	return 0.80
}

func (g *GeminiParsingStrategy) GetProvider() string {
	return "gemini"
}

// UniversalParsingStrategy implements a fallback parsing strategy
type UniversalParsingStrategy struct{}

func (u *UniversalParsingStrategy) Parse(content string) (*ParsedContent, error) {
	blocks := []ContentBlock{}
	confidence := 0.70 // Lower confidence for universal parsing

	// Use robust, universal patterns
	blocks = append(blocks, u.parseCodeBlocks(content)...)
	blocks = append(blocks, u.parseTextBlocks(content, blocks)...)

	return &ParsedContent{
		Blocks:     blocks,
		Provider:   "universal",
		Confidence: confidence,
		ParsedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"strategy": "universal_fallback",
			"version":  "1.0",
		},
	}, nil
}

func (u *UniversalParsingStrategy) parseCodeBlocks(content string) []ContentBlock {
	blocks := []ContentBlock{}

	// Multiple code block patterns
	patterns := []string{
		"```([^\\n`]*?)\\n([\\s\\S]*?)```",           // Standard backticks
		"'''([^\\n']*?)\\n([\\s\\S]*?)'''",           // Triple quotes
		"(?i)^Code:\\s*$\\n([\\s\\S]*?)(?:\\n\\n|$)", // Code: header
	}

	for i, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			var language, code string
			var format string

			switch i {
			case 0, 1:
				language = strings.TrimSpace(match[1])
				code = strings.TrimSpace(match[2])
				format = "backticks"
			case 2:
				language = "python" // Default assumption
				code = strings.TrimSpace(match[1])
				format = "header"
			}

			blocks = append(blocks, ContentBlock{
				Type:       "code",
				Content:    code,
				Language:   language,
				Confidence: 0.75,
				Metadata: map[string]interface{}{
					"format": format,
					"pattern_index": i,
				},
			})
		}
	}

	return blocks
}

func (u *UniversalParsingStrategy) parseTextBlocks(content string, existingBlocks []ContentBlock) []ContentBlock {
	remainingContent := content

	// Remove all code blocks
	patterns := []string{
		"```[\\s\\S]*?```",
		"'''[\\s\\S]*?'''",
		"(?i)^Code:\\s*$\\n[\\s\\S]*?(?:\\n\\n|$)",
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		remainingContent = regex.ReplaceAllString(remainingContent, "")
	}

	paragraphs := strings.Split(strings.TrimSpace(remainingContent), "\n\n")
	blocks := []ContentBlock{}

	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) != "" {
			blocks = append(blocks, ContentBlock{
				Type:       "text",
				Content:    strings.TrimSpace(paragraph),
				Confidence: 0.70,
				Metadata: map[string]interface{}{
					"format": "universal_paragraph",
				},
			})
		}
	}

	return blocks
}

func (u *UniversalParsingStrategy) GetConfidence() float64 {
	return 0.70
}

func (u *UniversalParsingStrategy) GetProvider() string {
	return "universal"
}

//  renderContentBlocksToMarkdown converts parsed content blocks back to proper markdown format
func renderContentBlocksToMarkdown(blocks []ContentBlock) string {
	var result strings.Builder

	for i, block := range blocks {
		switch block.Type {
		case "code":
			// Render code block with proper fences
			result.WriteString("```")
			if block.Language != "" {
				result.WriteString(block.Language)
			}
			result.WriteString("\n")
			result.WriteString(block.Content)
			result.WriteString("\n```\n")
		case "text":
			// Render text blocks
			result.WriteString(block.Content)
		case "math":
			// Render math blocks
			if block.Metadata != nil {
				if displayMode, ok := block.Metadata["display_mode"].(string); ok && displayMode == "block" {
					result.WriteString("$$\n")
					result.WriteString(block.Content)
					result.WriteString("\n$$\n")
				} else {
					result.WriteString("$")
					result.WriteString(block.Content)
					result.WriteString("$")
				}
			} else {
				result.WriteString("$")
				result.WriteString(block.Content)
				result.WriteString("$")
			}
		default:
			// Default text rendering
			result.WriteString(block.Content)
		}

		// Add spacing between blocks
		if i < len(blocks)-1 {
			// Add extra spacing after code blocks and before code blocks
			if block.Type == "code" || (i+1 < len(blocks) && blocks[i+1].Type == "code") {
				result.WriteString("\n\n")
			} else {
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}

// Initialize the content parser service
func initContentParserService() {
	contentParserService = NewContentParserService()
	log.Printf("✅ Content parser service initialized with %d strategies", len(contentParserService.strategies))
}