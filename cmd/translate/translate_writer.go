// Package main provides file writing functionality for translated markdown.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TranslationWriter handles writing translated markdown files.
type TranslationWriter struct {
	inputPath string
}

// NewTranslationWriter creates a new TranslationWriter.
func NewTranslationWriter(inputPath string) *TranslationWriter {
	return &TranslationWriter{
		inputPath: inputPath,
	}
}

// WriteTranslation writes a translated markdown file to disk.
// It places the file in the same directory as the input file.
func (w *TranslationWriter) WriteTranslation(mf *MarkdownFile, targetLang string) (string, error) {
	// Get the directory of the input file
	dir := filepath.Dir(w.inputPath)

	// Create the output filename (e.g., index.es.md for Spanish)
	outputFilename := fmt.Sprintf("index.%s.md", targetLang)
	outputPath := filepath.Join(dir, outputFilename)

	// Serialize the markdown file
	content := mf.SerializeToMarkdown()

	// Write to file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing file %s: %w", outputPath, err)
	}

	return outputPath, nil
}

// GetOutputPath returns the expected output path for a given language code.
func (w *TranslationWriter) GetOutputPath(langCode string) string {
	dir := filepath.Dir(w.inputPath)
	outputFilename := fmt.Sprintf("index.%s.md", langCode)
	return filepath.Join(dir, outputFilename)
}

// GetRelativePath returns a relative path from the current directory if possible.
func GetRelativePath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}

	relPath, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	return relPath
}

// FormatOutputPath formats a path for display, showing relative path if possible.
func FormatOutputPath(path string) string {
	relPath := GetRelativePath(path)
	// If the relative path is shorter and doesn't start with many "..", use it
	if len(relPath) < len(path) && !strings.HasPrefix(relPath, "../..") {
		return relPath
	}
	return path
}
