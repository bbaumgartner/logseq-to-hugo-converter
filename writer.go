// This file handles writing blog posts in Hugo's expected format.
// Hugo is a static site generator that expects specific file structures
// and front matter (metadata) format.
package main

import (
	"fmt"           // Formatted I/O
	"os"            // Operating system functions
	"path/filepath" // File path manipulation
	"strings"       // String manipulation for escaping
)

// HugoWriter is responsible for writing blog posts in Hugo format.
// Hugo expects:
//   - An index.md file in each post's directory
//   - TOML front matter (between +++ markers) with metadata
//   - Content after the front matter
type HugoWriter struct {
	outputDir string // Directory where the index.md file should be created
}

// NewHugoWriter creates a new HugoWriter instance.
// This is a constructor function that initializes the writer.
// Parameters:
//
//	outputDir: The directory where Hugo files should be written
//
// Returns:
//
//	*HugoWriter: A pointer to the new writer instance
func NewHugoWriter(outputDir string) *HugoWriter {
	// Return a pointer to a new HugoWriter struct
	// The & operator creates a pointer to the struct
	return &HugoWriter{outputDir: outputDir}
}

// getFilename determines the correct filename based on the language.
// Parameters:
//
//	language: The language code from metadata (e.g., "german", "english")
//
// Returns:
//
//	string: The filename to use (e.g., "index.de.md", "index.en.md")
func (w *HugoWriter) getFilename(language string) string {
	// Normalize language to lowercase for case-insensitive comparison
	language = strings.ToLower(strings.TrimSpace(language))

	switch language {
	case "german":
		return "index.de.md"
	case "english":
		return "index.en.md"
	default:
		// Default to German if no language is specified
		return "index.de.md"
	}
}

// Write creates an index file with Hugo-formatted content.
// This method generates the front matter and writes the complete file.
// The filename is determined by the language metadata.
// Parameters:
//
//	meta: BlogMeta struct containing all the metadata
//	content: The processed blog content (markdown text)
//
// Returns:
//
//	filename: The name of the file created (e.g., "index.de.md")
//	error: An error if something went wrong, nil if successful
func (w *HugoWriter) Write(meta BlogMeta, content string) (string, error) {
	// Determine the filename based on the language
	// Default to index.de.md if no language is set
	filename := w.getFilename(meta.Language)

	// Build the full path to the index file
	// filepath.Join combines directory and filename with correct separator
	indexPath := filepath.Join(w.outputDir, filename)

	// Create (or overwrite) the index file
	// os.Create creates a new file or truncates an existing one
	f, err := os.Create(indexPath)

	// Check if file creation failed
	if err != nil {
		// Return a formatted error with context
		// %w wraps the original error, %s is string formatting
		return "", fmt.Errorf("creating %s: %w", filename, err)
	}

	// Defer closing the file until the function exits
	// This ensures the file is always closed, even if an error occurs
	defer f.Close()

	// Build the Hugo front matter in TOML format
	// TOML uses +++ delimiters and key = "value" syntax (with double quotes)
	// We must escape any double quotes in the values with \"
	// fmt.Sprintf formats a string with variables substituted
	// The %s placeholders are replaced with the actual values
	frontMatter := fmt.Sprintf(
		// Each line in this string becomes part of the front matter
		"+++\n"+ // Opening delimiter
			"date = \"%s\"\n"+ // Publication date (double quotes)
			"lastmod = \"%s\"\n"+ // Last modified date (same as date)
			"draft = false\n"+ // Not a draft (published)
			"title = \"%s\"\n"+ // Post title (escaped)
			"summary = \"%s\"\n"+ // Post summary/excerpt (escaped)
			"[params]\n"+ // Custom parameters section
			"  author = \"%s\"\n"+ // Author name (indented under params)
			"+++\n\n", // Closing delimiter + blank line
		escapeTomlString(meta.Date),    // Escape date
		escapeTomlString(meta.Date),    // Escape lastmod
		escapeTomlString(meta.Title),   // Escape title
		escapeTomlString(meta.Summary), // Escape summary
		escapeTomlString(meta.Author),  // Escape author
	)

	// Write the complete file content
	// f.WriteString writes a string to the file
	// We concatenate the front matter, content, and a final newline
	_, err = f.WriteString(frontMatter + content + "\n")

	// Check if writing failed
	if err != nil {
		// Return a formatted error
		return "", fmt.Errorf("writing content: %w", err)
	}

	// Success! Return the filename and nil (no error)
	// In Go, functions often return error as the last return value
	// nil means "no error"
	return filename, nil
}

// escapeTomlString escapes special characters for TOML string values.
// TOML requires double quotes to be escaped with a backslash.
// It also escapes backslashes themselves to avoid ambiguity.
// Parameters:
//
//	s: The string to escape
//
// Returns:
//
//	string: The escaped string safe for TOML
func escapeTomlString(s string) string {
	// First, escape backslashes (must be done first!)
	// If we do this last, we'd escape the backslashes we just added
	s = strings.ReplaceAll(s, `\`, `\\`)

	// Then, escape double quotes
	// \" becomes \\\" in the TOML (backslash + escaped quote)
	s = strings.ReplaceAll(s, `"`, `\"`)

	// Return the escaped string
	return s
}
