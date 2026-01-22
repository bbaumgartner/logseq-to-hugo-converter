package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// ImageProcessor handles image processing and copying
type ImageProcessor struct {
	inputDir   string
	outputDir  string
	assetRegex *regexp.Regexp
}

func NewImageProcessor(inputDir, outputDir string) *ImageProcessor {
	return &ImageProcessor{
		inputDir:   inputDir,
		outputDir:  outputDir,
		assetRegex: regexp.MustCompile(`!\[(.*?)\]\((.*?assets\/)(.*?)\)`),
	}
}

func (p *ImageProcessor) ProcessContent(content string) string {
	matches := p.assetRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		src := filepath.Join(p.inputDir, match[2]+match[3])
		dst := filepath.Join(p.outputDir, match[3])
		p.copyFile(src, dst)
	}

	return p.assetRegex.ReplaceAllString(content, "![$1]($3)")
}

func (p *ImageProcessor) ProcessHeaderImage(headerPath string) {
	if headerPath == "" {
		return
	}

	src := filepath.Join(p.inputDir, headerPath)
	ext := filepath.Ext(filepath.Base(headerPath))
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
