package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeminiResponseFromTestData tests Gemini response parsing using testdata files
func TestGeminiResponseFromTestData(t *testing.T) {
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	testDataDir := "testdata/gemini/raw"
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
		t.Skip("No Gemini test data files found in testdata/gemini/raw - add JSON test files")
	}

	t.Logf("Found %d Gemini test data files", len(testFiles))

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

			t.Logf("Testing Gemini response: %s", testData.Description)
			t.Logf("Content length: %d characters", len(testData.Content))

			parsed, err := service.ParseContent(testData.Content, "gemini")
			if err != nil {
				t.Errorf("Failed to parse Gemini response: %v", err)
				return
			}

			if parsed == nil {
				t.Error("ParseContent returned nil for Gemini response")
				return
			}

			if len(parsed.Blocks) == 0 {
				t.Error("ParseContent returned no blocks for Gemini response")
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

// TestGeminiResponseParsing tests Gemini response parsing using actual saved responses
func TestGeminiResponseParsing(t *testing.T) {
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	debugDir := "/tmp/chatgpt_responses"
	geminiFiles := []string{}
	if _, err := os.Stat(debugDir); err == nil {
		err := filepath.WalkDir(debugDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.Contains(d.Name(), "gemini_response_") {
				geminiFiles = append(geminiFiles, path)
			}
			return nil
		})
		if err != nil {
			t.Logf("Warning: Could not walk debug directory: %v", err)
		}
	}

	if len(geminiFiles) == 0 {
		t.Skip("No Gemini response files found in /tmp/chatgpt_responses - run some Gemini conversations first")
	}

	t.Logf("Found %d Gemini response files to test", len(geminiFiles))

	for _, filePath := range geminiFiles {
		filename := filepath.Base(filePath)
		t.Run(filename, func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read Gemini response file %s: %v", filePath, err)
			}

			response := string(content)
			if response == "" {
				t.Skip("Empty response file")
			}

			t.Logf("Testing Gemini response (length: %d characters)", len(response))
			t.Logf("First 200 chars: %q", func() string {
				if len(response) > 200 {
					return response[:200] + "..."
				}
				return response
			}())

			parsed, err := service.ParseContent(response, "gemini")
			if err != nil {
				t.Errorf("Failed to parse Gemini response: %v", err)
				return
			}

			if parsed == nil {
				t.Error("ParseContent returned nil for Gemini response")
				return
			}

			if len(parsed.Blocks) == 0 {
				t.Error("ParseContent returned no blocks for Gemini response")
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

// TestGeminiResponseStructure tests specific aspects of Gemini response structure
func TestGeminiResponseStructure(t *testing.T) {
	service := NewContentParserService()

	// Test cases based on common Gemini response patterns
	testCases := []struct {
		name        string
		content     string
		expectCode  bool
		expectLang  string
	}{
		{
			name: "Gemini Python Code",
			content: `Here's a Python solution for your problem:

` + "```python" + `
def calculate_fibonacci(n):
    """Calculate the nth Fibonacci number."""
    if n <= 1:
        return n
    return calculate_fibonacci(n-1) + calculate_fibonacci(n-2)

# Example usage
print(calculate_fibonacci(10))
` + "```" + `

This recursive implementation calculates Fibonacci numbers efficiently.`,
			expectCode: true,
			expectLang: "python",
		},
		{
			name: "Gemini JavaScript Code",
			content: `Here's how you can implement this in JavaScript:

` + "```javascript" + `
const processArray = (arr) => {
    return arr
        .filter(item => item !== null && item !== undefined)
        .map(item => item.toString().trim())
        .sort();
};

console.log(processArray([3, null, 1, undefined, 2]));
` + "```" + `

This function filters out null/undefined values and sorts the array.`,
			expectCode: true,
			expectLang: "javascript",
		},
		{
			name: "Gemini Multiple Languages",
			content: `I'll show you implementations in both Python and Go:

**Python version:**
` + "```python" + `
def greet_user(name):
    return f"Hello, {name}! Welcome to our application."
` + "```" + `

**Go version:**
` + "```go" + `
package main

import "fmt"

func greetUser(name string) string {
    return fmt.Sprintf("Hello, %s! Welcome to our application.", name)
}
` + "```" + `

Both implementations provide the same functionality.`,
			expectCode: true,
			expectLang: "python", // Should detect first language
		},
		{
			name: "Gemini Text Response",
			content: `Based on your requirements, I recommend the following approach:

1. **Analysis Phase**: Start by examining the existing codebase structure
2. **Planning Phase**: Design the new feature architecture
3. **Implementation Phase**: Build the feature incrementally
4. **Testing Phase**: Ensure comprehensive test coverage
5. **Deployment Phase**: Roll out with proper monitoring

This methodology ensures a systematic and reliable development process.`,
			expectCode: false,
			expectLang: "",
		},
		{
			name: "Gemini JSON Response",
			content: `Here's the configuration you'll need:

` + "```json" + `
{
  "server": {
    "host": "localhost",
    "port": 3000,
    "ssl": false
  },
  "database": {
    "type": "postgresql",
    "host": "localhost",
    "port": 5432,
    "name": "myapp"
  },
  "features": {
    "authentication": true,
    "logging": true,
    "caching": false
  }
}
` + "```" + `

This configuration provides the basic setup for your application.`,
			expectCode: true,
			expectLang: "json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := service.ParseContent(tc.content, "gemini")
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

// BenchmarkGeminiResponseParsing benchmarks Gemini response parsing performance
func BenchmarkGeminiResponseParsing(b *testing.B) {
	service := NewContentParserService()

	// Typical Gemini response with mixed content
	geminiResponse := `I'll help you solve this step by step. Here's a comprehensive solution:

` + "```python" + `
import json
from typing import List, Dict, Any

def process_user_data(users: List[Dict[str, Any]]) -> Dict[str, Any]:
    """
    Process user data and generate summary statistics.

    Args:
        users: List of user dictionaries containing user information

    Returns:
        Dictionary containing processed statistics
    """
    if not users:
        return {"total_users": 0, "active_users": 0, "summary": "No users found"}

    active_users = [user for user in users if user.get('active', False)]

    statistics = {
        "total_users": len(users),
        "active_users": len(active_users),
        "activity_rate": len(active_users) / len(users) * 100,
        "summary": f"Processed {len(users)} users, {len(active_users)} active"
    }

    return statistics

# Example usage
sample_users = [
    {"id": 1, "name": "Alice", "active": True},
    {"id": 2, "name": "Bob", "active": False},
    {"id": 3, "name": "Charlie", "active": True}
]

result = process_user_data(sample_users)
print(json.dumps(result, indent=2))
` + "```" + `

**Key features of this solution:**

1. **Type Hints**: Uses proper type annotations for better code clarity
2. **Error Handling**: Checks for empty input and handles edge cases
3. **Documentation**: Includes comprehensive docstring
4. **Flexibility**: Easy to extend with additional statistics

**Performance considerations:**
- Time complexity: O(n) where n is the number of users
- Space complexity: O(1) for the statistics, O(k) for active users list
- Suitable for datasets up to millions of users

You can test this with different data sets and modify the statistics as needed.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ParseContent(geminiResponse, "gemini")
		if err != nil {
			b.Fatalf("ParseContent failed: %v", err)
		}
	}
}