package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDetectLanguage tests language detection from filenames
func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"German file", "index.de.md", "de"},
		{"English file", "index.en.md", "en"},
		{"Spanish file", "index.es.md", "es"},
		{"French file", "index.fr.md", "fr"},
		{"Italian file", "index.it.md", "it"},
		{"With path", "/path/to/blog/index.de.md", "de"},
		{"Invalid format", "blog.md", ""},
		{"Invalid format 2", "index.md", ""},
		{"Wrong extension", "index.de.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLanguage(tt.filename)
			if got != tt.want {
				t.Errorf("detectLanguage(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

// TestGetTargetLanguages tests getting target languages excluding source
func TestGetTargetLanguages(t *testing.T) {
	tests := []struct {
		name       string
		sourceLang string
		wantCount  int
		wantCodes  []string
	}{
		{
			name:       "Source is German",
			sourceLang: "de",
			wantCount:  4,
			wantCodes:  []string{"en", "es", "fr", "it"},
		},
		{
			name:       "Source is English",
			sourceLang: "en",
			wantCount:  4,
			wantCodes:  []string{"de", "es", "fr", "it"},
		},
		{
			name:       "Source is Spanish",
			sourceLang: "es",
			wantCount:  4,
			wantCodes:  []string{"en", "de", "fr", "it"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTargetLanguages(tt.sourceLang)

			if len(got) != tt.wantCount {
				t.Errorf("GetTargetLanguages(%q) returned %d languages, want %d",
					tt.sourceLang, len(got), tt.wantCount)
			}

			// Check that source language is not in the results
			for _, lang := range got {
				if lang.Code == tt.sourceLang {
					t.Errorf("GetTargetLanguages(%q) includes source language", tt.sourceLang)
				}
			}

			// Check that all expected codes are present
			gotCodes := make(map[string]bool)
			for _, lang := range got {
				gotCodes[lang.Code] = true
			}

			for _, wantCode := range tt.wantCodes {
				if !gotCodes[wantCode] {
					t.Errorf("GetTargetLanguages(%q) missing language code %q",
						tt.sourceLang, wantCode)
				}
			}
		})
	}
}

// TestExtractFirstParagraph tests first paragraph extraction
func TestExtractFirstParagraph(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "Simple paragraph",
			content: "This is the first paragraph.\n\nThis is the second paragraph.",
			want:    "This is the first paragraph.",
		},
		{
			name:    "Paragraph with multiple lines",
			content: "First line.\nSecond line.\nThird line.\n\nNew paragraph.",
			want:    "First line. Second line. Third line.",
		},
		{
			name:    "Paragraph before heading",
			content: "First paragraph.\n\n## Heading\n\nOther content.",
			want:    "First paragraph.",
		},
		{
			name:    "Stop at heading without blank line",
			content: "First paragraph.\n## Heading",
			want:    "First paragraph.",
		},
		{
			name:    "Leading empty lines",
			content: "\n\nFirst paragraph.\n\nSecond paragraph.",
			want:    "First paragraph.",
		},
		{
			name:    "With horizontal rule",
			content: "First paragraph.\n---\nAfter rule.",
			want:    "First paragraph.",
		},
		{
			name:    "Single line",
			content: "Only one line.",
			want:    "Only one line.",
		},
		{
			name:    "Empty content",
			content: "",
			want:    "",
		},
		{
			name:    "Only whitespace",
			content: "   \n\n   ",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstParagraph(tt.content)
			if got != tt.want {
				t.Errorf("extractFirstParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestEscapeTomlString tests TOML string escaping
func TestEscapeTomlString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "No special characters",
			input: "Simple text",
			want:  "Simple text",
		},
		{
			name:  "Double quotes",
			input: `Text with "quotes"`,
			want:  `Text with \"quotes\"`,
		},
		{
			name:  "Backslashes",
			input: `Text with \backslash`,
			want:  `Text with \\backslash`,
		},
		{
			name:  "Both quotes and backslashes",
			input: `Text with "quotes" and \backslash`,
			want:  `Text with \"quotes\" and \\backslash`,
		},
		{
			name:  "Dialog with quotes",
			input: `She said "Hello, world!"`,
			want:  `She said \"Hello, world!\"`,
		},
		{
			name:  "Path with backslashes",
			input: `C:\Users\Name\File.txt`,
			want:  `C:\\Users\\Name\\File.txt`,
		},
		{
			name:  "Mixed special chars",
			input: `He wrote "\n" for newline`,
			want:  `He wrote \"\\n\" for newline`,
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeTomlString(tt.input)
			if got != tt.want {
				t.Errorf("escapeTomlString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseMarkdownFile tests parsing of markdown files
func TestParseMarkdownFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		filename    string
		content     string
		wantErr     bool
		wantLang    string
		wantTitle   string
		wantSummary string
	}{
		{
			name:     "Valid German file",
			filename: "index.de.md",
			content: `+++
date = "2025-01-20"
lastmod = "2025-01-20"
draft = false
title = "Test Titel"
summary = "Test Zusammenfassung"
[params]
  author = "TestAuthor"
+++

This is the content of the blog post.

## Section

More content here.`,
			wantErr:     false,
			wantLang:    "de",
			wantTitle:   "Test Titel",
			wantSummary: "Test Zusammenfassung",
		},
		{
			name:     "Valid English file",
			filename: "index.en.md",
			content: `+++
date = "2025-01-20"
lastmod = "2025-01-20"
draft = true
title = "Test Title"
summary = "Test Summary"
[params]
  author = "TestAuthor"
+++

Content goes here.`,
			wantErr:     false,
			wantLang:    "en",
			wantTitle:   "Test Title",
			wantSummary: "Test Summary",
		},
		{
			name:     "Missing closing +++",
			filename: "index.de.md",
			content: `+++
date = "2025-01-20"
title = "Test"

Content without closing marker.`,
			wantErr: true,
		},
		{
			name:     "No frontmatter",
			filename: "index.de.md",
			content:  `Just content without frontmatter.`,
			wantErr:  true,
		},
		{
			name:     "Invalid filename",
			filename: "blog.md",
			content: `+++
title = "Test"
+++
Content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testPath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(testPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Parse the file
			got, err := ParseMarkdownFile(testPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMarkdownFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMarkdownFile() unexpected error: %v", err)
				return
			}

			// Check results
			if got.SourceLang != tt.wantLang {
				t.Errorf("SourceLang = %q, want %q", got.SourceLang, tt.wantLang)
			}
			if got.Frontmatter.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Frontmatter.Title, tt.wantTitle)
			}
			if got.Frontmatter.Summary != tt.wantSummary {
				t.Errorf("Summary = %q, want %q", got.Frontmatter.Summary, tt.wantSummary)
			}
		})
	}
}

// TestSerializeToMarkdown tests markdown serialization
func TestSerializeToMarkdown(t *testing.T) {
	mf := &MarkdownFile{
		Frontmatter: Frontmatter{
			Date:    "2025-01-20",
			LastMod: "2025-01-20",
			Draft:   false,
			Title:   "Test Title",
			Summary: "Test Summary",
			Params: map[string]string{
				"author": "TestAuthor",
			},
		},
		Content:    "This is the content.\n\n## Section\n\nMore content.",
		SourceLang: "en",
	}

	result := mf.SerializeToMarkdown()

	// Check that it contains the expected components
	expectedParts := []string{
		"+++",
		`date = "2025-01-20"`,
		`lastmod = "2025-01-20"`,
		"draft = false",
		`title = "Test Title"`,
		`summary = "Test Summary"`,
		"[params]",
		`author = "TestAuthor"`,
		"This is the content.",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("SerializeToMarkdown() missing expected part: %q", part)
		}
	}

	// Check structure
	if !strings.HasPrefix(result, "+++\n") {
		t.Error("SerializeToMarkdown() should start with +++")
	}

	// Count +++ markers (should be exactly 2)
	count := strings.Count(result, "+++")
	if count != 2 {
		t.Errorf("SerializeToMarkdown() has %d +++ markers, want 2", count)
	}
}

// TestSerializeToMarkdownWithEscaping tests that special characters are escaped
func TestSerializeToMarkdownWithEscaping(t *testing.T) {
	mf := &MarkdownFile{
		Frontmatter: Frontmatter{
			Date:    "2025-01-20",
			LastMod: "2025-01-20",
			Draft:   false,
			Title:   `Title with "quotes"`,
			Summary: `Summary with "quotes" and \backslash`,
			Params: map[string]string{
				"author": `Author "Name"`,
			},
		},
		Content:    "Content",
		SourceLang: "en",
	}

	result := mf.SerializeToMarkdown()

	// Check that quotes are escaped
	if !strings.Contains(result, `title = "Title with \"quotes\""`) {
		t.Error("SerializeToMarkdown() did not escape quotes in title")
	}

	if !strings.Contains(result, `summary = "Summary with \"quotes\" and \\backslash"`) {
		t.Error("SerializeToMarkdown() did not escape special chars in summary")
	}

	if !strings.Contains(result, `author = "Author \"Name\""`) {
		t.Error("SerializeToMarkdown() did not escape quotes in author")
	}
}

// TestGetTranslationDisclaimer tests disclaimer generation
func TestGetTranslationDisclaimer(t *testing.T) {
	tests := []struct {
		name         string
		targetLang   string
		sourceLang   string
		wantContains []string
		wantLink     string
	}{
		{
			name:       "English disclaimer from German",
			targetLang: "en",
			sourceLang: "de",
			wantContains: []string{
				"---",
				"automatically translated",
				"Large Language Model",
				"original blog post",
			},
			wantLink: "index.de.md",
		},
		{
			name:       "German disclaimer from English",
			targetLang: "de",
			sourceLang: "en",
			wantContains: []string{
				"---",
				"automatisch",
				"Large Language Model",
				"originalen Blogbeitrag",
			},
			wantLink: "index.en.md",
		},
		{
			name:       "Spanish disclaimer",
			targetLang: "es",
			sourceLang: "en",
			wantContains: []string{
				"---",
				"traducida automáticamente",
				"Large Language Model",
				"publicación original",
			},
			wantLink: "index.en.md",
		},
		{
			name:       "French disclaimer",
			targetLang: "fr",
			sourceLang: "de",
			wantContains: []string{
				"---",
				"traduit automatiquement",
				"Large Language Model",
				"article original",
			},
			wantLink: "index.de.md",
		},
		{
			name:       "Italian disclaimer",
			targetLang: "it",
			sourceLang: "en",
			wantContains: []string{
				"---",
				"tradotto automaticamente",
				"Large Language Model",
				"post originale",
			},
			wantLink: "index.en.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTranslationDisclaimer(tt.targetLang, tt.sourceLang)

			// Check that all expected strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("getTranslationDisclaimer() missing expected text %q in result:\n%s",
						want, got)
				}
			}

			// Check that the correct link is present
			if !strings.Contains(got, tt.wantLink) {
				t.Errorf("getTranslationDisclaimer() missing expected link %q in result:\n%s",
					tt.wantLink, got)
			}

			// Check that it starts with ---
			if !strings.HasPrefix(got, "---") {
				t.Errorf("getTranslationDisclaimer() should start with ---")
			}

			// Check that it contains markdown link syntax
			if !strings.Contains(got, "](") || !strings.Contains(got, "[") {
				t.Errorf("getTranslationDisclaimer() should contain markdown link syntax")
			}
		})
	}
}

// TestRoundTrip tests parsing and serialization round-trip
func TestRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	originalContent := `+++
date = "2025-01-20"
lastmod = "2025-01-20"
draft = false
title = "Test Title"
summary = "Test Summary"
[params]
  author = "TestAuthor"
+++

This is the content.

## Section

More content here.`

	// Write original file
	testPath := filepath.Join(tmpDir, "index.en.md")
	err := os.WriteFile(testPath, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse
	parsed, err := ParseMarkdownFile(testPath)
	if err != nil {
		t.Fatalf("ParseMarkdownFile() error: %v", err)
	}

	// Serialize
	serialized := parsed.SerializeToMarkdown()

	// Parse again
	testPath2 := filepath.Join(tmpDir, "index.de.md")
	err = os.WriteFile(testPath2, []byte(serialized), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	parsed2, err := ParseMarkdownFile(testPath2)
	if err != nil {
		t.Fatalf("Second ParseMarkdownFile() error: %v", err)
	}

	// Compare parsed structures
	if parsed.Frontmatter.Date != parsed2.Frontmatter.Date {
		t.Errorf("Date mismatch after round-trip")
	}
	if parsed.Frontmatter.Title != parsed2.Frontmatter.Title {
		t.Errorf("Title mismatch after round-trip")
	}
	if parsed.Frontmatter.Summary != parsed2.Frontmatter.Summary {
		t.Errorf("Summary mismatch after round-trip")
	}
	if strings.TrimSpace(parsed.Content) != strings.TrimSpace(parsed2.Content) {
		t.Errorf("Content mismatch after round-trip")
	}
}
