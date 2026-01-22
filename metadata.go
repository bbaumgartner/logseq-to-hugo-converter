// This file handles parsing of metadata from Logseq markdown files.
// Metadata in Logseq is written as "key:: value" pairs.
package main

import (
	"regexp" // Regular expressions package for pattern matching
	"strings" // String manipulation functions
)

// MetadataParser is responsible for parsing metadata lines and converting them
// into a BlogMeta struct. It uses regular expressions to extract key-value pairs.
type MetadataParser struct {
	regex *regexp.Regexp // Compiled regular expression pattern (pointer to avoid copying)
}

// NewMetadataParser creates and returns a new instance of MetadataParser.
// In Go, constructor functions typically start with "New" and return a pointer.
// The pointer (*) allows us to modify the original object, not a copy.
func NewMetadataParser() *MetadataParser {
	// Return a pointer to a new MetadataParser struct
	// The & operator gets the memory address (pointer) of the struct
	return &MetadataParser{
		// Compile the regex pattern once for better performance
		// Pattern: (\w+)::\s*(.*)
		//   (\w+) = capture one or more word characters (the key)
		//   ::    = literal double colons
		//   \s*   = zero or more whitespace characters
		//   (.*) = capture everything else (the value)
		regex: regexp.MustCompile(`(\w+)::\s*(.*)`),
	}
}

// Parse extracts metadata from an array of lines and returns a BlogMeta struct.
// The receiver (p *MetadataParser) means this is a method on MetadataParser.
// The * makes it a pointer receiver, so we work with the original, not a copy.
func (p *MetadataParser) Parse(lines []string) BlogMeta {
	// Create an empty BlogMeta struct to fill with parsed data
	// := is short variable declaration (type is inferred)
	meta := BlogMeta{}
	
	// Loop through each line in the input slice
	// range returns index and value for each element
	// _ (underscore) discards the index since we don't need it
	for _, line := range lines {
		// Try to match the regex pattern against the line
		// FindStringSubmatch returns an array of matches
		// match[0] = entire match, match[1] = first capture group, etc.
		if match := p.regex.FindStringSubmatch(line); match != nil {
			// nil means no match; if not nil, we found metadata
			key := match[1]                  // First capture group (the key)
			value := strings.TrimSpace(match[2]) // Second capture group (the value), trimmed
			
			// Set the appropriate field in the meta struct
			p.setField(&meta, key, value) // &meta passes a pointer to meta
		}
	}
	
	// Return the completed metadata struct
	return meta
}

// setField sets a specific field in the BlogMeta struct based on the key name.
// This is a private method (lowercase first letter) only used internally.
// Parameters:
//   meta: pointer to the BlogMeta struct to modify
//   key: the field name (e.g., "date", "title")
//   value: the value to set
func (p *MetadataParser) setField(meta *BlogMeta, key, value string) {
	// Switch statement checks the key and sets the appropriate field
	// In Go, switch doesn't need break statements - it exits after one match
	switch key {
	case "date":
		meta.Date = value // Set the Date field
	case "title":
		meta.Title = value // Set the Title field
	case "author":
		meta.Author = value // Set the Author field
	case "header":
		// Header contains image syntax, extract just the path
		meta.Header = extractPath(value)
	case "status":
		meta.Status = value // Set the Status field (e.g., "online")
	// If the key doesn't match any case, do nothing (ignore it)
	}
}

// extractPath extracts a file path from markdown image syntax.
// For example: "![image](path/to/file.jpg)" returns "path/to/file.jpg"
// This is a standalone function (not a method) because it doesn't need parser state.
func extractPath(raw string) string {
	// Regex pattern to find text inside parentheses
	// \( and \) are escaped parentheses (literal characters)
	// (.*?) captures everything inside (non-greedy)
	re := regexp.MustCompile(`\((.*?)\)`)
	
	// Try to find a match
	if match := re.FindStringSubmatch(raw); len(match) > 1 {
		// match[0] = entire match including parentheses
		// match[1] = captured text inside parentheses
		return match[1] // Return the path
	}
	
	// If no parentheses found, return the original string
	return raw
}
