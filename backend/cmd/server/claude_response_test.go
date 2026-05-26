package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type TestData struct {
	Provider           string   `json:"provider"`
	Model              string   `json:"model"`
	Content            string   `json:"content"`
	ExpectedCodeBlocks int      `json:"expectedCodeBlocks"`
	ExpectedLanguages  []string `json:"expectedLanguages"`
	Description        string   `json:"description"`
}

// TestClaudeResponseFromTestData tests Claude response parsing using testdata files
func TestClaudeResponseFromTestData(t *testing.T) {
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	testDataDir := "testdata/claude/raw"
	testFiles := []string{}
	if _, err := os.Stat(testDataDir); err == nil {
		err := filepath.WalkDir(testDataDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
				testFiles = append(testFiles, path)
			}
			return nil
		})
		if err != nil {
			t.Logf("Warning: Could not walk testdata directory: %v", err)
		}
	}

	if len(testFiles) == 0 {
		t.Skip("No Claude test data files found in testdata/claude/raw - add JSON test files")
	}

	t.Logf("Found %d Claude test data files", len(testFiles))

	for _, filePath := range testFiles {
		filename := filepath.Base(filePath)
		t.Run(filename, func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read test data file %s: %v", filePath, err)
			}

			var testData TestData
			if err := json.Unmarshal(content, &testData); err != nil {
				t.Fatalf("Failed to parse test data file %s: %v", filePath, err)
			}

			t.Logf("Testing Claude response: %s", testData.Description)
			t.Logf("Content length: %d characters", len(testData.Content))

			parsed, err := service.ParseContent(testData.Content, "claude")
			if err != nil {
				t.Errorf("Failed to parse Claude response: %v", err)
				return
			}

			if parsed == nil {
				t.Error("ParseContent returned nil for Claude response")
				return
			}

			if len(parsed.Blocks) == 0 {
				t.Error("ParseContent returned no blocks for Claude response")
				return
			}

			codeBlocks := 0
			var detectedLanguages []string

			for _, block := range parsed.Blocks {
				if block.Type == "code" {
					codeBlocks++
					detectedLanguages = append(detectedLanguages, block.Language)
					t.Logf("Code block detected: language=%s, content_length=%d", block.Language, len(block.Content))
				}
			}

			if codeBlocks != testData.ExpectedCodeBlocks {
				t.Errorf("Expected %d code blocks but found %d", testData.ExpectedCodeBlocks, codeBlocks)
			}

			if len(detectedLanguages) != len(testData.ExpectedLanguages) {
				t.Errorf("Expected %d languages but detected %d", len(testData.ExpectedLanguages), len(detectedLanguages))
			} else {
				for i, expectedLang := range testData.ExpectedLanguages {
					if i < len(detectedLanguages) && detectedLanguages[i] != expectedLang {
						t.Errorf("Expected language %s at position %d but got %s", expectedLang, i, detectedLanguages[i])
					}
				}
			}

			t.Logf("Test passed: %d code blocks, languages: %v", codeBlocks, detectedLanguages)
		})
	}
}

// TestClaudeResponseParsing tests Claude response parsing using actual saved responses
func TestClaudeResponseParsing(t *testing.T) {
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	debugDir := "/tmp/chatgpt_responses"
	claudeFiles := []string{}

	if _, err := os.Stat(debugDir); err == nil {
		err := filepath.WalkDir(debugDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.Contains(d.Name(), "anthropic_response_") {
				claudeFiles = append(claudeFiles, path)
			}
			return nil
		})
		if err != nil {
			t.Logf("Warning: Could not walk debug directory: %v", err)
		}
	}

	if len(claudeFiles) == 0 {
		t.Skip("No Claude response files found in /tmp/chatgpt_responses - run some Claude conversations first")
	}

	t.Logf("Found %d Claude response files to test", len(claudeFiles))

	for _, filePath := range claudeFiles {
		filename := filepath.Base(filePath)
		t.Run(filename, func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read Claude response file %s: %v", filePath, err)
			}

			response := string(content)
			if response == "" {
				t.Skip("Empty response file")
			}

			t.Logf("Testing Claude response (length: %d characters)", len(response))
			t.Logf("First 200 chars: %q", func() string {
				if len(response) > 200 {
					return response[:200] + "..."
				}
				return response
			}())

			parsed, err := service.ParseContent(response, "claude")
			if err != nil {
				t.Errorf("Failed to parse Claude response: %v", err)
				return
			}

			if parsed == nil {
				t.Error("ParseContent returned nil for Claude response")
				return
			}

			if len(parsed.Blocks) == 0 {
				t.Error("ParseContent returned no blocks for Claude response")
				return
			}

			codeBlocks := 0
			textBlocks := 0
			totalContent := 0

			for _, block := range parsed.Blocks {
				if block.Type == "code" {
					codeBlocks++
					if block.Language == "" {
						t.Logf("Code block without language detection")
					} else {
						t.Logf("Code block detected: language=%s, content_length=%d", block.Language, len(block.Content))
					}
				} else {
					textBlocks++
				}
				totalContent += len(block.Content)
			}

			t.Logf("Parse results: %d code blocks, %d text blocks, %d total chars", codeBlocks, textBlocks, totalContent)

			if totalContent == 0 {
				t.Error("All blocks are empty - parsing may have failed")
			}

			converted := convertParsedContentToMarkdown(parsed)
			if converted == "" {
				t.Error("Failed to convert parsed content back to markdown")
			}

			if strings.Contains(response, "```") {
				if codeBlocks == 0 {
					t.Error("Original response contains code blocks but none were detected")
				}
			}
		})
	}
}

// TestClaudeResponseStructure tests specific aspects of Claude response structure
func TestClaudeResponseStructure(t *testing.T) {
	service := NewContentParserService()

	// Test cases based on common Claude response patterns
	testCases := []struct {
		name       string
		content    string
		expectCode bool
		expectLang string
	}{
		{
			name: "Claude Python Code",
			content: `Here's a Python function to solve this:

` + "```python" + `
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
` + "```" + `

This function uses recursion to calculate the nth Fibonacci number.`,
			expectCode: true,
			expectLang: "python",
		},
		{
			name: "Claude JavaScript Code",
			content: `Here's the JavaScript implementation:

` + "```javascript" + `
function sortArray(arr) {
    return arr.sort((a, b) => a - b);
}
` + "```" + `

This sorts the array in ascending order.`,
			expectCode: true,
			expectLang: "javascript",
		},
		{
			name: "Claude Multiple Code Blocks",
			content: `Here are both implementations:

Python version:
` + "```python" + `
def greet(name):
    return f"Hello, {name}!"
` + "```" + `

JavaScript version:
` + "```javascript" + `
function greet(name) {
    return ` + "`Hello, ${name}!`" + `;
}
` + "```",
			expectCode: true,
			expectLang: "python",
		},
		{
			name: "Claude Text Only",
			content: `This is a detailed explanation of the algorithm without any code.
The approach involves several steps that need careful consideration.
First, we analyze the input data structure.
Then, we determine the optimal strategy.`,
			expectCode: false,
			expectLang: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := service.ParseContent(tc.content, "claude")
			if err != nil {
				t.Fatalf("ParseContent failed: %v", err)
			}

			if parsed == nil {
				t.Fatal("ParseContent returned nil")
			}

			codeBlocks := 0
			var firstCodeLang string

			for _, block := range parsed.Blocks {
				if block.Type == "code" {
					codeBlocks++
					if firstCodeLang == "" {
						firstCodeLang = block.Language
					}
				}
			}

			if tc.expectCode && codeBlocks == 0 {
				t.Errorf("Expected code blocks but none found")
			}

			if !tc.expectCode && codeBlocks > 0 {
				t.Errorf("Expected no code blocks but found %d", codeBlocks)
			}

			if tc.expectLang != "" && firstCodeLang != tc.expectLang {
				t.Errorf("Expected language %s but got %s", tc.expectLang, firstCodeLang)
			}
		})
	}
}

// BenchmarkClaudeResponseParsing benchmarks Claude response parsing performance
func BenchmarkClaudeResponseParsing(b *testing.B) {
	service := NewContentParserService()

	// Typical Claude response with mixed content
	claudeResponse := `I'll help you implement that function. Here's a solution:

` + "```python" + `
def process_data(data):
    """
    Process the input data and return cleaned results.

    Args:
        data (list): Input data to process

    Returns:
        list: Processed and cleaned data
    """
    if not data:
        return []

    # Remove None values and duplicates
    cleaned = list(set(item for item in data if item is not None))

    # Sort the results
    return sorted(cleaned)
` + "```" + `

This function handles edge cases like empty input and None values. The approach:

1. **Input validation**: Check if data is empty
2. **Cleaning**: Remove None values and duplicates using set comprehension
3. **Sorting**: Return sorted results for consistent output

You can test it like this:

` + "```python" + `
# Example usage
test_data = [3, 1, None, 2, 1, 4, None]
result = process_data(test_data)
print(result)  # Output: [1, 2, 3, 4]
` + "```" + `

The time complexity is O(n log n) due to sorting, and space complexity is O(n) for the set operations.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ParseContent(claudeResponse, "claude")
		if err != nil {
			b.Fatalf("ParseContent failed: %v", err)
		}
	}
}
