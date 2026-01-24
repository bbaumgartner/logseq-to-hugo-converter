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
	
	// Convert the input file (may contain multiple blog posts)
	// os.Args[1] is the input file path
	// := declares a new variable and infers its type
	outputPaths, err := converter.Convert(os.Args[1])
	
	// Check if conversion failed
	if err != nil {
		// Print the error message to the console
		// %v is a placeholder that prints the value in default format
		fmt.Printf("Error: %v\n", err)
		return // Exit the program
	}

	// Success! Print where each blog post was created
	// range iterates over the slice of output paths
	for _, outputPath := range outputPaths {
		fmt.Printf("Created: %s/index.md\n", outputPath)
	}
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
// A single file can contain multiple blog posts, all will be converted.
// This is the main method that orchestrates all the steps:
//   1. Read the input file
//   2. Parse the markdown
//   3. Extract all blog posts using strategies
//   4. Validate and process each post
//   5. Process images for each post
//   6. Write output for each post
// Parameters:
//   inputPath: Path to the Logseq markdown file
// Returns:
//   []string: Slice of paths to created output directories
//   error: An error if something went wrong, nil if successful
func (c *BlogConverter) Convert(inputPath string) ([]string, error) {
	// Step 1: Read the entire input file into memory
	// os.ReadFile reads a file and returns its contents as bytes
	source, err := os.ReadFile(inputPath)
	if err != nil {
		// If reading fails, wrap the error with context and return it
		// %w wraps the original error so it can be unwrapped later
		return nil, fmt.Errorf("reading input file: %w", err)
	}

	// Step 2: Parse the markdown into an Abstract Syntax Tree (AST)
	// goldmark.New() creates a new markdown parser
	// .Parser() gets the parser component
	// .Parse() converts the text into an AST
	doc := goldmark.New().Parser().Parse(text.NewReader(source))

	// Step 3: Extract all blog posts using our strategies
	posts := c.extractBlogPosts(doc, source)
	if len(posts) == 0 {
		return nil, fmt.Errorf("no blog post found with 'type:: blog' marker")
	}

	// Slice to collect all output directory paths
	var outputDirs []string

	// Get the directory containing the input file (for resolving relative paths)
	inputDir := filepath.Dir(inputPath)

	// Step 4-8: Process each blog post
	for _, post := range posts {
		// Step 4: Validate that the post status is "online"
		// We only convert posts marked as online, not drafts
		if post.Meta.Status != "online" {
			// Skip this post, but continue with others
			fmt.Printf("Skipping blog post '%s': status is '%s', only 'online' posts are converted\n", 
				post.Meta.Title, post.Meta.Status)
			continue
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
			return nil, fmt.Errorf("creating output directory: %w", err)
		}

		// Step 6: Build the content from content blocks
		content := c.buildContent(post.Content)
		
		// Step 7: Process images
		// Create an image processor for this post
		processor := NewImageProcessor(inputDir, outputDir)
		
		// Process all images in the content (copies files, updates references)
		content = processor.ProcessContent(content)
		
		// Process the header/featured image
		processor.ProcessHeaderImage(post.Meta.Header)

		// Step 8: Write the Hugo-formatted output
		writer := NewHugoWriter(outputDir)
		if err := writer.Write(post.Meta, content); err != nil {
			return nil, err // Return error if writing fails
		}

		// Add this output directory to our results
		outputDirs = append(outputDirs, outputDir)
	}

	// Success! Return all output directory paths
	return outputDirs, nil
}

// extractBlogPosts tries each extraction strategy and collects all found blog posts.
// This implements the Strategy Pattern - we try multiple strategies
// and collect posts from all strategies that find any.
// Parameters:
//   doc: The parsed markdown AST
//   source: The raw markdown content
// Returns:
//   []*BlogPost: Slice of all extracted blog posts (may be empty)
func (c *BlogConverter) extractBlogPosts(doc interface{}, source []byte) []*BlogPost {
	// Slice to collect all found blog posts
	var allPosts []*BlogPost

	// Try each extractor in order
	// range loops over slices, returning index and value
	// _ discards the index since we don't need it
	for _, extractor := range c.extractors {
		// Try this extraction strategy
		// Each extractor returns a slice of posts it found
		posts := extractor.Extract(doc, source)
		
		// If this strategy found any posts, add them to our collection
		if len(posts) > 0 {
			allPosts = append(allPosts, posts...)
			// Don't break - continue trying other strategies
			// This allows mixing formats if needed
		}
	}
	
	// Return all found posts (may be empty)
	return allPosts
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
// Note: If the file contains multiple blog posts, only the first one's path is returned.
// Parameters:
//   inputPath: Path to the Logseq markdown file
//   outputPath: Directory where output should be written
// Returns:
//   string: Path to the output directory (first post if multiple)
//   error: An error if conversion failed
func convertLogseqToHugo(inputPath, outputPath string) (string, error) {
	// Read and parse the file first to check status before calling Convert
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("reading input file: %w", err)
	}

	doc := goldmark.New().Parser().Parse(text.NewReader(source))
	converter := NewBlogConverter(outputPath)
	posts := converter.extractBlogPosts(doc, source)
	
	if len(posts) == 0 {
		return "", fmt.Errorf("no blog post found with 'type:: blog' marker")
	}
	
	// Check if the first post has status != "online" for backward compatibility
	if posts[0].Meta.Status != "online" {
		return "", fmt.Errorf("blog post status is '%s', only 'online' posts are converted", posts[0].Meta.Status)
	}
	
	// Now do the actual conversion
	outputPaths, err := converter.Convert(inputPath)
	if err != nil {
		return "", err
	}
	
	// Return the first output path for backward compatibility
	if len(outputPaths) > 0 {
		return outputPaths[0], nil
	}
	
	// This shouldn't happen if there's no error, but handle it anyway
	return "", fmt.Errorf("conversion succeeded but no output was generated")
}
