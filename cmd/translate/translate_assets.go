package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	markdownImageRE = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	shortcodeSrcRE    = regexp.MustCompile(`\{\{<[^}]*\ssrc="([^"]+)"[^}]*>\}\}`)
)

// restoreAssetReferences copies image link targets and Hugo shortcode src values from
// the source markdown into the translation. Alt text may still be translated.
func restoreAssetReferences(source, translated string) string {
	translated = restoreMarkdownImagePaths(source, translated)
	translated = restoreShortcodeSrc(source, translated)
	return translated
}

func extractMarkdownImagePaths(content string) []string {
	matches := markdownImageRE.FindAllStringSubmatch(content, -1)
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		paths = append(paths, m[2])
	}
	return paths
}

func restoreMarkdownImagePaths(source, translated string) string {
	sourcePaths := extractMarkdownImagePaths(source)
	if len(sourcePaths) == 0 {
		return translated
	}

	idx := 0
	return markdownImageRE.ReplaceAllStringFunc(translated, func(match string) string {
		sub := markdownImageRE.FindStringSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		if idx >= len(sourcePaths) {
			return match
		}
		restored := fmt.Sprintf("![%s](%s)", sub[1], sourcePaths[idx])
		idx++
		return restored
	})
}

func extractShortcodeSrcPaths(content string) []string {
	matches := shortcodeSrcRE.FindAllStringSubmatch(content, -1)
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		paths = append(paths, m[1])
	}
	return paths
}

func restoreShortcodeSrc(source, translated string) string {
	sourcePaths := extractShortcodeSrcPaths(source)
	if len(sourcePaths) == 0 {
		return translated
	}

	idx := 0
	return shortcodeSrcRE.ReplaceAllStringFunc(translated, func(match string) string {
		if idx >= len(sourcePaths) {
			return match
		}
		sub := shortcodeSrcRE.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		path := sourcePaths[idx]
		idx++
		return strings.Replace(match, `src="`+sub[1]+`"`, `src="`+path+`"`, 1)
	})
}
