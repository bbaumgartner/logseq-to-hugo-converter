package main

import (
	"testing"
)

func TestProcessVideoEmbeds_YouTube(t *testing.T) {
	processor := NewImageProcessor(t.TempDir(), t.TempDir())

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "YouTube watch URL",
			input: `{{video https://www.youtube.com/watch?v=pwo3MA2FTRw}}`,
			want:  `{{< youtube pwo3MA2FTRw >}}`,
		},
		{
			name:  "YouTube short URL",
			input: `{{video https://youtu.be/dQw4w9WgXcQ}}`,
			want:  `{{< youtube dQw4w9WgXcQ >}}`,
		},
		{
			name:  "YouTube embed URL",
			input: `{{video https://www.youtube.com/embed/abc123_-X}}`,
			want:  `{{< youtube abc123_-X >}}`,
		},
		{
			name:  "YouTube with extra spaces",
			input: `{{video  https://www.youtube.com/watch?v=pwo3MA2FTRw  }}`,
			want:  `{{< youtube pwo3MA2FTRw >}}`,
		},
		{
			name:  "Non-YouTube URL stays unchanged",
			input: `{{video https://example.com/video.mp4}}`,
			want:  `{{video https://example.com/video.mp4}}`,
		},
		{
			name:  "No video embed stays unchanged",
			input: `Just some normal text with no embeds`,
			want:  `Just some normal text with no embeds`,
		},
		{
			name:  "YouTube embed surrounded by text",
			input: "Some text before\n\n{{video https://www.youtube.com/watch?v=abc123}}\n\nSome text after",
			want:  "Some text before\n\n{{< youtube abc123 >}}\n\nSome text after",
		},
		{
			name:  "Multiple YouTube embeds",
			input: "{{video https://www.youtube.com/watch?v=aaa}}\n\n{{video https://youtu.be/bbb}}",
			want:  "{{< youtube aaa >}}\n\n{{< youtube bbb >}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.processVideoEmbeds(tt.input)
			if got != tt.want {
				t.Errorf("processVideoEmbeds() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestProcessContent_YouTubeEmbed(t *testing.T) {
	processor := NewImageProcessor(t.TempDir(), t.TempDir())

	input := "Some text\n\n{{video https://www.youtube.com/watch?v=pwo3MA2FTRw}}\n\nMore text"
	want := "Some text\n\n{{< youtube pwo3MA2FTRw >}}\n\nMore text"

	got := processor.ProcessContent(input)
	if got != want {
		t.Errorf("ProcessContent() with YouTube embed =\n%q\nwant\n%q", got, want)
	}
}

func TestProcessContent_SubfolderAssets(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	processor := NewImageProcessor(inputDir, outputDir)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "image in named subfolder",
			input: `![renand](../assets/Renan/renand.jpg)`,
			want:  `![renand](renand.jpg)`,
		},
		{
			name:  "image in date subfolder",
			input: `![photo](../assets/2026_04_24/lukas-oel_1777135107896_0.jpg)`,
			want:  `![photo](lukas-oel_1777135107896_0.jpg)`,
		},
		{
			name:  "video in subfolder",
			input: `![clip](../assets/2026_04_24/segeln-380_1777135173295_0.mp4)`,
			want:  `{{< video src="segeln-380_1777135173295_0.mp4" >}}`,
		},
		{
			name:  "flat asset still works",
			input: `![photo](../assets/photo.jpg)`,
			want:  `![photo](photo.jpg)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.ProcessContent(tt.input)
			if got != tt.want {
				t.Errorf("ProcessContent() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestProcessContent_MixedMediaAndYouTube(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	processor := NewImageProcessor(inputDir, outputDir)

	input := "![photo](../assets/photo.jpg)\n\n{{video https://www.youtube.com/watch?v=abc123}}\n\n![video](../assets/clip_123.mp4)"
	got := processor.ProcessContent(input)

	if expected := `{{< youtube abc123 >}}`; !contains(got, expected) {
		t.Errorf("ProcessContent() missing YouTube shortcode %q in:\n%s", expected, got)
	}
	if expected := `{{< video src="clip_123.mp4" >}}`; !contains(got, expected) {
		t.Errorf("ProcessContent() missing video shortcode %q in:\n%s", expected, got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
