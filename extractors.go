package main

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// NestedListExtractor extracts blog posts from nested list format (journals)
type NestedListExtractor struct {
	parser *MetadataParser
}

func NewNestedListExtractor() *NestedListExtractor {
	return &NestedListExtractor{parser: NewMetadataParser()}
}

func (e *NestedListExtractor) Extract(doc interface{}, source []byte) (*BlogPost, bool) {
	var post *BlogPost
	found := false

	ast.Walk(doc.(ast.Node), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != ast.KindList {
			return ast.WalkContinue, nil
		}

		firstItem := n.FirstChild()
		if firstItem == nil || !strings.Contains(string(firstItem.Text(source)), "type:: blog") {
			return ast.WalkContinue, nil
		}

		post = e.extractFromList(n, source)
		found = true
		return ast.WalkStop, nil
	})

	return post, found
}

func (e *NestedListExtractor) extractFromList(listNode ast.Node, source []byte) *BlogPost {
	post := &BlogPost{Content: []string{}}
	metadataLines := []string{}
	count := 0

	for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
		if count == 0 {
			lines := strings.Split(string(item.Text(source)), "\n")
			metadataLines = append(metadataLines, lines...)
		} else {
			post.Content = append(post.Content, extractNodeText(item, source))
		}
		count++
	}

	post.Meta = e.parser.Parse(metadataLines)
	if len(post.Content) > 0 {
		post.Meta.Summary = strings.ReplaceAll(post.Content[0], "\n", " ")
	}

	return post
}

// TopLevelMetadataExtractor extracts blog posts from top-level metadata format (pages)
type TopLevelMetadataExtractor struct {
	parser *MetadataParser
}

func NewTopLevelMetadataExtractor() *TopLevelMetadataExtractor {
	return &TopLevelMetadataExtractor{parser: NewMetadataParser()}
}

func (e *TopLevelMetadataExtractor) Extract(doc interface{}, source []byte) (*BlogPost, bool) {
	metadataLines := []string{}
	contentBlocks := []string{}
	foundBlogMarker := false

	ast.Walk(doc.(ast.Node), func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if n.Kind() == ast.KindParagraph {
			text := string(n.Text(source))
			if strings.Contains(text, "::") {
				for _, line := range strings.Split(text, "\n") {
					if strings.Contains(line, "::") {
						metadataLines = append(metadataLines, line)
						if strings.Contains(line, "type:: blog") {
							foundBlogMarker = true
						}
					}
				}
			}
		}

		if foundBlogMarker && n.Kind() == ast.KindList {
			for item := n.FirstChild(); item != nil; item = item.NextSibling() {
				contentBlocks = append(contentBlocks, extractNodeText(item, source))
			}
		}

		return ast.WalkContinue, nil
	})

	if !foundBlogMarker {
		return nil, false
	}

	post := &BlogPost{
		Meta:    e.parser.Parse(metadataLines),
		Content: contentBlocks,
	}

	if len(contentBlocks) > 0 {
		post.Meta.Summary = strings.ReplaceAll(contentBlocks[0], "\n", " ")
	}

	return post, true
}

func extractNodeText(n ast.Node, source []byte) string {
	var buf strings.Builder

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if heading, ok := child.(*ast.Heading); ok {
			buf.WriteString(strings.Repeat("#", heading.Level) + " ")
		}

		lines := child.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}

		if child.Kind() == ast.KindList {
			buf.WriteString("\n")
			buf.Write(child.Text(source))
		}
	}

	return buf.String()
}
