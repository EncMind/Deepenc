package main

import (
	"fmt"
	"strings"
	"testing"
)

// convertParsedContentToMarkdown converts ParsedContent back to markdown string
func convertParsedContentToMarkdown(parsed *ParsedContent) string {
	if parsed == nil {
		return ""
	}

	var result strings.Builder
	for _, block := range parsed.Blocks {
		if block.Type == "code" {
			result.WriteString("```")
			if block.Language != "" {
				result.WriteString(block.Language)
			}
			result.WriteString("\n")
			result.WriteString(block.Content)
			result.WriteString("\n```\n")
		} else {
			result.WriteString(block.Content)
			result.WriteString("\n")
		}
	}
	return strings.TrimSpace(result.String())
}
func TestContentParserService(t *testing.T) {
	// Initialize the content parser service
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	// Test basic initialization
	t.Run("Service Initialization", func(t *testing.T) {
		if service.strategies == nil {
			t.Error("Strategies map should not be nil")
		}
		if len(service.strategies) == 0 {
			t.Error("Strategies map should contain at least one strategy")
		}
	})
}

func TestProviderStrategies(t *testing.T) {
	service := NewContentParserService()

	providers := []string{"openai", "claude", "gemini"}
	testContent := "def test():\n    return 'Hello World'"

	for _, provider := range providers {
		t.Run(fmt.Sprintf("Provider_%s", provider), func(t *testing.T) {
			parsed, err := service.ParseContent(testContent, provider)
			if err != nil {
				t.Errorf("Provider %s failed: %v", provider, err)
				return
			}

			result := convertParsedContentToMarkdown(parsed)
			if result == "" {
				t.Errorf("Provider %s returned empty result", provider)
			}

			// OpenAI should wrap code, others might not need to
			if provider == "openai" {
				if !strings.Contains(result, "```") {
					t.Errorf("OpenAI provider should wrap code in blocks, got: %q", result)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkContentParsing(b *testing.B) {
	service := NewContentParserService()
	testContent := `Here's a Python function:

def factorial(n):
    if n == 0 or n == 1:
        return 1
    return n * factorial(n - 1)

This function calculates factorial recursively.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ParseContent(testContent, "openai")
		if err != nil {
			b.Fatalf("ParseContent failed: %v", err)
		}
	}
}

func BenchmarkLargeContent(b *testing.B) {
	service := NewContentParserService()

	// Generate large content
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("def function_%d():", i))
		lines = append(lines, "    return 42")
		lines = append(lines, "")
	}
	testContent := strings.Join(lines, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ParseContent(testContent, "openai")
		if err != nil {
			b.Fatalf("ParseContent failed: %v", err)
		}
	}
}