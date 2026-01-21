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
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <input_file.md>")
		return
	}

	inputPath := os.Args[1]
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

	// 1. Prepare Folder (Output will be created in the current working directory)
	folderName := fmt.Sprintf("%s_%s", meta.Date, strings.ReplaceAll(meta.Title, " ", "_"))
	os.MkdirAll(folderName, 0755)

	// 2. Process Content
	var finalContent strings.Builder
	for _, block := range contentBlocks {
		finalContent.WriteString(block + "\n")
	}

	// 3. Process Images (Passing inputDir to resolve ../assets)
	processedBody := processImages(finalContent.String(), folderName, inputDir)
	handleHeaderImage(meta.Header, folderName, inputDir)

	// 4. Write index.md
	writeIndex(folderName, meta, processedBody)
	fmt.Printf("Successfully converted blog to: %s/\n", folderName)
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
			// Extract text and preserve nesting for all subsequent items
			blockText := getRawNodeText(item, source)
			if count == 1 {
				meta.Summary = strings.TrimSpace(blockText)
			}
			*blocks = append(*blocks, blockText)
		}
		count++
	}
}

func getRawNodeText(n ast.Node, source []byte) string {
	var buf strings.Builder

	// Part 1: The current bullet's content (Paragraph or Heading)
	firstPart := n.FirstChild()
	if firstPart != nil {
		lines := firstPart.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(source))
		}
	}

	// Part 2: All nested children (sub-lists)
	for child := firstPart.NextSibling(); child != nil; child = child.NextSibling() {
		// Use the goldmark-native text printer for nested structures to keep formatting
		buf.WriteString("\n")
		buf.Write(child.Text(source))
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
		relAssetPath := m[2] + m[3] // e.g., "../assets/image.png"
		src := filepath.Join(inputDir, relAssetPath)
		dst := filepath.Join(folder, m[3])
		copyFile(src, dst)
	}
	// Clean the markdown to point to the local folder (no subfolders)
	return re.ReplaceAllString(content, `![$1]($3)`)
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

	summary := strings.Split(meta.Summary, "\n")[0]

	fmt.Fprintf(f, "+++\ndate = '%s'\nlastmod = '%s'\ndraft = false\ntitle = '%s'\nsummary = '%s'\n[params]\n  author = '%s'\n+++\n\n%s",
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
