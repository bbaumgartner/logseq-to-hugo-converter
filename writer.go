package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// HugoWriter writes blog posts in Hugo format
type HugoWriter struct {
	outputDir string
}

func NewHugoWriter(outputDir string) *HugoWriter {
	return &HugoWriter{outputDir: outputDir}
}

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
