// This file implements the extraction strategies for different Logseq formats.
// It contains two extractors: one for nested lists (journals) and one for
// top-level metadata (pages). Both implement the BlogExtractor interface.
package main

import (
	"strings" // String manipulation functions

	"github.com/yuin/goldmark/ast" // Abstract Syntax Tree types for markdown
)

// ═══════════════════════════════════════════════════════════════════════════
// NESTED LIST EXTRACTOR (for Journal Format)
// ═══════════════════════════════════════════════════════════════════════════

// NestedListExtractor extracts blog posts from the nested list format.
// This format is typically used in Logseq journals where metadata is nested
// inside a list item: - [[Blog]] → - type:: blog → - content
type NestedListExtractor struct {
	parser *MetadataParser // Pointer to a metadata parser instance
}

// NewNestedListExtractor creates a new instance of NestedListExtractor.
// This is a constructor function that initializes the extractor with a parser.
func NewNestedListExtractor() *NestedListExtractor {
	// Return a pointer to a new extractor struct
	// We initialize it with a new metadata parser
	return &NestedListExtractor{
		parser: NewMetadataParser(), // Create a new parser for this extractor
	}
}

// Extract implements the BlogExtractor interface for nested list format.
// It walks through the markdown AST looking for lists containing "type:: blog".
// Parameters:
//
//	doc: The parsed markdown document (we'll cast it to ast.Node)
//	source: The raw markdown content as bytes
//
// Returns:
//
//	[]*BlogPost: A slice of pointers to all extracted blog posts (empty if none found)
func (e *NestedListExtractor) Extract(doc interface{}, source []byte) []*BlogPost {
	// Slice to collect all blog posts found in the document
	var posts []*BlogPost
	
	// Track which lists we've already processed to avoid duplicates
	// When we find a blog list, we might encounter nested lists within it
	// We want to skip those to avoid extracting the same blog multiple times
	processedLists := make(map[ast.Node]bool)

	// Walk through the Abstract Syntax Tree (AST) of the markdown document
	// ast.Walk visits every node in the tree
	// The function we pass gets called for each node with two parameters:
	//   n: the current node
	//   entering: true when entering the node, false when leaving
	ast.Walk(doc.(ast.Node), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// We only process nodes when entering them, not when leaving
		// Also, we're only interested in List nodes
		if !entering || n.Kind() != ast.KindList {
			return ast.WalkContinue, nil // Continue to next node
		}

		// Skip if we've already processed this list
		if processedLists[n] {
			return ast.WalkContinue, nil
		}

		// Get the first item in the list
		firstItem := n.FirstChild()

		// If there's no first item, or it doesn't contain "type:: blog",
		// continue searching
		if firstItem == nil || !strings.Contains(string(firstItem.Text(source)), "type:: blog") {
			return ast.WalkContinue, nil
		}

		// We found a blog list! Extract it
		post := e.extractFromList(n, source)
		posts = append(posts, post)
		
		// Mark this list and all its nested lists as processed
		// This prevents us from extracting the same blog multiple times
		ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering && child.Kind() == ast.KindList {
				processedLists[child] = true
			}
			return ast.WalkContinue, nil
		})

		// Continue walking to find more blog posts (don't stop)
		return ast.WalkContinue, nil
	})

	// Return all extracted posts (may be empty if none found)
	return posts
}

// findDeepestNestedList recursively finds the deepest nested list within a node.
// This handles arbitrary nesting levels like: [[Category]] -> [[Subcategory]] -> [[Blog]] -> content
// Returns the deepest list found, or the original node if no nested lists exist.
func findDeepestNestedList(node ast.Node) ast.Node {
	deepestList := node

	// Walk through all children of the current node
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		// If we find a list, recursively check if it has even deeper lists
		if child.Kind() == ast.KindList {
			candidateList := findDeepestNestedList(child)
			// Update our deepest list to this new candidate
			deepestList = candidateList
			// Since we found a nested list, we should use that branch
			break
		}
	}

	return deepestList
}

// extractFromList extracts a blog post from a list node.
// The list structure can be either:
//   - [metadata item, content item 1, content item 2, ...] (flat structure)
//   - [[Blog] item with nested list containing [metadata, content...]] (nested structure)
//   - Multiple levels of nesting: [[Cat]] -> [[Subcat]] -> [[Blog]] -> [metadata, content...]
// This is a helper method that does the actual extraction work.
func (e *NestedListExtractor) extractFromList(listNode ast.Node, source []byte) *BlogPost {
	// Initialize a new BlogPost with an empty Content slice
	// []string{} creates an empty slice of strings
	post := &BlogPost{Content: []string{}}

	// Slice to collect metadata lines
	metadataLines := []string{}

	// Counter to track which item we're processing
	count := 0

	// Find the deepest nested list within the first item
	// This handles arbitrary nesting levels (e.g., [[Category]] -> [[Blog]] -> content)
	firstItem := listNode.FirstChild()
	if firstItem != nil {
		// Recursively find the deepest nested list
		deepestList := findDeepestNestedList(firstItem)
		// If we found a nested list, use it instead of the original
		if deepestList != firstItem {
			listNode = deepestList
		}
	}

	// Iterate through all items in the list (either original or deepest nested)
	// FirstChild() gets the first item, NextSibling() moves to the next
	// The loop continues while item is not nil
	for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
		if count == 0 {
			// First item (index 0) contains the metadata
			// Get the text of the item and split it into lines
			lines := strings.Split(string(item.Text(source)), "\n")
			// Add all lines to our metadata collection
			metadataLines = append(metadataLines, lines...)
		} else {
			// All other items (index 1+) are content blocks
			// Extract the text from this item and add to content
			post.Content = append(post.Content, extractNodeText(item, source))
		}
		count++ // Increment the counter for next iteration
	}

	// Parse the metadata lines into a BlogMeta struct
	post.Meta = e.parser.Parse(metadataLines)

	// If there's content, use the first block as the summary
	if len(post.Content) > 0 {
		// Replace newlines with spaces for a clean summary
		post.Meta.Summary = strings.ReplaceAll(post.Content[0], "\n", " ")
	}

	// Return the completed blog post
	return post
}

// ═══════════════════════════════════════════════════════════════════════════
// TOP-LEVEL METADATA EXTRACTOR (for Pages Format)
// ═══════════════════════════════════════════════════════════════════════════

// TopLevelMetadataExtractor extracts blog posts from the top-level metadata format.
// This format has metadata at the top of the file (not in a list), followed by
// content in list items.
type TopLevelMetadataExtractor struct {
	parser *MetadataParser // Pointer to a metadata parser instance
}

// NewTopLevelMetadataExtractor creates a new instance of TopLevelMetadataExtractor.
func NewTopLevelMetadataExtractor() *TopLevelMetadataExtractor {
	return &TopLevelMetadataExtractor{
		parser: NewMetadataParser(),
	}
}

// Extract implements the BlogExtractor interface for top-level metadata format.
// It looks for metadata in paragraphs and content in lists.
// This format typically has only one blog post per file.
func (e *TopLevelMetadataExtractor) Extract(doc interface{}, source []byte) []*BlogPost {
	// Slices to collect metadata and content
	metadataLines := []string{} // Will hold "key:: value" lines
	contentBlocks := []string{} // Will hold content paragraphs
	foundBlogMarker := false    // Flag: have we seen "type:: blog"?

	// Walk through the markdown AST
	ast.Walk(doc.(ast.Node), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// Only process when entering nodes (not leaving)
		if !entering {
			return ast.WalkContinue, nil
		}

		// Look for metadata in Paragraph nodes (top-level text)
		if n.Kind() == ast.KindParagraph {
			// Get the text content of this paragraph
			text := string(n.Text(source))

			// If it contains "::" it might be metadata
			if strings.Contains(text, "::") {
				// Split the paragraph into individual lines
				lines := strings.Split(text, "\n")

				// Check each line for metadata
				for _, line := range lines {
					if strings.Contains(line, "::") {
						// This line is metadata, save it
						metadataLines = append(metadataLines, line)

						// If it's the blog type marker, set our flag
						if strings.Contains(line, "type:: blog") {
							foundBlogMarker = true
						}
					}
				}
			}
		}

		// After finding the blog marker, collect content from lists
		// Only process top-level lists, not nested lists
		if foundBlogMarker && n.Kind() == ast.KindList {
			// Check if this list is nested (parent is a ListItem)
			// If so, skip it because it will be processed by its parent
			if n.Parent() != nil && n.Parent().Kind() == ast.KindListItem {
				return ast.WalkContinue, nil // Skip nested lists
			}

			// Iterate through all items in this top-level list
			for item := n.FirstChild(); item != nil; item = item.NextSibling() {
				// Extract the text from each list item
				// This will include nested lists formatted correctly
				contentBlocks = append(contentBlocks, extractNodeText(item, source))
			}
		}

		// Continue walking the tree
		return ast.WalkContinue, nil
	})

	// If we never found "type:: blog", this isn't a blog post
	if !foundBlogMarker {
		return []*BlogPost{} // Return empty slice (not found)
	}

	// Create the blog post from our collected data
	post := &BlogPost{
		Meta:    e.parser.Parse(metadataLines), // Parse metadata into struct
		Content: contentBlocks,                 // Set the content blocks
	}

	// If there's content, use first block as summary
	if len(contentBlocks) > 0 {
		post.Meta.Summary = strings.ReplaceAll(contentBlocks[0], "\n", " ")
	}

	// Return a slice containing the single blog post
	return []*BlogPost{post}
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// extractNodeText extracts clean text from a markdown AST node.
// This handles special cases like headings and nested lists.
// Parameters:
//
//	n: The AST node to extract text from
//	source: The original markdown content as bytes
//
// Returns:
//
//	string: The extracted and cleaned text
func extractNodeText(n ast.Node, source []byte) string {
	// strings.Builder is an efficient way to build strings
	// It's better than concatenating strings with +
	var buf strings.Builder

	// Iterate through all child nodes
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		// Special handling for heading nodes (# Heading)
		// Type assertion: child.(*ast.Heading) tries to convert child to *ast.Heading
		// ok will be true if successful, false otherwise
		if heading, ok := child.(*ast.Heading); ok {
			// Add the appropriate number of # characters
			// strings.Repeat repeats a string n times
			buf.WriteString(strings.Repeat("#", heading.Level) + " ")
		}

		// Get the lines that make up this node
		// Lines() returns a Segments collection
		lines := child.Lines()

		// Iterate through each line segment
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)           // Get the i-th segment
			buf.Write(line.Value(source)) // Write the line's bytes to the buffer
		}

		// Special handling for nested lists
		// Convert nested list items to use asterisks and proper formatting
		if child.Kind() == ast.KindList {
			buf.WriteString("\n") // Add a newline before the list

			// Iterate through each list item in the nested list
			for listItem := child.FirstChild(); listItem != nil; listItem = listItem.NextSibling() {
				// Write the list marker (asterisk)
				buf.WriteString("* ")

				// Extract text from this list item's children
				// We need to get the actual text content from the paragraph or text nodes
				for itemChild := listItem.FirstChild(); itemChild != nil; itemChild = itemChild.NextSibling() {
					itemLines := itemChild.Lines()
					for i := 0; i < itemLines.Len(); i++ {
						line := itemLines.At(i)
						buf.Write(line.Value(source))
					}
				}

				// Add a newline after each list item
				buf.WriteString("\n")
			}
		}
	}

	// Convert the buffer to a string and return it
	return buf.String()
}
