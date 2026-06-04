package main

import (
	"strings"
	"testing"
)

func TestRestoreMarkdownImagePaths(t *testing.T) {
	source := `Intro text.

![sunset-ibiza.jpg](sunset-ibiza_1769102013331_0.jpg)

More text.`

	translated := `Texto.

![atardecer ibiza](sunset_ibiza_1769102013331_0.jpg)

Más.`

	got := restoreMarkdownImagePaths(source, translated)
	want := `![atardecer ibiza](sunset-ibiza_1769102013331_0.jpg)`
	if !strings.Contains(got, want) {
		t.Fatalf("restoreMarkdownImagePaths() =\n%q\nwant substring %q", got, want)
	}
}

func TestRestoreShortcodeSrc(t *testing.T) {
	source := `{{< video src="ibiza-idiots_1769100828988_0.mp4" >}}`
	translated := `{{< video src="ibiza_idiots_1769100828988_0.mp4" >}}`

	got := restoreShortcodeSrc(source, translated)
	want := `{{< video src="ibiza-idiots_1769100828988_0.mp4" >}}`
	if got != want {
		t.Fatalf("restoreShortcodeSrc() = %q, want %q", got, want)
	}
}

func TestRestoreAssetReferences_multipleImages(t *testing.T) {
	source := `![a](first_1.jpg)
![b](second_2.jpg)`
	translated := `![x](wrong_1.jpg)
![y](wrong_2.jpg)`

	got := restoreAssetReferences(source, translated)
	if !strings.Contains(got, "first_1.jpg") || !strings.Contains(got, "second_2.jpg") {
		t.Fatalf("restoreAssetReferences() = %q", got)
	}
}
