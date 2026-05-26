package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestChatGPTResponseFromTestData tests ChatGPT response parsing using testdata files
func TestChatGPTResponseFromTestData(t *testing.T) {
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	testDataDir := "testdata/chatgpt/raw"
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
		t.Skip("No ChatGPT test data files found in testdata/chatgpt/raw - add JSON test files")
	}

	t.Logf("Found %d ChatGPT test data files", len(testFiles))

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

			t.Logf("Testing ChatGPT response: %s", testData.Description)
			t.Logf("Content length: %d characters", len(testData.Content))

			parsed, err := service.ParseContent(testData.Content, "openai")
			if err != nil {
				t.Errorf("Failed to parse ChatGPT response: %v", err)
				return
			}

			if parsed == nil {
				t.Error("ParseContent returned nil for ChatGPT response")
				return
			}

			if len(parsed.Blocks) == 0 {
				t.Error("ParseContent returned no blocks for ChatGPT response")
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

// TestChatGPTResponseParsing tests ChatGPT response parsing using actual saved responses
func TestChatGPTResponseParsing(t *testing.T) {
	service := NewContentParserService()
	if service == nil {
		t.Fatal("Failed to create ContentParserService")
	}

	debugDir := "/tmp/chatgpt_responses"
	chatgptFiles := []string{}

	if _, err := os.Stat(debugDir); err == nil {
		err := filepath.WalkDir(debugDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.Contains(d.Name(), "openai_response_") {
				chatgptFiles = append(chatgptFiles, path)
			}
			return nil
		})
		if err != nil {
			t.Logf("Warning: Could not walk debug directory: %v", err)
		}
	}

	if len(chatgptFiles) == 0 {
		t.Skip("No ChatGPT response files found in /tmp/chatgpt_responses - run some ChatGPT conversations first")
	}

	t.Logf("Found %d ChatGPT response files to test", len(chatgptFiles))

	for _, filePath := range chatgptFiles {
		filename := filepath.Base(filePath)
		t.Run(filename, func(t *testing.T) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read ChatGPT response file %s: %v", filePath, err)
			}

			response := string(content)
			if response == "" {
				t.Skip("Empty response file")
			}

			t.Logf("Testing ChatGPT response (length: %d characters)", len(response))
			t.Logf("First 200 chars: %q", func() string {
				if len(response) > 200 {
					return response[:200] + "..."
				}
				return response
			}())

			parsed, err := service.ParseContent(response, "openai")
			if err != nil {
				t.Errorf("Failed to parse ChatGPT response: %v", err)
				return
			}

			if parsed == nil {
				t.Error("ParseContent returned nil for ChatGPT response")
				return
			}

			if len(parsed.Blocks) == 0 {
				t.Error("ParseContent returned no blocks for ChatGPT response")
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

			// Check for potential truncation issues
			if len(response) > 100 && totalContent < len(response)/2 {
				t.Errorf("Potential data loss: original %d chars but parsed only %d chars", len(response), totalContent)
			}
		})
	}
}
