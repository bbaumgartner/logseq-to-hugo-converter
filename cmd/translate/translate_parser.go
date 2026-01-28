// Package main provides translation functionality for Hugo markdown files.
package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// MarkdownFile represents a parsed Hugo markdown file.
type MarkdownFile struct {
	Frontmatter Frontmatter
	Content     string
	SourceLang  string // e.g., "de", "en"
}

// Frontmatter represents the TOML frontmatter of a Hugo file.
type Frontmatter struct {
	Date    string            `toml:"date"`
	LastMod string            `toml:"lastmod"`
	Draft   bool              `toml:"draft"`
	Title   string            `toml:"title"`
	Summary string            `toml:"summary"`
	Params  map[string]string `toml:"params"`
}

// ParseMarkdownFile reads and parses a Hugo markdown file.
func ParseMarkdownFile(filePath string) (*MarkdownFile, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Split frontmatter and content
	content := string(data)

	// Check for TOML frontmatter (+++...+++)
	if !strings.HasPrefix(content, "+++") {
		return nil, fmt.Errorf("file does not start with TOML frontmatter (+++)")
	}

	// Find the closing +++
	parts := strings.SplitN(content[3:], "+++", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed frontmatter: missing closing +++")
	}

	frontmatterStr := strings.TrimSpace(parts[0])
	markdownContent := strings.TrimSpace(parts[1])

	// Parse TOML frontmatter
	var fm Frontmatter
	if err := toml.Unmarshal([]byte(frontmatterStr), &fm); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	// Detect source language from filename
	sourceLang := detectLanguage(filePath)
	if sourceLang == "" {
		return nil, fmt.Errorf("could not detect language from filename: %s", filePath)
	}

	return &MarkdownFile{
		Frontmatter: fm,
		Content:     markdownContent,
		SourceLang:  sourceLang,
	}, nil
}

// detectLanguage extracts the language code from a filename like "index.de.md"
func detectLanguage(filePath string) string {
	// Supported language codes
	supportedLangs := map[string]bool{
		"en": true,
		"de": true,
		"es": true,
		"fr": true,
		"it": true,
	}

	// Extract just the filename
	parts := strings.Split(filePath, "/")
	filename := parts[len(parts)-1]

	// Look for pattern: index.XX.md
	if strings.HasPrefix(filename, "index.") && strings.HasSuffix(filename, ".md") {
		// Extract the language code (e.g., "de" from "index.de.md")
		langPart := strings.TrimPrefix(filename, "index.")
		langPart = strings.TrimSuffix(langPart, ".md")

		// Validate that it's a supported language
		if supportedLangs[langPart] {
			return langPart
		}
	}

	return ""
}

// SerializeToMarkdown converts the MarkdownFile back to Hugo markdown format.
func (mf *MarkdownFile) SerializeToMarkdown() string {
	var buf bytes.Buffer

	// Write frontmatter
	buf.WriteString("+++\n")

	// Manually format TOML with proper escaping (same as writer.go)
	buf.WriteString(fmt.Sprintf("date = \"%s\"\n", escapeTomlString(mf.Frontmatter.Date)))
	buf.WriteString(fmt.Sprintf("lastmod = \"%s\"\n", escapeTomlString(mf.Frontmatter.LastMod)))
	buf.WriteString(fmt.Sprintf("draft = %t\n", mf.Frontmatter.Draft))
	buf.WriteString(fmt.Sprintf("title = \"%s\"\n", escapeTomlString(mf.Frontmatter.Title)))
	buf.WriteString(fmt.Sprintf("summary = \"%s\"\n", escapeTomlString(mf.Frontmatter.Summary)))

	// Write params section
	if len(mf.Frontmatter.Params) > 0 {
		buf.WriteString("[params]\n")
		for key, value := range mf.Frontmatter.Params {
			buf.WriteString(fmt.Sprintf("  %s = \"%s\"\n", key, escapeTomlString(value)))
		}
	}

	buf.WriteString("+++\n\n")

	// Write content
	buf.WriteString(mf.Content)
	buf.WriteString("\n")

	return buf.String()
}

// escapeTomlString escapes special characters for TOML string values.
// TOML requires double quotes and backslashes to be escaped.
func escapeTomlString(s string) string {
	// First, escape backslashes (must be done first!)
	s = strings.ReplaceAll(s, `\`, `\\`)
	// Then, escape double quotes
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// GetTargetLanguages returns all supported languages except the source language.
func GetTargetLanguages(sourceLang string) []Language {
	allLanguages := []Language{
		{Code: "en", Name: "English"},
		{Code: "de", Name: "German"},
		{Code: "es", Name: "Spanish"},
		{Code: "fr", Name: "French"},
		{Code: "it", Name: "Italian"},
	}

	var targets []Language
	for _, lang := range allLanguages {
		if lang.Code != sourceLang {
			targets = append(targets, lang)
		}
	}

	return targets
}

// Language represents a target language for translation.
type Language struct {
	Code string // e.g., "de", "en"
	Name string // e.g., "German", "English"
}
