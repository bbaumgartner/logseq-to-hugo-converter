// This file handles image processing for the blog conversion.
// It copies images from the Logseq assets directory to the Hugo output directory
// and updates image references in the content.
package main

import (
	"fmt"      // Formatted I/O (printing)
	"io"       // Input/Output operations
	"os"       // Operating system functions (file operations)
	"path/filepath" // File path manipulation
	"regexp"   // Regular expressions
)

// ImageProcessor is responsible for handling all image-related operations.
// It processes both inline images and header/featured images.
type ImageProcessor struct {
	inputDir   string         // Directory where input markdown file is located
	outputDir  string         // Directory where processed images should be copied
	assetRegex *regexp.Regexp // Compiled regex to find image references
}

// NewImageProcessor creates a new ImageProcessor instance.
// Parameters:
//   inputDir: The directory containing the source markdown file
//   outputDir: The directory where images should be copied to
// Returns:
//   *ImageProcessor: A pointer to the new processor
func NewImageProcessor(inputDir, outputDir string) *ImageProcessor {
	// Return a pointer to a new ImageProcessor struct
	return &ImageProcessor{
		inputDir:  inputDir,
		outputDir: outputDir,
		// Compile the regex pattern for finding images
		// Pattern breakdown:
		//   !\[(.*?)\]     = Markdown image alt text: ![anything]
		//   \(             = Opening parenthesis
		//   (.*?assets\/)  = Capture path including "assets/"
		//   (.*?)          = Capture the filename
		//   \)             = Closing parenthesis
		// Example match: ![photo](../assets/image.jpg)
		assetRegex: regexp.MustCompile(`!\[(.*?)\]\((.*?assets\/)(.*?)\)`),
	}
}

// ProcessContent processes all images in the content string.
// It finds image references, copies the image files, and updates the references.
// Parameters:
//   content: The markdown content containing image references
// Returns:
//   string: Updated content with simplified image paths
func (p *ImageProcessor) ProcessContent(content string) string {
	// Find all image references in the content
	// FindAllStringSubmatch returns a 2D slice:
	//   - Outer slice: one element per match
	//   - Inner slice: [full match, capture group 1, capture group 2, ...]
	// -1 means find all matches (not just the first)
	matches := p.assetRegex.FindAllStringSubmatch(content, -1)

	// Process each found image
	// range iterates over the slice, _ discards the index
	for _, match := range matches {
		// match[0] = entire match (e.g., "![photo](../assets/image.jpg)")
		// match[1] = alt text (e.g., "photo")
		// match[2] = path to assets (e.g., "../assets/")
		// match[3] = filename (e.g., "image.jpg")
		
		// Build the source path (where the image currently is)
		// filepath.Join combines path parts with the correct separator
		src := filepath.Join(p.inputDir, match[2]+match[3])
		
		// Build the destination path (where to copy the image)
		dst := filepath.Join(p.outputDir, match[3])
		
		// Copy the image file
		p.copyFile(src, dst)
	}

	// Update the content to use simplified paths
	// ReplaceAllString replaces matches with a new pattern
	// $1 and $3 reference capture groups from the original pattern
	// This changes "![alt](../assets/image.jpg)" to "![alt](image.jpg)"
	return p.assetRegex.ReplaceAllString(content, "![$1]($3)")
}

// ProcessHeaderImage copies the header image and renames it to "featured".
// Hugo expects the featured/header image to be named "featured.*"
// Parameters:
//   headerPath: Relative path to the header image (e.g., "../assets/header.jpg")
func (p *ImageProcessor) ProcessHeaderImage(headerPath string) {
	// If no header path is provided, do nothing
	// Empty string check
	if headerPath == "" {
		return // Early return - exit the function
	}

	// Extract just the filename from the path
	// filepath.Base returns the last element of the path
	// e.g., "../assets/photo.jpg" -> "photo.jpg"
	fileName := filepath.Base(headerPath)
	
	// Build the full source path
	src := filepath.Join(p.inputDir, headerPath)
	
	// Get the file extension (e.g., ".jpg", ".png")
	// filepath.Ext returns the extension including the dot
	ext := filepath.Ext(fileName)
	
	// Build destination path with Hugo's expected name: "featured.ext"
	dst := filepath.Join(p.outputDir, "featured"+ext)
	
	// Copy the file
	p.copyFile(src, dst)
}

// copyFile copies a file from source to destination.
// This is a helper method used internally by the processor.
// Parameters:
//   src: Source file path
//   dst: Destination file path
func (p *ImageProcessor) copyFile(src, dst string) {
	// Open the source file for reading
	// os.Open returns a file handle and an error
	in, err := os.Open(src)
	
	// Check if there was an error opening the file
	if err != nil {
		// If the file doesn't exist or can't be opened, print a warning
		// We don't stop the entire conversion for missing images
		fmt.Printf("Warning: Missing image %s\n", src)
		return // Exit this function early
	}
	// defer means "run this when the function exits"
	// This ensures the file is closed even if an error occurs later
	defer in.Close()

	// Create (or overwrite) the destination file
	out, err := os.Create(dst)
	if err != nil {
		// If we can't create the destination file, just return
		// (We could log this error too, but we keep it simple)
		return
	}
	// Ensure the output file is also closed when we're done
	defer out.Close()

	// Copy all data from source to destination
	// io.Copy reads from 'in' and writes to 'out' until EOF
	// We ignore the return values (bytes copied and error)
	// because we're doing basic file copying
	io.Copy(out, in)
	
	// Note: In production code, you might want to check the error from io.Copy
}
