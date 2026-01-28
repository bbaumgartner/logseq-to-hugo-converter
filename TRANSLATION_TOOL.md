# Hugo Markdown Translation Tool

A standalone Go tool that translates Hugo markdown blog posts to multiple languages using OpenAI GPT-4-turbo.

## Features

- Translates Hugo markdown files to Spanish, English, German, French, and Italian
- Automatically detects source language from filename
- Skips translating to the source language
- Translates both frontmatter (title, summary) and content
- Preserves markdown formatting, images, links, and Hugo shortcodes
- Outputs translated files in the same directory as the input file

## Prerequisites

1. **Go** (version 1.25 or higher) - Already installed for the main converter
2. **OpenAI API Key** - Get one from [platform.openai.com](https://platform.openai.com)

## Installation

The tool uses the same dependencies as the main converter. Dependencies are already installed.

To compile the translation tool:

```bash
cd cmd/translate
go build -o translate .
```

This creates a `translate` binary in the `cmd/translate/` directory.

## Configuration

Set your OpenAI API key as an environment variable:

```bash
export OPENAI_API_KEY='sk-your-api-key-here'
```

For permanent configuration, add it to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
echo "export OPENAI_API_KEY='sk-your-api-key-here'" >> ~/.zshrc
source ~/.zshrc
```

## Usage

### Basic Usage

From the repository root:

```bash
go run ./cmd/translate <path-to-index-file>
```

Or if you've compiled the binary:

```bash
cd cmd/translate
./translate <path-to-index-file>
```

### Examples

**Translate a German blog post:**
```bash
go run ./cmd/translate 2025-09-13_SKS/index.de.md
```

**Output:**
```
📖 Parsing 2025-09-13_SKS/index.de.md...
✓ Detected source language: German

🌍 Translating from German to 4 languages...
  → Translating to English... ✓
  ✓ Created: 2025-09-13_SKS/index.en.md
  → Translating to Spanish... ✓
  ✓ Created: 2025-09-13_SKS/index.es.md
  → Translating to French... ✓
  ✓ Created: 2025-09-13_SKS/index.fr.md
  → Translating to Italian... ✓
  ✓ Created: 2025-09-13_SKS/index.it.md

✅ Successfully translated to 4/4 languages
```

**Translate an English blog post:**
```bash
go run ./cmd/translate 2024-06-14_Renan/index.en.md
```

This will create German, Spanish, French, and Italian versions (skipping English since it's the source).

## Input File Requirements

Input files must:
1. Be named in format: `index.<lang>.md` (e.g., `index.de.md`, `index.en.md`)
2. Have TOML frontmatter enclosed in `+++` markers
3. Contain at least these frontmatter fields:
   - `date` - Publication date
   - `title` - Post title (will be translated)
   - `summary` - Post summary (will be translated)
   - Other fields (`lastmod`, `draft`, `params.author`) are preserved as-is

## Supported Languages

| Language Code | Language Name |
|---------------|---------------|
| `en` | English |
| `de` | German |
| `es` | Spanish |
| `fr` | French |
| `it` | Italian |

## Translation Behavior

### What Gets Translated
- Frontmatter `title` field
- Frontmatter `summary` field
- All markdown content (paragraphs, lists, headings, etc.)

### What Gets Preserved
- Frontmatter fields: `date`, `lastmod`, `draft`, `params.*`
- Markdown formatting (bold, italic, links, images, etc.)
- Hugo shortcodes (e.g., `{{< video src="..." >}}`)
- File paths and URLs
- Proper nouns (kept in original form unless commonly translated)

## Cost Estimation

Translation costs depend on content length. Approximate costs with GPT-4-turbo:
- Short blog post (~500 words): ~$0.05-0.10 per language
- Medium blog post (~1500 words): ~$0.15-0.30 per language
- Long blog post (~3000 words): ~$0.30-0.60 per language

Each translation to 4 languages costs approximately 4x the per-language rate.

## Troubleshooting

### Error: "OPENAI_API_KEY environment variable not set"
Make sure you've exported your API key:
```bash
export OPENAI_API_KEY='sk-...'
```

### Error: "File not found"
Verify the file path is correct and the file exists:
```bash
ls -la 2025-09-13_SKS/index.de.md
```

### Error: "could not detect language from filename"
Ensure your file follows the naming pattern: `index.<lang>.md`
- ✅ Correct: `index.de.md`, `index.en.md`
- ❌ Incorrect: `blog.de.md`, `index-de.md`, `index.md`

### API Rate Limits
If you hit rate limits, the tool will automatically retry with exponential backoff (3 attempts).

## Advanced Usage

### Batch Translation Script

Create a script to translate multiple blog posts:

```bash
#!/bin/bash
# translate_all.sh

for file in */index.de.md; do
    echo "Translating $file..."
    go run ./cmd/translate "$file"
    echo ""
done
```

Make it executable and run:
```bash
chmod +x translate_all.sh
./translate_all.sh
```

### Integration with Main Converter

You can integrate translation into your workflow:

```bash
# 1. Convert from Logseq to Hugo
go run . examples/journals/2026_01_17.md ./output

# 2. Translate the generated blog post
go run ./cmd/translate ./output/2026-01-17_*/index.de.md
```

## Technical Details

### Architecture
The translation tool is located in `cmd/translate/`:
- `translate.go` - Main CLI entry point
- `translate_parser.go` - Parses TOML frontmatter and markdown content
- `translate_llm.go` - Handles OpenAI API integration
- `translate_writer.go` - Writes translated files to disk

### Model Configuration
- Model: `gpt-4-turbo`
- Temperature: 0.3 (deterministic translations)
- Retry attempts: 3
- Timeout: 10 minutes per translation run

## Project Structure

The translation tool is located in the `cmd/translate/` directory:
```
cmd/translate/
├── translate.go          # Main program
├── translate_parser.go   # File parsing
├── translate_llm.go      # OpenAI integration
└── translate_writer.go   # File writing
```

This separation allows both tools (converter and translator) to coexist without conflicts.
