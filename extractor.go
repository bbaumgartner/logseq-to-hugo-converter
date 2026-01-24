// Package main contains the blog extraction logic.
// This file handles finding and extracting blog posts from Logseq markdown.
package main

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// extractBlogPosts finds all blog posts in a markdown document.
// It handles two formats:
// 1. List format: metadata in first list item
// 2. Top-level format: metadata as paragraphs, content in lists
func extractBlogPosts(doc ast.Node, source []byte) []*BlogPost {
	var posts []*BlogPost
	processedLists := make(map[ast.Node]bool)
	parser := NewMetadataParser()

	// First, check for top-level metadata format
	if topLevelPost := extractTopLevelPost(doc, source, parser); topLevelPost != nil {
		posts = append(posts, topLevelPost)
		return posts
	}

	// Walk through the AST looking for list-based blog posts
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != ast.KindList || processedLists[n] {
			return ast.WalkContinue, nil
		}

		// Check if first item contains "type:: blog"
		firstItem := n.FirstChild()
		if firstItem == nil || !strings.Contains(string(firstItem.Text(source)), "type:: blog") {
			return ast.WalkContinue, nil
		}

		// Found a blog list! Extract it
		post := extractListPost(n, firstItem, source, parser)
		if post != nil {
			posts = append(posts, post)
		}

		// Mark this list and all nested lists as processed
		ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering && child.Kind() == ast.KindList {
				processedLists[child] = true
			}
			return ast.WalkContinue, nil
		})

		return ast.WalkContinue, nil
	})

	return posts
}

// extractTopLevelPost extracts a blog post from top-level metadata format.
// In this format, metadata is in paragraphs at the start, followed by content lists.
func extractTopLevelPost(doc ast.Node, source []byte, parser *MetadataParser) *BlogPost {
	var metadataLines []string
	var contentBlocks []string
	foundBlogMarker := false

	// Walk and collect metadata and content
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Look for metadata in paragraphs
		if n.Kind() == ast.KindParagraph {
			text := string(n.Text(source))
			if strings.Contains(text, "::") {
				lines := strings.Split(text, "\n")
				for _, line := range lines {
					if strings.Contains(line, "::") {
						metadataLines = append(metadataLines, line)
						if strings.Contains(line, "type:: blog") {
							foundBlogMarker = true
						}
					}
				}
			}
		}

		// Collect content from top-level lists (skip nested)
		if foundBlogMarker && n.Kind() == ast.KindList {
			if n.Parent() != nil && n.Parent().Kind() == ast.KindListItem {
				return ast.WalkContinue, nil
			}
			for item := n.FirstChild(); item != nil; item = item.NextSibling() {
				contentBlocks = append(contentBlocks, extractText(item, source))
			}
		}

		return ast.WalkContinue, nil
	})

	if !foundBlogMarker {
		return nil
	}

	meta := parser.Parse(metadataLines)
	post := &BlogPost{
		Meta:    meta,
		Content: contentBlocks,
	}

	if len(contentBlocks) > 0 && post.Meta.Summary == "" {
		post.Meta.Summary = strings.ReplaceAll(contentBlocks[0], "\n", " ")
	}

	return post
}

// extractListPost extracts a single blog post from a list node.
// It handles both flat and nested list structures.
func extractListPost(listNode ast.Node, firstItem ast.Node, source []byte, parser *MetadataParser) *BlogPost {
	// Find the deepest nested list (handles arbitrary nesting)
	deepestList := findDeepestList(firstItem)
	if deepestList != firstItem {
		listNode = deepestList
	}

	// Extract metadata and content
	var metadataLines []string
	var contentBlocks []string

	count := 0
	for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
		if count == 0 {
			// First item contains metadata
			lines := strings.Split(string(item.Text(source)), "\n")
			metadataLines = append(metadataLines, lines...)
		} else {
			// Remaining items are content
			content := extractText(item, source)
			if content != "" {
				contentBlocks = append(contentBlocks, content)
			}
		}
		count++
	}

	// Parse metadata
	meta := parser.Parse(metadataLines)

	// Create blog post
	post := &BlogPost{
		Meta:    meta,
		Content: contentBlocks,
	}

	// Use first content block as summary if available
	if len(contentBlocks) > 0 && post.Meta.Summary == "" {
		post.Meta.Summary = strings.ReplaceAll(contentBlocks[0], "\n", " ")
	}

	return post
}

// findDeepestList recursively finds the deepest nested list within a node.
func findDeepestList(node ast.Node) ast.Node {
	deepest := node
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindList {
			return findDeepestList(child)
		}
	}
	return deepest
}

// extractText extracts text from an AST node while preserving markdown formatting.
func extractText(n ast.Node, source []byte) string {
	var builder strings.Builder
	
	// Walk through children to extract content
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindList {
			// Handle nested lists - convert to flat bullet points
			builder.WriteString("\n")
			for listItem := child.FirstChild(); listItem != nil; listItem = listItem.NextSibling() {
				builder.WriteString("* ")
				builder.WriteString(string(listItem.Text(source)))
				builder.WriteString("\n")
			}
		} else if child.Kind() == ast.KindHeading {
			// For headings, we need to manually add the ### markers
			// because Lines() only returns the text content
			heading := child.(*ast.Heading)
			for i := 0; i < heading.Level; i++ {
				builder.WriteString("#")
			}
			builder.WriteString(" ")
			builder.WriteString(string(child.Text(source)))
			builder.WriteString("\n")
		} else {
			// For other children (paragraphs, images, etc.), get their raw lines
			// This preserves markdown formatting like **, [], etc.
			lines := child.Lines()
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				builder.Write(line.Value(source))
			}
		}
	}

	return strings.TrimSpace(builder.String())
}
