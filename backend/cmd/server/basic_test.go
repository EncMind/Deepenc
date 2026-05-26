package main

import (
	"strings"
	"testing"
)

func TestBasicContentParser(t *testing.T) {
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	if service.strategies == nil {
		t.Error("Strategies map should not be nil")
	}

	if len(service.strategies) == 0 {
		t.Error("Strategies map should contain at least one strategy")
	}

	// Test basic parsing
	testContent := "def hello():\n    print('world')"
	parsed, err := service.ParseContent(testContent, "openai")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if parsed == nil {
		t.Fatal("ParseContent returned nil")
	}

	if len(parsed.Blocks) == 0 {
		t.Error("ParseContent should return at least one block")
	}

	// Check that code was detected
	hasCodeBlock := false
	for _, block := range parsed.Blocks {
		if block.Type == "code" {
			hasCodeBlock = true
			break
		}
	}

	if !hasCodeBlock {
		t.Error("Expected code block to be detected in Python code")
	}
}

func TestContentParserWithTestData(t *testing.T) {
	service := NewContentParserService()

	testCases := []struct {
		name     string
		content  string
		provider string
		expectCode bool
	}{
		{
			name:     "Python Function",
			content:  "def factorial(n):\n    return 1 if n <= 1 else n * factorial(n-1)",
			provider: "openai",
			expectCode: true,
		},
		{
			name:     "JavaScript Function",
			content:  "function greet(name) {\n    console.log(`Hello, ${name}!`);\n}",
			provider: "openai",
			expectCode: true,
		},
		{
			name:     "Plain Text",
			content:  "This is just regular text without any code.",
			provider: "openai",
			expectCode: false,
		},
		{
			name:     "Already Formatted",
			content:  "```python\ndef hello():\n    print('world')\n```",
			provider: "claude",
			expectCode: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := service.ParseContent(tc.content, tc.provider)
			if err != nil {
				t.Fatalf("ParseContent failed: %v", err)
			}

			if parsed == nil {
				t.Fatal("ParseContent returned nil")
			}

			hasCodeBlock := false
			for _, block := range parsed.Blocks {
				if block.Type == "code" {
					hasCodeBlock = true
					break
				}
			}

			if tc.expectCode && !hasCodeBlock {
				t.Errorf("Expected code block but none found for %s", tc.name)
			}

			if !tc.expectCode && hasCodeBlock && !strings.Contains(tc.content, "```") {
				t.Errorf("Unexpected code block found for %s", tc.name)
			}
		})
	}
}