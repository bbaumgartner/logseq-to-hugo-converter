package main

import (
	"regexp"
	"strings"
)

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
			p.setField(&meta, key, value)
		}
	}
	return meta
}

func (p *MetadataParser) setField(meta *BlogMeta, key, value string) {
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

func extractPath(raw string) string {
	re := regexp.MustCompile(`\((.*?)\)`)
	if match := re.FindStringSubmatch(raw); len(match) > 1 {
		return match[1]
	}
	return raw
}
