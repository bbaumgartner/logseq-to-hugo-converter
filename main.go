// Package main is the entry point for the Logseq to Hugo converter application.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run . <input_file.md> <output_directory>")
		return
	}

	inputPath := os.Args[1]
	outputBasePath := os.Args[2]

	// Convert the file
	outputPaths, err := convertFile(inputPath, outputBasePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print success messages
	for _, outputPath := range outputPaths {
		fmt.Printf("Created: %s/index.md\n", outputPath)
	}
}

// convertFile converts a Logseq markdown file to Hugo format.
// It finds all blog posts in the file and converts each one.
func convertFile(inputPath, outputBasePath string) ([]string, error) {
	// Read the input file
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("reading input file: %w", err)
	}

	// Parse the markdown
	doc := goldmark.New().Parser().Parse(text.NewReader(source))

	// Extract all blog posts
	posts := extractBlogPosts(doc, source)
	if len(posts) == 0 {
		return nil, fmt.Errorf("no blog post found with 'type:: blog' marker")
	}

	var outputPaths []string
	inputDir := filepath.Dir(inputPath)

	// Convert each blog post
	for _, post := range posts {
		// Skip non-online posts
		if post.Meta.Status != "online" {
			fmt.Printf("Skipping blog post '%s': status is '%s'\n", post.Meta.Title, post.Meta.Status)
			continue
		}

		// Create output directory
		outputDir := createOutputDir(outputBasePath, post.Meta)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("creating output directory: %w", err)
		}

		// Build content
		content := buildContent(post.Content)

		// Process images and videos
		processor := NewImageProcessor(inputDir, outputDir)
		content = processor.ProcessContent(content)
		processor.ProcessHeaderImage(post.Meta.Header)

		// Write output
		writer := NewHugoWriter(outputDir)
		if err := writer.Write(post.Meta, content); err != nil {
			return nil, err
		}

		outputPaths = append(outputPaths, outputDir)
	}

	return outputPaths, nil
}

// createOutputDir builds the output directory path from metadata.
func createOutputDir(basePath string, meta BlogMeta) string {
	// Replace spaces with underscores in title
	title := strings.ReplaceAll(meta.Title, " ", "_")

	// Format: YYYY-MM-DD_Title
	dirName := fmt.Sprintf("%s_%s", meta.Date, title)
	return filepath.Join(basePath, dirName)
}

// buildContent combines content blocks into a single string.
func buildContent(blocks []string) string {
	var builder strings.Builder
	for _, block := range blocks {
		if cleaned := strings.TrimSpace(block); cleaned != "" {
			builder.WriteString(cleaned)
			builder.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(builder.String())
}
