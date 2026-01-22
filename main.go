package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// BlogMeta represents the metadata of a blog post
type BlogMeta struct {
	Date    string
	Title   string
	Author  string
	Header  string
	Summary string
	Status  string
}

// BlogPost represents a complete blog post with metadata and content
type BlogPost struct {
	Meta    BlogMeta
	Content []string
}

// BlogExtractor is the strategy interface for extracting blog posts from different formats
type BlogExtractor interface {
	Extract(doc ast.Node, source []byte) (*BlogPost, bool)
}

// MetadataParser handles parsing of metadata lines
type MetadataParser struct {
	regex *regexp.Regexp
}

// NewMetadataParser creates a new metadata parser
func NewMetadataParser() *MetadataParser {
	return &MetadataParser{
		regex: regexp.MustCompile(`(\w+)::\s*(.*)`),
	}
}

// Parse extracts metadata from lines
func (p *MetadataParser) Parse(lines []string) BlogMeta {
	meta := BlogMeta{}
	for _, line := range lines {
		if match := p.regex.FindStringSubmatch(line); match != nil {
			key := match[1]
			value := strings.TrimSpace(match[2])
			p.setMetadataField(&meta, key, value)
		}
	}
	return meta
}

func (p *MetadataParser) setMetadataField(meta *BlogMeta, key, value string) {
	switch key {
	case "date":
		meta.Date = value
	case "title":
		meta.Title = value
	case "author":
		meta.Author = value
	case "header":
		meta.Header = extractPath(value)
	case "status":
		meta.Status = value
	}
}

// NestedListExtractor extracts blog posts from nested list format (journals)
type NestedListExtractor struct {
	parser *MetadataParser
}

// NewNestedListExtractor creates a new nested list extractor
func NewNestedListExtractor() *NestedListExtractor {
	return &NestedListExtractor{
		parser: NewMetadataParser(),
	}
}

// Extract implements BlogExtractor for nested list format
func (e *NestedListExtractor) Extract(doc ast.Node, source []byte) (*BlogPost, bool) {
	var post *BlogPost
	found := false

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != ast.KindList {
			return ast.WalkContinue, nil
		}

		firstItem := n.FirstChild()
		if firstItem == nil {
			return ast.WalkContinue, nil
		}

		itemText := string(firstItem.Text(source))
		if !strings.Contains(itemText, "type:: blog") {
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
			// First item contains metadata
			lines := strings.Split(string(item.Text(source)), "\n")
			metadataLines = append(metadataLines, lines...)
		} else {
			// Subsequent items are content
			rawText := extractNodeText(item, source)
			post.Content = append(post.Content, rawText)
		}
		count++
	}

	// Parse metadata and set summary from first content block
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

// NewTopLevelMetadataExtractor creates a new top-level metadata extractor
func NewTopLevelMetadataExtractor() *TopLevelMetadataExtractor {
	return &TopLevelMetadataExtractor{
		parser: NewMetadataParser(),
	}
}

// Extract implements BlogExtractor for top-level metadata format
func (e *TopLevelMetadataExtractor) Extract(doc ast.Node, source []byte) (*BlogPost, bool) {
	metadataLines := []string{}
	contentBlocks := []string{}
	foundBlogMarker := false

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Extract metadata from paragraphs
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

		// Extract content from lists
		if foundBlogMarker && n.Kind() == ast.KindList {
			for item := n.FirstChild(); item != nil; item = item.NextSibling() {
				rawText := extractNodeText(item, source)
				contentBlocks = append(contentBlocks, rawText)
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

// ImageProcessor handles image processing and copying
type ImageProcessor struct {
	inputDir   string
	outputDir  string
	assetRegex *regexp.Regexp
}

// NewImageProcessor creates a new image processor
func NewImageProcessor(inputDir, outputDir string) *ImageProcessor {
	return &ImageProcessor{
		inputDir:   inputDir,
		outputDir:  outputDir,
		assetRegex: regexp.MustCompile(`!\[(.*?)\]\((.*?assets\/)(.*?)\)`),
	}
}

// ProcessContent processes images in content and returns updated content
func (p *ImageProcessor) ProcessContent(content string) string {
	matches := p.assetRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		relAssetPath := match[2] + match[3]
		src := filepath.Join(p.inputDir, relAssetPath)
		dst := filepath.Join(p.outputDir, match[3])
		p.copyFile(src, dst)
	}

	return p.assetRegex.ReplaceAllString(content, "![$1]($3)")
}

// ProcessHeaderImage copies the header image as featured image
func (p *ImageProcessor) ProcessHeaderImage(headerPath string) {
	if headerPath == "" {
		return
	}

	fileName := filepath.Base(headerPath)
	src := filepath.Join(p.inputDir, headerPath)
	ext := filepath.Ext(fileName)
	dst := filepath.Join(p.outputDir, "featured"+ext)
	p.copyFile(src, dst)
}

func (p *ImageProcessor) copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		fmt.Printf("Warning: Missing image %s\n", src)
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer out.Close()

	io.Copy(out, in)
}

// HugoWriter writes blog posts in Hugo format
type HugoWriter struct {
	outputDir string
}

// NewHugoWriter creates a new Hugo writer
func NewHugoWriter(outputDir string) *HugoWriter {
	return &HugoWriter{outputDir: outputDir}
}

// Write writes the blog post to disk
func (w *HugoWriter) Write(meta BlogMeta, content string) error {
	indexPath := filepath.Join(w.outputDir, "index.md")
	f, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("creating index.md: %w", err)
	}
	defer f.Close()

	frontMatter := fmt.Sprintf(
		"+++\ndate = '%s'\nlastmod = '%s'\ndraft = false\ntitle = '%s'\nsummary = '%s'\n[params]\n  author = '%s'\n+++\n\n",
		meta.Date, meta.Date, meta.Title, meta.Summary, meta.Author,
	)

	if _, err := f.WriteString(frontMatter + content + "\n"); err != nil {
		return fmt.Errorf("writing content: %w", err)
	}

	return nil
}

// BlogConverter orchestrates the conversion process
type BlogConverter struct {
	extractors     []BlogExtractor
	outputBasePath string
}

// NewBlogConverter creates a new blog converter
func NewBlogConverter(outputBasePath string) *BlogConverter {
	return &BlogConverter{
		extractors: []BlogExtractor{
			NewNestedListExtractor(),
			NewTopLevelMetadataExtractor(),
		},
		outputBasePath: outputBasePath,
	}
}

// Convert converts a Logseq markdown file to Hugo format
func (c *BlogConverter) Convert(inputPath string) (string, error) {
	// Read input file
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("reading input file: %w", err)
	}

	inputDir := filepath.Dir(inputPath)

	// Parse markdown
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(source))

	// Try each extractor strategy
	var post *BlogPost
	for _, extractor := range c.extractors {
		if p, found := extractor.Extract(doc, source); found {
			post = p
			break
		}
	}

	if post == nil {
		return "", fmt.Errorf("no blog post found with 'type:: blog' marker")
	}

	// Validate status
	if post.Meta.Status != "online" {
		return "", fmt.Errorf("blog post status is '%s', only 'online' posts are converted", post.Meta.Status)
	}

	// Prepare output directory
	folderName := fmt.Sprintf("%s_%s", post.Meta.Date, strings.ReplaceAll(post.Meta.Title, " ", "_"))
	outputDir := filepath.Join(c.outputBasePath, folderName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	// Process content
	contentStr := c.buildContent(post.Content)

	// Process images
	imageProcessor := NewImageProcessor(inputDir, outputDir)
	contentStr = imageProcessor.ProcessContent(contentStr)
	imageProcessor.ProcessHeaderImage(post.Meta.Header)

	// Write output
	writer := NewHugoWriter(outputDir)
	if err := writer.Write(post.Meta, contentStr); err != nil {
		return "", err
	}

	return outputDir, nil
}

func (c *BlogConverter) buildContent(blocks []string) string {
	var builder strings.Builder
	for _, block := range blocks {
		if cleaned := strings.TrimSpace(block); cleaned != "" {
			builder.WriteString(cleaned)
			builder.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(builder.String())
}

// Helper functions

func extractNodeText(n ast.Node, source []byte) string {
	var buf strings.Builder

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		// Handle headings specifically
		if heading, ok := child.(*ast.Heading); ok {
			hashes := strings.Repeat("#", heading.Level)
			buf.WriteString(hashes + " ")
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

func extractPath(raw string) string {
	re := regexp.MustCompile(`\((.*?)\)`)
	if match := re.FindStringSubmatch(raw); len(match) > 1 {
		return match[1]
	}
	return raw
}

// Main entry point

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <input_file.md> <output_directory>")
		return
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	fullOutputPath, err := convertLogseqToHugo(inputPath, outputPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Created: %s/index.md\n", fullOutputPath)
}

// convertLogseqToHugo is the main conversion function (kept for backward compatibility with tests)
func convertLogseqToHugo(inputPath, outputPath string) (string, error) {
	converter := NewBlogConverter(outputPath)
	return converter.Convert(inputPath)
}
