# logseq-to-hugo-converter
Takes a logseq md file and converts special annotated lists to a blog post ready to be served with HUGO. Includes automatic translation to multiple languages (English, German, Spanish, French, Italian).

We use logseq for our log book and wanted to also be able to create blog post right out of the log book. See https://sailingnomads.ch for the blog.

For example, having a logseq page or journal at /logseq-data with following form:

![example.png](example.png)

## Installation

### Prerequisites

This converter requires **Go** (Golang) to be installed on your system.

#### Installing Go on macOS

**Option 1: Using Homebrew (Recommended)**
```bash
brew install go
```

**Option 2: Official Installer**
1. Download the macOS installer from [golang.org/dl](https://golang.org/dl/)
2. Open the downloaded `.pkg` file and follow the installation prompts
3. Go will be installed to `/usr/local/go` by default

**Verify Installation:**
```bash
go version
```

#### Installing Go on Linux/Ubuntu

**Option 1: Using apt (Easier, but may not be the latest version)**
```bash
sudo apt update
sudo apt install golang-go
```

**Option 2: Official Binary (Recommended for latest version)**
```bash
# Download and extract (replace 1.22.0 with the latest version from golang.org/dl)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Add Go to PATH (add this to ~/.bashrc or ~/.profile for persistence)
export PATH=$PATH:/usr/local/go/bin

# Reload your shell configuration
source ~/.bashrc
```

**Verify Installation:**
```bash
go version
```

### Installing Dependencies

Once Go is installed, fetch the required Go modules:

```bash
cd logseq-to-hugo-converter
go mod download
```

### File Watching Tools

The `watch-and-convert.sh` script automatically detects your operating system and uses the appropriate file watching tool. Install the tool for your platform:

#### macOS
Install `fswatch`:
```bash
brew install fswatch
```

#### Linux/Ubuntu
Install `inotify-tools`:
```bash
sudo apt install inotify-tools
```

The script will automatically use the correct tool based on your OS.

### OpenAI API Key (for automatic translation)

To enable automatic translation of blog posts, you need to set the `OPENAI_API_KEY` environment variable:

```bash
export OPENAI_API_KEY='sk-...'
```

You can get an API key from [OpenAI](https://platform.openai.com/api-keys).

**To make it persistent**, add the export command to your shell configuration:
- macOS/Linux (bash): Add to `~/.bashrc` or `~/.bash_profile`
- macOS/Linux (zsh): Add to `~/.zshrc`

```bash
echo "export OPENAI_API_KEY='sk-...'" >> ~/.zshrc
source ~/.zshrc
```

**Note:** Translation is optional. If the API key is not set, the script will skip translation and only perform conversion.

### Running Tests

To verify the installation and ensure everything is working correctly, run the test suite:

```bash
# Run all tests
go test

# Run tests with verbose output
go test -v
```


## Usage

### Running the File Watcher

The `watch-and-convert.sh` script is cross-platform and works on both macOS and Linux. It automatically detects your operating system and uses the appropriate file watching tool (`fswatch` on macOS, `inotifywait` on Linux).

```bash
./watch-and-convert.sh [-try] <input_directory> <output_directory> [git_repo_directory]
```

**Examples:**
```bash
# Normal mode (with automatic translation and git push)
./watch-and-convert.sh /logseq-data ../hugo-data/content/posts/ ../hugo-data

# Try mode (with automatic translation but no git push - useful for testing)
./watch-and-convert.sh -try /logseq-data ../hugo-data/content/posts/ ../hugo-data
```

**Parameters:**
- `-try` (optional): Do everything except git push (useful for testing)
- `input_directory`: Path to your Logseq data directory
- `output_directory`: Where converted blog posts should be written
- `git_repo_directory` (optional): Git repository to automatically commit and push changes to

**Features:**
- Automatically detects OS (macOS/Linux) and uses the appropriate file watching tool
- Monitors changes in `assets/`, `journals/`, and `pages/` subdirectories
- Waits 30 minutes after detecting changes to batch multiple edits together
- **Automatic translation**: Detects new or changed markdown files and translates them to all supported languages (English, German, Spanish, French, Italian)
- Optionally commits and pushes changes to a git repository
- Try mode (`-try` flag) for testing without pushing to remote

**Workflow:**
1. Watches for changes in Logseq directories
2. Converts all markdown files to Hugo format
3. **Automatically translates** any new or modified `index.<lang>.md` files using the translation tool
4. Commits all changes (conversions + translations)
5. Pushes to remote (unless `-try` flag is used)

### Manual Conversion

You can also convert individual files without the watcher:

```bash
go run . <input_file.md> <output_directory>
```

**Example:**
```bash
go run . examples/journals/2026_01_17.md ./output
```

**Note:** Use `go run .` (dot) to compile all source files, not just `main.go`.

### Automatic Translation

When using the file watcher with a git repository configured, the script automatically translates any new or modified markdown files after conversion. This feature requires:

1. **OpenAI API Key**: Set the `OPENAI_API_KEY` environment variable:
   ```bash
   export OPENAI_API_KEY='sk-...'
   ```

2. **File naming convention**: Only files matching the pattern `index.<lang>.md` are translated (e.g., `index.de.md`, `index.en.md`)

3. **Supported languages**: English (en), German (de), Spanish (es), French (fr), Italian (it)

**How it works:**
- After converting files, the script runs `git status` to detect new or modified `.md` files
- Only changed files are translated (avoiding expensive re-translations)
- For each source file (e.g., `index.de.md`), translations are automatically created for all other languages
- All translations are included in the same git commit

**Manual translation:**
You can also translate individual files manually:
```bash
go run ./cmd/translate/translate.go <input_file.md>
```

**Example:**
```bash
go run ./cmd/translate/translate.go 2025-09-13_SKS/index.de.md
```

This will create `index.en.md`, `index.es.md`, `index.fr.md`, and `index.it.md` in the same directory.

For more details, see [TRANSLATION_TOOL.md](TRANSLATION_TOOL.md).

### Requirements for Blog Posts

All blog posts must include the following metadata fields:
- `type:: blog` - Marks the content as a blog post
- `status:: online` - Only posts with this status are converted (draft posts are ignored)
- `date:: YYYY-MM-DD` - Publication date
- `title:: Your Title` - Post title
- `author:: Author Name` - Author name
- `header:: ![image](path/to/image.jpg)` - (Optional) Featured image

## Supported Formats

The converter supports two different Logseq formats:

### Format 1: Nested List Structure (Journals)

This format is commonly used in Logseq journals where you organize content under topic headings.

```markdown
- [[Blog]]
  - type:: blog
    status:: online
    date:: 2026-01-17
    title:: Spring Plans 2026
    author:: benno
    header:: ![image](../assets/featured.jpg)
  - First paragraph of content
  - ## Section Heading
  - More content here
  - Another paragraph
```

**Key characteristics:**
- Metadata is in the first list item
- Content follows as subsequent list items
- Each list item becomes a paragraph in the output

**Example:** [examples/journals/2026_01_17.md](examples/journals/2026_01_17.md) → [2026-01-17_Frühlingspläne_2026/index.md](2026-01-17_Frühlingspläne_2026/index.md)

### Format 2: Top-Level Metadata (Pages)

This format places metadata at the top of the file, followed by list items for content.

```markdown
type:: blog
status:: online
date:: 2024-06-14
title:: My Blog Post
author:: Author Name
header:: ![image](../assets/header.jpg)

- First paragraph of content
- Second paragraph
- ![image](../assets/photo.jpg)
- More content
```

**Key characteristics:**
- Metadata fields at the top level (not in a list)
- Content organized as list items below the metadata
- Clean separation between metadata and content

**Example:** [examples/pages/Renan.md](examples/pages/Renan.md) → [2024-06-14_Renan/index.md](2024-06-14_Renan/index.md)

## Software Design

### Architecture

The converter uses a simple, functional approach with clear separation of concerns:

```plantuml
@startuml
!theme plain

package "Core Types" {
  class BlogMeta {
    +Date: string
    +Title: string
    +Author: string
    +Header: string
    +Summary: string
    +Status: string
  }
  
  class BlogPost {
    +Meta: BlogMeta
    +Content: []string
  }
}

package "Extraction" {
  class "extractor.go" as Extractor {
    +extractBlogPosts(doc, source) []*BlogPost
    +extractListPost(...) *BlogPost
    +extractTopLevelPost(...) *BlogPost
    +extractText(node, source) string
  }
}

package "Processing" {
  class MetadataParser {
    -regex: Regexp
    +Parse(lines) BlogMeta
    -setField(meta, key, value)
  }
  
  class ImageProcessor {
    -inputDir: string
    -outputDir: string
    +ProcessContent(content) string
    +ProcessHeaderImage(path)
  }
  
  class HugoWriter {
    -outputDir: string
    +Write(meta, content) error
  }
}

package "Main" {
  class "main.go" as Main {
    +convertFile(inputPath, outputBasePath) ([]string, error)
    -createOutputDir(basePath, meta) string
    -buildContent(blocks) string
  }
}

Main --> Extractor : uses
Main --> ImageProcessor : creates
Main --> HugoWriter : creates
Extractor --> MetadataParser : uses
Extractor ..> BlogPost : returns
BlogPost *-- BlogMeta : contains
MetadataParser ..> BlogMeta : creates

note right of Main
  Entry point that:
  1. Reads markdown file
  2. Extracts all blog posts
  3. Filters by status
  4. Processes each post
  5. Writes Hugo output
end note

note right of Extractor
  Handles both formats:
  - List-based (journals)
  - Top-level metadata (pages)
  Supports arbitrary nesting
end note

@enduml
```

### File Structure

```
📁 logseq-to-hugo-converter/
├── main.go              ⭐ Entry point & conversion logic (119 lines)
├── types.go             📋 Data structures (22 lines)
├── metadata.go          🏷️  Metadata parsing (105 lines)
├── extractor.go         🔍 Blog extraction (204 lines)
├── processors.go        🖼️  Image/video processing (215 lines)
├── writer.go            📝 Hugo format writing (158 lines)
├── main_test.go         ✅ Tests (364 lines)
├── cmd/translate/       🌍 Translation tool
│   ├── translate.go     📝 CLI entry point (127 lines)
│   ├── translate_llm.go 🤖 OpenAI integration (190 lines)
│   ├── translate_parser.go 📖 Markdown parsing (170 lines)
│   └── translate_writer.go 💾 File writing (75 lines)
├── test-nesting.md      📄 Deep nesting test
├── test-multiple.md     📄 Multiple posts test
└── watch-and-convert.sh 👀 File watcher + auto-translation (341 lines)
```

**Total:** ~1,471 lines of code (excluding tests)

### Design Principles

- **Simplicity**: Direct function calls, no unnecessary abstractions
- **Single Responsibility**: Each file has one clear purpose
- **Extensibility**: Easy to add new metadata fields or processing steps
- **Testability**: Pure functions with clear inputs/outputs