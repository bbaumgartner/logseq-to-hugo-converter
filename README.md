# logseq-to-hugo-converter
Takes a logseq md file and converts special annotated lists to a blog post ready to be served with HUGO.

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

### Platform-Specific Tools

#### macOS
Install `fswatch` for file watching:
```bash
brew install fswatch
```

#### Linux/Ubuntu
Install `inotify-tools` for file watching:
```bash
sudo apt install inotify-tools
```

## Usage

### Running the File Watcher

**On macOS:**
```bash
./watch-and-convert.sh <input_directory> <output_directory> [git_repo_directory]
```

**On Linux/Ubuntu:**
```bash
./watch-and-convert-linux.sh <input_directory> <output_directory> [git_repo_directory]
```

**Example:**
```bash
# macOS
./watch-and-convert.sh /logseq-data ../hugo-data/content/posts/ ../hugo-data

# Linux
./watch-and-convert-linux.sh /logseq-data ../hugo-data/content/posts/ ../hugo-data
```

**Parameters:**
- `input_directory`: Path to your Logseq data directory
- `output_directory`: Where converted blog posts should be written
- `git_repo_directory` (optional): Git repository to automatically commit and push changes to

### Manual Conversion

You can also convert individual files without the watcher:

```bash
go run main.go <input_file.md> <output_directory>
```

**Example:**
```bash
go run main.go examples/journals/2026_01_17.md ./output
```

### Requirements for Blog Posts

For a Logseq list to be converted, it must include:
- `type:: blog` - Marks the list as a blog post
- `status:: online` - Only posts with this status are converted
- `date:: YYYY-MM-DD` - Publication date
- `title:: Your Title` - Post title
- `author:: Author Name` - Author name

See the [example](2026-01-17_Frühlingspläne_2026/index.md) for a converted blog post.
