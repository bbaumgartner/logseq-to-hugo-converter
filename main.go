// Package main is the entry point for the Logseq to Hugo converter application.
// This file contains the main function and the BlogConverter orchestrator that
// coordinates all the conversion steps.
package main

import (
	"fmt"    // Formatted I/O (printing to console)
	"os"     // Operating system functions (command-line args, file operations)
	"path/filepath" // File path manipulation
	"strings" // String manipulation functions

	"github.com/yuin/goldmark" // Markdown parser library
	"github.com/yuin/goldmark/text" // Text reader for goldmark
)

// ═══════════════════════════════════════════════════════════════════════════
// MAIN ENTRY POINT
// ═══════════════════════════════════════════════════════════════════════════

// main is the entry point of the program.
// This function is automatically called when the program starts.
// In Go, every executable program must have exactly one main function.
func main() {
	// Check if the user provided enough command-line arguments
	// os.Args is a slice containing the command-line arguments
	//   os.Args[0] = program name
	//   os.Args[1] = first argument (input file)
	//   os.Args[2] = second argument (output directory)
	// len() returns the length of a slice
	if len(os.Args) < 3 {
		// Not enough arguments, print usage instructions
		fmt.Println("Usage: go run main.go <input_file.md> <output_directory>")
		return // Exit the function (and program)
	}

	// Create a new blog converter
	// os.Args[2] is the output directory path
	converter := NewBlogConverter(os.Args[2])
	
	// Convert the input file
	// os.Args[1] is the input file path
	// := declares a new variable and infers its type
	outputPath, err := converter.Convert(os.Args[1])
	
	// Check if conversion failed
	if err != nil {
		// Print the error message to the console
		// %v is a placeholder that prints the value in default format
		fmt.Printf("Error: %v\n", err)
		return // Exit the program
	}

	// Success! Print where the file was created
	fmt.Printf("Created: %s/index.md\n", outputPath)
}

// ═══════════════════════════════════════════════════════════════════════════
// BLOG CONVERTER (Orchestrator)
// ═══════════════════════════════════════════════════════════════════════════

// BlogConverter is the main orchestrator that coordinates the entire conversion process.
// It uses the Strategy Pattern to try different extraction methods and manages
// the overall workflow from reading input to writing output.
type BlogConverter struct {
	extractors     []BlogExtractor // Slice of extraction strategies to try
	outputBasePath string          // Base directory for output files
}

// NewBlogConverter creates a new BlogConverter instance.
// This is a constructor function that sets up the converter with all available
// extraction strategies.
// Parameters:
//   outputBasePath: The directory where converted blogs should be written
// Returns:
//   *BlogConverter: A pointer to the new converter
func NewBlogConverter(outputBasePath string) *BlogConverter {
	// Return a pointer to a new BlogConverter
	return &BlogConverter{
		// Initialize the extractors slice with our two strategies
		// []BlogExtractor{...} creates a slice of BlogExtractor interface
		extractors: []BlogExtractor{
			NewNestedListExtractor(),       // Strategy 1: Journal format
			NewTopLevelMetadataExtractor(), // Strategy 2: Pages format
		},
		outputBasePath: outputBasePath,
	}
}

// Convert performs the complete conversion of a Logseq markdown file to Hugo format.
// This is the main method that orchestrates all the steps:
//   1. Read the input file
//   2. Parse the markdown
//   3. Extract blog post using strategies
//   4. Validate the post
//   5. Process images
//   6. Write output
// Parameters:
//   inputPath: Path to the Logseq markdown file
// Returns:
//   string: Path to the created output directory
//   error: An error if something went wrong, nil if successful
func (c *BlogConverter) Convert(inputPath string) (string, error) {
	// Step 1: Read the entire input file into memory
	// os.ReadFile reads a file and returns its contents as bytes
	source, err := os.ReadFile(inputPath)
	if err != nil {
		// If reading fails, wrap the error with context and return it
		// %w wraps the original error so it can be unwrapped later
		return "", fmt.Errorf("reading input file: %w", err)
	}

	// Step 2: Parse the markdown into an Abstract Syntax Tree (AST)
	// goldmark.New() creates a new markdown parser
	// .Parser() gets the parser component
	// .Parse() converts the text into an AST
	doc := goldmark.New().Parser().Parse(text.NewReader(source))

	// Step 3: Extract the blog post using our strategies
	post, err := c.extractBlogPost(doc, source)
	if err != nil {
		return "", err // Return the error if extraction failed
	}

	// Step 4: Validate that the post status is "online"
	// We only convert posts marked as online, not drafts
	if post.Meta.Status != "online" {
		// Return an error explaining why we're not converting this post
		return "", fmt.Errorf("blog post status is '%s', only 'online' posts are converted", post.Meta.Status)
	}

	// Step 5: Create the output directory
	// The directory name is based on the date and title
	outputDir := c.createOutputDir(post.Meta)
	
	// os.MkdirAll creates the directory and all parent directories
	// 0755 is the permission mode (rwxr-xr-x)
	//   Owner: read, write, execute
	//   Group: read, execute
	//   Others: read, execute
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Step 6: Build the content from content blocks
	content := c.buildContent(post.Content)
	
	// Step 7: Process images
	// Get the directory containing the input file (for resolving relative paths)
	inputDir := filepath.Dir(inputPath)
	
	// Create an image processor
	processor := NewImageProcessor(inputDir, outputDir)
	
	// Process all images in the content (copies files, updates references)
	content = processor.ProcessContent(content)
	
	// Process the header/featured image
	processor.ProcessHeaderImage(post.Meta.Header)

	// Step 8: Write the Hugo-formatted output
	writer := NewHugoWriter(outputDir)
	if err := writer.Write(post.Meta, content); err != nil {
		return "", err // Return error if writing fails
	}

	// Success! Return the output directory path
	return outputDir, nil
}

// extractBlogPost tries each extraction strategy until one succeeds.
// This implements the Strategy Pattern - we try multiple strategies
// until we find one that works.
// Parameters:
//   doc: The parsed markdown AST
//   source: The raw markdown content
// Returns:
//   *BlogPost: The extracted blog post
//   error: An error if no strategy succeeded
func (c *BlogConverter) extractBlogPost(doc interface{}, source []byte) (*BlogPost, error) {
	// Try each extractor in order
	// range loops over slices, returning index and value
	// _ discards the index since we don't need it
	for _, extractor := range c.extractors {
		// Try this extraction strategy
		// Each extractor returns the post (or nil) and whether it found one
		if post, found := extractor.Extract(doc, source); found {
			// This strategy worked! Return the post
			return post, nil
		}
	}
	
	// None of the strategies found a blog post
	// Return nil for the post and an error message
	return nil, fmt.Errorf("no blog post found with 'type:: blog' marker")
}

// createOutputDir builds the output directory path from metadata.
// Hugo expects directories named like: "2026-01-17_Title_With_Underscores"
// Parameters:
//   meta: The blog metadata containing date and title
// Returns:
//   string: The full path to the output directory
func (c *BlogConverter) createOutputDir(meta BlogMeta) string {
	// Build the folder name from date and title
	// %s is a string placeholder
	// strings.ReplaceAll replaces all spaces with underscores
	folderName := fmt.Sprintf("%s_%s", meta.Date, strings.ReplaceAll(meta.Title, " ", "_"))
	
	// Combine the base output path with the folder name
	// filepath.Join uses the correct path separator for the OS
	return filepath.Join(c.outputBasePath, folderName)
}

// buildContent combines content blocks into a single string.
// It cleans up whitespace and joins blocks with blank lines.
// Parameters:
//   blocks: Slice of content strings (paragraphs/sections)
// Returns:
//   string: The combined and cleaned content
func (c *BlogConverter) buildContent(blocks []string) string {
	// strings.Builder is efficient for building strings
	// It's much faster than concatenating with + in a loop
	var builder strings.Builder
	
	// Process each content block
	for _, block := range blocks {
		// strings.TrimSpace removes leading and trailing whitespace
		if cleaned := strings.TrimSpace(block); cleaned != "" {
			// If the block has content, add it to the builder
			builder.WriteString(cleaned)
			builder.WriteString("\n\n") // Add two newlines (blank line)
		}
	}
	
	// Convert to string and trim any trailing whitespace
	return strings.TrimSpace(builder.String())
}

// ═══════════════════════════════════════════════════════════════════════════
// BACKWARD COMPATIBILITY
// ═══════════════════════════════════════════════════════════════════════════

// convertLogseqToHugo provides backward compatibility with existing tests.
// This function was the original API and is kept to avoid breaking tests.
// New code should use NewBlogConverter().Convert() instead.
// Parameters:
//   inputPath: Path to the Logseq markdown file
//   outputPath: Directory where output should be written
// Returns:
//   string: Path to the output directory
//   error: An error if conversion failed
func convertLogseqToHugo(inputPath, outputPath string) (string, error) {
	// Create a new converter and run the conversion
	// This is just a wrapper around the new API
	return NewBlogConverter(outputPath).Convert(inputPath)
}
