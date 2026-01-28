// Package main provides OpenAI integration for translation.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Translator handles translation using OpenAI GPT-4-turbo.
type Translator struct {
	client *openai.Client
}

// NewTranslator creates a new Translator with OpenAI client.
func NewTranslator() (*Translator, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &Translator{
		client: &client,
	}, nil
}

// TranslateText translates text to the target language using GPT-4-turbo.
func (t *Translator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	systemPrompt := fmt.Sprintf(`You are a professional translator. Translate the following text from %s to %s.

IMPORTANT RULES:
1. Preserve ALL markdown formatting exactly (links, images, headers, bold, italic, lists, tables, etc.)
2. Keep proper nouns in their original form unless they have a commonly used translation
3. Maintain the same tone and style as the original
4. Do NOT add any explanations, notes, or comments
5. Return ONLY the translated text, nothing else
6. Keep all HTML tags and shortcodes unchanged (e.g., {{< video src="..." >}})
7. Do not translate file paths or URLs`, sourceLang, targetLang)

	// Create chat completion with retry logic
	var translation string
	var err error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		completion, apiErr := t.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4Turbo,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(text),
			},
			Temperature: openai.Float(0.3), // Lower temperature for more deterministic translations
		})

		if apiErr != nil {
			err = apiErr
			if attempt < maxRetries-1 {
				// Wait before retrying
				time.Sleep(time.Second * time.Duration(attempt+1))
				continue
			}
			return "", fmt.Errorf("OpenAI API call failed after %d attempts: %w", maxRetries, err)
		}

		if len(completion.Choices) == 0 {
			return "", fmt.Errorf("no translation returned from API")
		}

		translation = completion.Choices[0].Message.Content
		break
	}

	return translation, nil
}

// TranslateFrontmatter translates the title and summary fields of the frontmatter.
func (t *Translator) TranslateFrontmatter(ctx context.Context, fm *Frontmatter, sourceLang, targetLang string) (*Frontmatter, error) {
	translated := *fm // Copy the frontmatter

	// Translate title
	if fm.Title != "" {
		translatedTitle, err := t.TranslateText(ctx, fm.Title, sourceLang, targetLang)
		if err != nil {
			return nil, fmt.Errorf("translating title: %w", err)
		}
		translated.Title = translatedTitle
	}

	// Translate summary
	if fm.Summary != "" {
		translatedSummary, err := t.TranslateText(ctx, fm.Summary, sourceLang, targetLang)
		if err != nil {
			return nil, fmt.Errorf("translating summary: %w", err)
		}
		translated.Summary = translatedSummary
	}

	return &translated, nil
}

// TranslateMarkdownFile translates an entire markdown file to the target language.
func (t *Translator) TranslateMarkdownFile(ctx context.Context, mf *MarkdownFile, targetLang Language) (*MarkdownFile, error) {
	fmt.Printf("  → Translating to %s...", targetLang.Name)

	// Translate frontmatter
	translatedFM, err := t.TranslateFrontmatter(ctx, &mf.Frontmatter, mf.SourceLang, targetLang.Code)
	if err != nil {
		return nil, fmt.Errorf("translating frontmatter: %w", err)
	}

	// Translate content
	translatedContent, err := t.TranslateText(ctx, mf.Content, mf.SourceLang, targetLang.Code)
	if err != nil {
		return nil, fmt.Errorf("translating content: %w", err)
	}

	fmt.Println(" ✓")

	return &MarkdownFile{
		Frontmatter: *translatedFM,
		Content:     translatedContent,
		SourceLang:  targetLang.Code,
	}, nil
}
