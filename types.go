// Package main contains the Logseq to Hugo blog converter.
// This file defines the core data types used throughout the application.
package main

// BlogMeta represents the metadata (information about) of a blog post.
// In Go, a struct is a collection of fields grouped together.
// The fields use uppercase first letters, which makes them "exported" (publicly accessible).
type BlogMeta struct {
	Date    string // Publication date in YYYY-MM-DD format
	Title   string // The title of the blog post
	Author  string // Name of the author
	Header  string // Path to the header/featured image
	Summary string // Short summary or excerpt of the post
	Status  string // Publication status (e.g., "online", "draft")
}

// BlogPost represents a complete blog post with both metadata and content.
// This struct combines the BlogMeta with the actual content blocks.
type BlogPost struct {
	Meta    BlogMeta // The metadata about the post (embedded struct)
	Content []string // A slice (dynamic array) of content blocks/paragraphs
}

// BlogExtractor is an interface that defines how blog posts are extracted.
// In Go, an interface is a contract that defines method signatures.
// Any type that implements all methods in an interface automatically satisfies it.
// This is the Strategy Pattern - different implementations can extract blogs differently.
type BlogExtractor interface {
	// Extract attempts to extract a blog post from a parsed markdown document.
	// Parameters:
	//   doc: The parsed markdown document (interface{} means "any type")
	//   source: The raw markdown content as bytes
	// Returns:
	//   *BlogPost: A pointer to the extracted blog post (nil if not found)
	//   bool: true if a blog post was found, false otherwise
	Extract(doc interface{}, source []byte) (*BlogPost, bool)
}
