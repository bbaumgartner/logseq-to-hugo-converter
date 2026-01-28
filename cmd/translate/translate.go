// Package main provides a CLI tool for translating Hugo markdown files.
//
// Usage:
//
//	go run translate.go <input_file.md>
//	go run translate.go 2025-09-13_SKS/index.de.md
//
// The program will:
// 1. Parse the input markdown file
// 2. Detect the source language from the filename (e.g., index.de.md → German)
// 3. Translate to all other supported languages (English, Spanish, French, Italian, German)
// 4. Write translated files in the same directory as the input file
//
// Requirements:
// - OPENAI_API_KEY environment variable must be set
// - Input file must be in format: index.<lang>.md (e.g., index.de.md, index.en.md)
// - Input file must have TOML frontmatter (+++...+++)
package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	// Check command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run translate.go <input_file.md>")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  go run translate.go 2025-09-13_SKS/index.de.md")
		fmt.Println()
		fmt.Println("Requirements:")
		fmt.Println("  - OPENAI_API_KEY environment variable must be set")
		fmt.Println("  - Input file must be in format: index.<lang>.md")
		os.Exit(1)
	}

	inputPath := os.Args[1]

	// Verify file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("Error: File not found: %s\n", inputPath)
		os.Exit(1)
	}

	// Parse the input file
	fmt.Printf("📖 Parsing %s...\n", FormatOutputPath(inputPath))
	markdownFile, err := ParseMarkdownFile(inputPath)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	sourceLangName := getLanguageName(markdownFile.SourceLang)
	fmt.Printf("✓ Detected source language: %s\n\n", sourceLangName)

	// Get target languages (all languages except source)
	targetLanguages := GetTargetLanguages(markdownFile.SourceLang)

	if len(targetLanguages) == 0 {
		fmt.Println("No target languages to translate to.")
		os.Exit(0)
	}

	fmt.Printf("🌍 Translating from %s to %d languages...\n", sourceLangName, len(targetLanguages))

	// Create translator
	translator, err := NewTranslator()
	if err != nil {
		fmt.Printf("Error initializing translator: %v\n", err)
		fmt.Println("\nMake sure OPENAI_API_KEY environment variable is set:")
		fmt.Println("  export OPENAI_API_KEY='sk-...'")
		os.Exit(1)
	}

	// Create writer
	writer := NewTranslationWriter(inputPath)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Translate to each target language
	successCount := 0
	for _, targetLang := range targetLanguages {
		translatedFile, err := translator.TranslateMarkdownFile(ctx, markdownFile, targetLang)
		if err != nil {
			fmt.Printf("  ✗ Failed to translate to %s: %v\n", targetLang.Name, err)
			continue
		}

		// Write the translated file
		outputPath, err := writer.WriteTranslation(translatedFile, targetLang.Code)
		if err != nil {
			fmt.Printf("  ✗ Failed to write %s translation: %v\n", targetLang.Name, err)
			continue
		}

		fmt.Printf("  ✓ Created: %s\n", FormatOutputPath(outputPath))
		successCount++
	}

	fmt.Printf("\n✅ Successfully translated to %d/%d languages\n", successCount, len(targetLanguages))

	if successCount < len(targetLanguages) {
		os.Exit(1)
	}
}

// getLanguageName returns the full language name for a language code.
func getLanguageName(code string) string {
	names := map[string]string{
		"en": "English",
		"de": "German",
		"es": "Spanish",
		"fr": "French",
		"it": "Italian",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}
