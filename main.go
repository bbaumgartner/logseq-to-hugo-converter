package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <input_file.md> <output_directory>")
		return
	}

	converter := NewBlogConverter(os.Args[2])
	outputPath, err := converter.Convert(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Created: %s/index.md\n", outputPath)
}

// BlogConverter orchestrates the conversion process
type BlogConverter struct {
	extractors     []BlogExtractor
	outputBasePath string
}

func NewBlogConverter(outputBasePath string) *BlogConverter {
	return &BlogConverter{
		extractors: []BlogExtractor{
			NewNestedListExtractor(),
			NewTopLevelMetadataExtractor(),
		},
		outputBasePath: outputBasePath,
	}
}

func (c *BlogConverter) Convert(inputPath string) (string, error) {
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("reading input file: %w", err)
	}

	doc := goldmark.New().Parser().Parse(text.NewReader(source))

	post, err := c.extractBlogPost(doc, source)
	if err != nil {
		return "", err
	}

	if post.Meta.Status != "online" {
		return "", fmt.Errorf("blog post status is '%s', only 'online' posts are converted", post.Meta.Status)
	}

	outputDir := c.createOutputDir(post.Meta)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	content := c.buildContent(post.Content)

	inputDir := filepath.Dir(inputPath)
	processor := NewImageProcessor(inputDir, outputDir)
	content = processor.ProcessContent(content)
	processor.ProcessHeaderImage(post.Meta.Header)

	writer := NewHugoWriter(outputDir)
	if err := writer.Write(post.Meta, content); err != nil {
		return "", err
	}

	return outputDir, nil
}

func (c *BlogConverter) extractBlogPost(doc interface{}, source []byte) (*BlogPost, error) {
	for _, extractor := range c.extractors {
		if post, found := extractor.Extract(doc, source); found {
			return post, nil
		}
	}
	return nil, fmt.Errorf("no blog post found with 'type:: blog' marker")
}

func (c *BlogConverter) createOutputDir(meta BlogMeta) string {
	folderName := fmt.Sprintf("%s_%s", meta.Date, strings.ReplaceAll(meta.Title, " ", "_"))
	return filepath.Join(c.outputBasePath, folderName)
}

func (c *BlogConverter) buildContent(blocks []string) string {
	var builder strings.Builder
	for _, block := range blocks {
		if cleaned := strings.TrimSpace(block); cleaned != "" {
			builder.WriteString(cleaned + "\n\n")
		}
	}
	return strings.TrimSpace(builder.String())
}

// convertLogseqToHugo provides backward compatibility with tests
func convertLogseqToHugo(inputPath, outputPath string) (string, error) {
	return NewBlogConverter(outputPath).Convert(inputPath)
}
