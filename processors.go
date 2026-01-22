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
	"strings"  // String manipulation for extension checking
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

// ProcessContent processes all images and videos in the content string.
// It finds media references, copies the files, and updates the references.
// Videos are converted to Hugo shortcode format: {{< video src="file.mp4" >}}
// Parameters:
//   content: The markdown content containing media references
// Returns:
//   string: Updated content with simplified paths and video shortcodes
func (p *ImageProcessor) ProcessContent(content string) string {
	// Find all media references in the content
	// FindAllStringSubmatch returns a 2D slice:
	//   - Outer slice: one element per match
	//   - Inner slice: [full match, capture group 1, capture group 2, ...]
	// -1 means find all matches (not just the first)
	matches := p.assetRegex.FindAllStringSubmatch(content, -1)

	// Process each found media file
	// range iterates over the slice, _ discards the index
	for _, match := range matches {
		// match[0] = entire match (e.g., "![photo](../assets/image.jpg)")
		// match[1] = alt text (e.g., "photo")
		// match[2] = path to assets (e.g., "../assets/")
		// match[3] = filename (e.g., "image.jpg")
		
		// Build the source path (where the media file currently is)
		// filepath.Join combines path parts with the correct separator
		src := filepath.Join(p.inputDir, match[2]+match[3])
		
		// Build the destination path (where to copy the media file)
		dst := filepath.Join(p.outputDir, match[3])
		
		// Copy the media file
		p.copyFile(src, dst)
	}

	// Update the content with a custom replacement function
	// This allows us to check each match and decide how to replace it
	result := p.assetRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the parts of this match
		parts := p.assetRegex.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match // If pattern doesn't match, return unchanged
		}
		
		altText := parts[1]  // The alt text
		filename := parts[3]  // The filename
		
		// Check if this is a video file by extension
		if isVideoFile(filename) {
			// Convert to Hugo video shortcode
			// {{< video src="filename.mp4" >}}
			return fmt.Sprintf(`{{< video src="%s" >}}`, filename)
		}
		
		// For images, use simplified markdown syntax
		// "![alt](../assets/image.jpg)" -> "![alt](image.jpg)"
		return fmt.Sprintf("![%s](%s)", altText, filename)
	})
	
	return result
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

// isVideoFile checks if a filename has a video file extension.
// This function determines whether a file should be treated as a video
// and converted to Hugo's video shortcode format.
// Parameters:
//   filename: The name of the file to check
// Returns:
//   bool: true if it's a video file, false otherwise
func isVideoFile(filename string) bool {
	// Get the file extension in lowercase
	// filepath.Ext returns the extension including the dot (e.g., ".mp4")
	// strings.ToLower converts to lowercase for case-insensitive comparison
	ext := strings.ToLower(filepath.Ext(filename))
	
	// List of common video file extensions
	// Check if the extension matches any of these
	videoExtensions := []string{
		".mp4",   // MPEG-4 video
		".mov",   // QuickTime movie
		".avi",   // Audio Video Interleave
		".wmv",   // Windows Media Video
		".flv",   // Flash Video
		".webm",  // WebM video
		".mkv",   // Matroska video
		".m4v",   // MPEG-4 video file
		".mpg",   // MPEG video
		".mpeg",  // MPEG video
	}
	
	// Check if the extension is in our list
	for _, videoExt := range videoExtensions {
		if ext == videoExt {
			return true // It's a video file
		}
	}
	
	// Not a video file
	return false
}
