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

type BlogMeta struct {
	Date    string
	Title   string
	Author  string
	Header  string
	Summary string
	IsBlog  bool
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <input_file.md> <output_directory>")
		return
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]
	source, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Determine the directory of the input file to resolve relative asset paths
	inputDir := filepath.Dir(inputPath)

	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	meta, contentBlocks := extractBlogByFirstItem(doc, source)

	if !meta.IsBlog {
		fmt.Println("No list starting with 'type:: blog' found.")
		return
	}

	// 1. Prepare Folder
	folderName := fmt.Sprintf("%s_%s", meta.Date, strings.ReplaceAll(meta.Title, " ", "_"))
	fullOutputPath := filepath.Join(outputPath, folderName)
	os.MkdirAll(fullOutputPath, 0755)

	// 2. Process Content
	var finalContent strings.Builder
	for _, block := range contentBlocks {
		cleaned := strings.TrimSpace(block)
		if cleaned != "" {
			finalContent.WriteString(cleaned + "\n\n")
		}
	}

	// 3. Process Images & Header
	bodyStr := processImages(finalContent.String(), fullOutputPath, inputDir)
	handleHeaderImage(meta.Header, fullOutputPath, inputDir)

	// 4. Write index.md
	writeIndex(fullOutputPath, meta, strings.TrimSpace(bodyStr))
	fmt.Printf("Created: %s/index.md\n", fullOutputPath)
}

func extractBlogByFirstItem(doc ast.Node, source []byte) (BlogMeta, []string) {
	var meta BlogMeta
	var contentBlocks []string

	// Corrected Walk signature to match ast.Walker interface
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindList {
			firstItem := n.FirstChild()
			if firstItem != nil {
				itemText := string(firstItem.Text(source))
				if strings.Contains(itemText, "type:: blog") {
					parseBlogList(n, source, &meta, &contentBlocks)
					return ast.WalkStop, nil
				}
			}
		}
		return ast.WalkContinue, nil
	})

	if err != nil {
		fmt.Printf("Walk error: %v\n", err)
	}

	return meta, contentBlocks
}

func parseBlogList(listNode ast.Node, source []byte, meta *BlogMeta, blocks *[]string) {
	reMeta := regexp.MustCompile(`(\w+)::\s*(.*)`)
	count := 0

	for item := listNode.FirstChild(); item != nil; item = item.NextSibling() {
		if count == 0 {
			// Extract metadata from the trigger item
			meta.IsBlog = true
			lines := strings.Split(string(item.Text(source)), "\n")
			for _, line := range lines {
				if m := reMeta.FindStringSubmatch(line); m != nil {
					val := strings.TrimSpace(m[2])
					switch m[1] {
					case "date":
						meta.Date = val
					case "title":
						meta.Title = val
					case "author":
						meta.Author = val
					case "header":
						meta.Header = extractPath(val)
					}
				}
			}
		} else {
			rawText := getCleanNodeText(item, source)
			if count == 1 {
				meta.Summary = strings.ReplaceAll(rawText, "\n", " ")
			}
			*blocks = append(*blocks, rawText)
		}
		count++
	}
}

func getCleanNodeText(n ast.Node, source []byte) string {
	var buf strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {

		// Fix: Handle Headings specifically
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
	if m := re.FindStringSubmatch(raw); len(m) > 1 {
		return m[1]
	}
	return raw
}

func processImages(content string, folder string, inputDir string) string {
	// Match both ../assets/ and assets/ paths
	re := regexp.MustCompile(`!\[(.*?)\]\((.*?assets\/)(.*?)\)`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		relAssetPath := m[2] + m[3]
		src := filepath.Join(inputDir, relAssetPath)
		dst := filepath.Join(folder, m[3])
		copyFile(src, dst)
	}
	return re.ReplaceAllString(content, "![$1]($3)")
}

func handleHeaderImage(relPath, folder, inputDir string) {
	if relPath == "" {
		return
	}
	fileName := filepath.Base(relPath)
	src := filepath.Join(inputDir, relPath)
	ext := filepath.Ext(fileName)
	copyFile(src, filepath.Join(folder, "featured"+ext))
}

func writeIndex(folder string, meta BlogMeta, content string) {
	f, _ := os.Create(filepath.Join(folder, "index.md"))
	defer f.Close()

	summary := meta.Summary

	fmt.Fprintf(f, "+++\ndate = '%s'\nlastmod = '%s'\ndraft = false\ntitle = '%s'\nsummary = '%s'\n[params]\n  author = '%s'\n+++\n\n%s\n",
		meta.Date, meta.Date, meta.Title, summary, meta.Author, content)
}

func copyFile(src, dst string) {
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
