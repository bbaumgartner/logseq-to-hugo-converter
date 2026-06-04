// Package main provides OpenAI integration for translation.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// translationModel is the OpenAI chat model used for all translations.
const translationModel = openai.ChatModelGPT4_1

// Translator handles translation using OpenAI GPT-4.1.
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

// languageDisplayName returns a human-readable language name for prompts.
func languageDisplayName(code string) string {
	names := map[string]string{
		"en":   "English",
		"de":   "German",
		"es":   "Spanish",
		"fr":   "French",
		"it":   "Italian",
		"arrr": "Pirate Speak",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

// buildSystemPrompt returns the appropriate system prompt for the given language pair.
func buildSystemPrompt(sourceLang, targetLang string) string {
	if targetLang == "arrr" {
		return `Ye be a foul-mouthed, barnacle-covered sea dog of a rewriter. Rewrite the followin' English blog text into the most extreme, over-the-top pirate speak imaginable — like a mad captain three sheets to the wind on the high seas.

STYLE:
- Keep the same meaning, structure, and humor as the original; only the voice changes.
- Go EXTREME with pirate speak — every sentence must drip with nautical swagger:
  - Replace "you/your" with "ye/yer" throughout
  - Replace "my/I/me" with "me" throughout
  - Replace "is/are/was" with "be" throughout
  - Replace "the" with "th'" frequently
  - Pepper every few sentences with outbursts like "ARRR!", "BLIMEY!", "SHIVER ME TIMBERS!", "AVAST!", "BY DAVY JONES' LOCKER!"
  - Replace mundane words with nautical equivalents where it fits (house→vessel, car→horseless carriage, toilet→the poop deck, travel→voyage, money→doubloons, etc.)
  - Add dramatic pirate interjections mid-paragraph occasionally

RULES:
1. Preserve ALL markdown formatting exactly (links, images, headers, bold, italic, lists, tables, etc.)
2. In image syntax ![alt](filename.jpg), you may change alt text but the filename inside parentheses must stay byte-for-byte identical to the source
3. Keep proper nouns, place names, and certification codes unchanged
4. Do not change file paths, URLs, or any src="..." value in Hugo shortcodes (e.g. {{< video src="..." >}})
5. Return ONLY the rewritten text — no explanations or notes`
	}

	sourceName := languageDisplayName(sourceLang)
	targetName := languageDisplayName(targetLang)

	return fmt.Sprintf(`You are an expert translator for personal blog posts. Translate from %s to %s.

QUALITY:
- Write fluent, natural %s that reads like a native blogger wrote it — avoid stiff, literal, or machine-like phrasing.
- Preserve the author's voice: informal, witty, serious, or sarcastic — match the source register.
- Adapt idioms and cultural references for %s readers when needed; do not translate word-for-word if a natural equivalent exists.
- Use consistent terminology and wording throughout the entire text (including repeated terms and names).

CONTENT:
1. Preserve ALL markdown formatting exactly (links, images, headers, bold, italic, lists, tables, etc.)
2. In image syntax ![alt](filename.jpg), you may translate the alt text but the filename inside parentheses must stay byte-for-byte identical to the source — never rename, translate, or reformat it (keep hyphens, underscores, and extensions exactly)
3. Keep proper nouns, brand names, place names, and certification or license codes (e.g. SKS, SBF See, RYA, ASA) unless a standard %s exonym exists.
4. Do not translate file paths, URLs, HTML tags, or Hugo shortcode src attributes (e.g. {{< video src="clip.mp4" >}} must keep the same src value)

OUTPUT:
- Return ONLY the translated text.
- No explanations, notes, glossaries, or translator comments.`, sourceName, targetName, targetName, targetName, targetName)
}

// temperatureFor returns the model temperature for a given target language.
// Pirate speak uses a higher temperature for more creative, unpredictable output.
// Real language translations use a low temperature for accuracy and consistency.
func temperatureFor(targetLang string) float64 {
	if targetLang == "arrr" {
		return 0.9
	}
	return 0.3
}

// TranslateText translates text to the target language using GPT-4.1.
func (t *Translator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	systemPrompt := buildSystemPrompt(sourceLang, targetLang)

	// Create chat completion with retry logic
	var translation string
	var err error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		completion, apiErr := t.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: translationModel,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(text),
			},
			Temperature: openai.Float(temperatureFor(targetLang)),
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

// TranslateFrontmatter translates only the title field of the frontmatter.
// The summary will be extracted from the first paragraph of translated content.
func (t *Translator) TranslateFrontmatter(ctx context.Context, fm *Frontmatter, sourceLang, targetLang string) (*Frontmatter, error) {
	translated := *fm // Copy the frontmatter

	// Translate title — pirate speak keeps the original English title to avoid comically long results
	if fm.Title != "" && targetLang != "arrr" {
		translatedTitle, err := t.TranslateText(ctx, fm.Title, sourceLang, targetLang)
		if err != nil {
			return nil, fmt.Errorf("translating title: %w", err)
		}
		translated.Title = translatedTitle
	}

	// Note: Summary will be set from the first paragraph of translated content
	// This is done in TranslateMarkdownFile to save tokens and speed up translation

	return &translated, nil
}

// extractFirstParagraph extracts the first paragraph from markdown content.
// A paragraph is defined as text before the first blank line or heading.
func extractFirstParagraph(content string) string {
	lines := strings.Split(content, "\n")
	var firstParagraph []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at the start
		if len(firstParagraph) == 0 && trimmed == "" {
			continue
		}

		// Stop at first blank line after we've started collecting
		if len(firstParagraph) > 0 && trimmed == "" {
			break
		}

		// Stop at headings (lines starting with #)
		if strings.HasPrefix(trimmed, "#") {
			break
		}

		// Stop at horizontal rules
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			break
		}

		firstParagraph = append(firstParagraph, line)
	}

	return strings.TrimSpace(strings.Join(firstParagraph, " "))
}

// TranslateMarkdownFile translates an entire markdown file to the target language.
func (t *Translator) TranslateMarkdownFile(ctx context.Context, mf *MarkdownFile, targetLang Language) (*MarkdownFile, error) {
	fmt.Printf("  → Translating to %s...", targetLang.Name)

	// Translate content first
	translatedContent, err := t.TranslateText(ctx, mf.Content, mf.SourceLang, targetLang.Code)
	if err != nil {
		return nil, fmt.Errorf("translating content: %w", err)
	}
	translatedContent = restoreAssetReferences(mf.Content, translatedContent)

	// Add translation disclaimer at the end
	disclaimer := getTranslationDisclaimer(targetLang.Code, mf.SourceLang)
	translatedContent = translatedContent + "\n\n" + disclaimer

	// Translate frontmatter (only title, not summary)
	translatedFM, err := t.TranslateFrontmatter(ctx, &mf.Frontmatter, mf.SourceLang, targetLang.Code)
	if err != nil {
		return nil, fmt.Errorf("translating frontmatter: %w", err)
	}

	// Extract first paragraph from translated content and use as summary
	// Note: Escaping is handled by SerializeToMarkdown when writing to file
	translatedFM.Summary = extractFirstParagraph(translatedContent)

	fmt.Println(" ✓")

	return &MarkdownFile{
		Frontmatter: *translatedFM,
		Content:     translatedContent,
		SourceLang:  targetLang.Code,
	}, nil
}

// getTranslationDisclaimer returns a translated disclaimer with link to original.
func getTranslationDisclaimer(targetLang, sourceLang string) string {
	disclaimers := map[string]string{
		"en":   "---\n\n*This blog post has been automatically translated by a Large Language Model.",
		"de":   "---\n\n*Dieser Blogbeitrag wurde automatisch von einem Large Language Model übersetzt.",
		"es":   "---\n\n*Esta publicación de blog ha sido traducida automáticamente por un Large Language Model.",
		"fr":   "---\n\n*Cet article de blog a été traduit automatiquement par un Large Language Model.",
		"it":   "---\n\n*Questo post del blog è stato tradotto automaticamente da un Large Language Model.",
		"arrr": "---\n\n*Arrr, this here blog post be rewritten in the tongue o' pirates by a Large Language Model, ye scallywag!*",
	}

	if disclaimer, ok := disclaimers[targetLang]; ok {
		return disclaimer
	}

	// Fallback to English if language not found
	return disclaimers["en"]
}
