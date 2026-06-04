#!/bin/bash

# watch-and-convert.sh
# Watches a directory for changes and converts all .md files using the logseq-to-hugo converter
# Cross-platform: supports both macOS (fswatch) and Linux (inotifywait)

set -e

# Store the directory where this script is located (the converter repository)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Detect OS and check for appropriate file watching tool
OS_TYPE=$(uname)

if [[ "$OS_TYPE" == "Darwin" ]]; then
    # macOS
    if ! command -v fswatch &> /dev/null; then
        echo -e "${RED}Error: fswatch is not installed${NC}"
        echo "Please install it with: brew install fswatch"
        exit 1
    fi
    WATCH_TOOL="fswatch"
    echo -e "${GREEN}Detected macOS - using fswatch${NC}"
elif [[ "$OS_TYPE" == "Linux" ]]; then
    # Linux
    if ! command -v inotifywait &> /dev/null; then
        echo -e "${RED}Error: inotifywait is not installed${NC}"
        echo "Please install it with: sudo apt install inotify-tools"
        exit 1
    fi
    WATCH_TOOL="inotifywait"
    echo -e "${GREEN}Detected Linux - using inotifywait${NC}"
else
    echo -e "${RED}Error: Unsupported operating system: $OS_TYPE${NC}"
    echo "This script supports macOS (Darwin) and Linux only."
    exit 1
fi

# Parse -try flag
TRY_MODE=false
if [ "$1" = "-try" ]; then
    TRY_MODE=true
    shift
fi

# Check parameters
if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo -e "${RED}Usage: $0 [-try] <input_directory> <output_directory> [git_repo_directory]${NC}"
    echo "Example: $0 ./logseq/journals ./hugo/content/posts ./hugo"
    echo "Example: $0 -try ./logseq/journals ./hugo/content/posts ./hugo"
    echo ""
    echo "Options:"
    echo "  -try    Do everything except git commit and push (useful for testing)"
    echo ""
    echo "The optional git_repo_directory will be used to automatically commit and push changes."
    exit 1
fi

INPUT_DIR="$1"
OUTPUT_DIR="$2"
GIT_REPO_DIR="${3:-}"

# Validate input directory exists
if [ ! -d "$INPUT_DIR" ]; then
    echo -e "${RED}Error: Input directory '$INPUT_DIR' does not exist${NC}"
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Validate git repository directory if provided
if [ -n "$GIT_REPO_DIR" ]; then
    if [ ! -d "$GIT_REPO_DIR" ]; then
        echo -e "${RED}Error: Git repository directory '$GIT_REPO_DIR' does not exist${NC}"
        exit 1
    fi
    if [ ! -d "$GIT_REPO_DIR/.git" ]; then
        echo -e "${RED}Error: '$GIT_REPO_DIR' is not a git repository${NC}"
        exit 1
    fi
fi

# Define the subdirectories to watch
WATCH_DIRS=("assets" "journals" "pages")

# Validate that at least one watched directory exists
at_least_one_exists=false
for dir in "${WATCH_DIRS[@]}"; do
    if [ -d "$INPUT_DIR/$dir" ]; then
        at_least_one_exists=true
        break
    fi
done

if [ "$at_least_one_exists" = false ]; then
    echo -e "${RED}Error: None of the watched directories (assets/, journals/, pages/) exist in '$INPUT_DIR'${NC}"
    exit 1
fi

echo -e "${GREEN}Starting file watcher...${NC}"
echo -e "Input directory: ${YELLOW}$INPUT_DIR${NC}"
echo -e "Watching subdirectories: ${YELLOW}${WATCH_DIRS[*]}${NC}"
echo -e "Output directory: ${YELLOW}$OUTPUT_DIR${NC}"
if [ -n "$GIT_REPO_DIR" ]; then
    if [ "$TRY_MODE" = true ]; then
        echo -e "Git repository: ${YELLOW}$GIT_REPO_DIR${NC} ${YELLOW}(try mode: no commit or push)${NC}"
    else
        echo -e "Git repository: ${YELLOW}$GIT_REPO_DIR${NC} ${GREEN}(auto-commit enabled)${NC}"
    fi
else
    echo -e "Git repository: ${YELLOW}disabled${NC}"
fi
echo ""

# Function to commit and push git changes
git_commit_and_push() {
    if [ -z "$GIT_REPO_DIR" ]; then
        return
    fi

    if [ "$TRY_MODE" = true ]; then
        echo ""
        echo -e "${YELLOW}[TRY MODE] Skipping git commit and push${NC}"
        return
    fi
    
    echo ""
    echo -e "${YELLOW}Checking for git changes...${NC}"
    
    cd "$GIT_REPO_DIR"
    
    # Check if there are any changes at all
    if ! git diff --quiet || ! git diff --cached --quiet || [ -n "$(git ls-files --others --exclude-standard)" ]; then

        # Check whether anything changed beyond the journey map files.
        # Changes limited to journey-map.mp4 / journey.json alone do not
        # warrant a deployment, so we skip the commit in that case.
        # grep exits with code 1 when all lines are filtered out. Under `set -e`
        # that would abort the script, so we treat "no matches" as a valid result.
        non_journey_changes=$(git status --porcelain | grep -v -E '(static/journey-map\.mp4|data/journey\.json)$' || true)

        if [ -z "$non_journey_changes" ]; then
            echo -e "${YELLOW}Only journey map files changed — skipping commit to avoid unnecessary deployment${NC}"
        else
            echo -e "${GREEN}Changes detected, committing...${NC}"
            
            # Add all changes (journey map included when something else also changed)
            git add --all
            
            # Commit with message
            git commit -m "automatic change by logseq-to-hugo-converter"
            
            echo -e "${YELLOW}Pushing to remote...${NC}"
            if git push; then
                echo -e "${GREEN}Successfully pushed changes to remote${NC}"
            else
                echo -e "${RED}Failed to push changes${NC}"
            fi
        fi
    else
        echo -e "${YELLOW}No git changes detected${NC}"
    fi
    
    # Return to original directory
    cd - > /dev/null
}

# Function to translate changed markdown files
translate_changed_files() {
    if [ -z "$GIT_REPO_DIR" ]; then
        return
    fi
    
    echo ""
    echo -e "${YELLOW}🌍 Translating changed markdown files...${NC}"
    
    cd "$GIT_REPO_DIR"
    
    # Get list of new or modified .md files using git status --porcelain -z.
    # The NUL-delimited format avoids quoted/escaped paths for non-ASCII names
    # (e.g. "Törn"), so regex and basename matching remain reliable.
    # --untracked-files=all ensures files inside brand-new directories are listed
    # individually (without it, git collapses a new dir to "?? dir/" which doesn't
    # match the \.md$ pattern and causes translation to be silently skipped).
    # Format: XY<space>path\0
    # A  = new file (staged)
    # M  = modified (staged)
    #  M = modified (unstaged)
    # MM = modified, staged, then modified again
    changed_files=()
    while IFS= read -r -d '' entry; do
        status="${entry:0:2}"
        file="${entry:3}"

        if [[ "$status" =~ ^(A |M |\ M|MM|\?\?)$ ]] && [[ "$file" == *.md ]]; then
            changed_files+=("$file")
        fi
    done < <(git status --porcelain -z --untracked-files=all)
    
    if [ ${#changed_files[@]} -eq 0 ]; then
        echo -e "${YELLOW}No .md files to translate${NC}"
        cd - > /dev/null
        return
    fi
    
    # Filter for files that match index.<lang>.md pattern
    translate_files=()
    for file in "${changed_files[@]}"; do
        basename=$(basename "$file")
        if [[ "$basename" =~ ^index\.[a-z]{2}\.md$ ]]; then
            translate_files+=("$file")
        fi
    done
    
    if [ ${#translate_files[@]} -eq 0 ]; then
        echo -e "${YELLOW}No index.<lang>.md files to translate${NC}"
        cd - > /dev/null
        return
    fi
    
    echo -e "Found ${GREEN}${#translate_files[@]}${NC} .md file(s) to translate:"
    for file in "${translate_files[@]}"; do
        echo -e "  - $file"
    done
    echo ""
    
    # Translate each file
    success_count=0
    error_count=0
    
    for file in "${translate_files[@]}"; do
        echo -e "${YELLOW}Translating:${NC} $file"
        
        # Get absolute path for the file
        abs_file_path="$GIT_REPO_DIR/$file"
        
        # Run the translate command from the converter directory
        # Use package path to compile all source files (excluding tests)
        if (cd "$SCRIPT_DIR" && go run ./cmd/translate "$abs_file_path") 2>&1; then
            success_count=$((success_count+1))
        else
            error_count=$((error_count+1))
            echo -e "${RED}  ✗ Failed to translate: $file${NC}"
        fi
        echo ""
    done
    
    echo -e "${GREEN}✅ Translation complete: $success_count/${#translate_files[@]} files translated successfully${NC}"
    if [ $error_count -gt 0 ]; then
        echo -e "${RED}Errors: $error_count${NC}"
    fi
    
    # Return to original directory
    cd - > /dev/null
}

# Flag to track if this is the first run
FIRST_RUN=true

# Function to convert all markdown files
convert_all_files() {
    if [ "$FIRST_RUN" = true ]; then
        echo -e "${GREEN}Running immediate conversion on startup...${NC}"
        FIRST_RUN=false
    else
        echo -e "${YELLOW}Change detected! Waiting 30 minutes for additional changes...${NC}"
        sleep 1800  # 30 minutes = 1800 seconds
    fi
    
    echo -e "${GREEN}Converting all markdown files...${NC}"
    
    # Find all .md files in the watched subdirectories
    file_count=0
    success_count=0
    error_count=0
    
    # Build find command to search only in watched directories
    find_paths=()
    for dir in "${WATCH_DIRS[@]}"; do
        if [ -d "$INPUT_DIR/$dir" ]; then
            find_paths+=("$INPUT_DIR/$dir")
        fi
    done
    
    # Only process if we have directories to search
    if [ ${#find_paths[@]} -eq 0 ]; then
        echo -e "${YELLOW}No watched directories found, skipping...${NC}"
        return
    fi
    
    while IFS= read -r -d '' md_file; do
        file_count=$((file_count+1))
        echo -e "\n${YELLOW}Processing:${NC} $md_file"
        
        # Run the converter from the script's directory
        # Use 'go run .' to compile all Go files in the directory, not just main.go
        if (cd "$SCRIPT_DIR" && go run . "$md_file" "$OUTPUT_DIR") 2>&1; then
            success_count=$((success_count+1))
        else
            error_count=$((error_count+1))
            echo -e "${RED}Failed to convert: $md_file${NC}"
        fi
    done < <(find "${find_paths[@]}" -type f -name "*.md" -print0)
    
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Conversion complete!${NC}"
    echo -e "Total files: $file_count"
    echo -e "Successful: ${GREEN}$success_count${NC}"
    if [ $error_count -gt 0 ]; then
        echo -e "Errors: ${RED}$error_count${NC}"
    else
        echo -e "Errors: $error_count"
    fi
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # Generate animated journey map if git repo (Hugo site) is configured
    if [ -n "$GIT_REPO_DIR" ]; then
        echo ""
        echo -e "${YELLOW}Generating animated journey map...${NC}"
        JOURNEY_JSON="$GIT_REPO_DIR/data/journey.json"
        JOURNEY_MP4="$GIT_REPO_DIR/static/journey-map.mp4"
        if (cd "$SCRIPT_DIR" && go run ./cmd/journeymap "$INPUT_DIR/journals" "$JOURNEY_JSON") 2>&1; then
            if [ -f "$JOURNEY_JSON" ]; then
                if (cd "$SCRIPT_DIR" && go run ./cmd/animatemap "$JOURNEY_JSON" "$JOURNEY_MP4") 2>&1; then
                    echo -e "${GREEN}Animated journey map written to $JOURNEY_MP4${NC}"
                else
                    echo -e "${RED}Failed to render animated journey map${NC}"
                fi
            fi
        else
            echo -e "${RED}Failed to extract journey positions${NC}"
        fi
    fi

    # Translate changed files before committing
    translate_changed_files
    
    # Commit and push changes if git repository is configured
    git_commit_and_push
    
    echo ""
    echo -e "${YELLOW}Watching for changes... (Press Ctrl+C to stop)${NC}"
}

# Initial conversion on startup
echo -e "${YELLOW}Running initial conversion...${NC}"
convert_all_files

# Watch for changes and trigger conversion with debouncing
while true; do
    # Build list of directories to watch
    watch_paths=()
    for dir in "${WATCH_DIRS[@]}"; do
        if [ -d "$INPUT_DIR/$dir" ]; then
            watch_paths+=("$INPUT_DIR/$dir")
        fi
    done
    
    # Watch for any change in the watched directories
    if [ ${#watch_paths[@]} -gt 0 ]; then
        # Use the appropriate file watching tool based on OS
        if [[ "$WATCH_TOOL" == "fswatch" ]]; then
            # macOS: -1 flag makes fswatch exit after first event, so we can debounce in our loop
            fswatch -1 -r "${watch_paths[@]}" > /dev/null
        else
            # Linux: Wait for any file system event (modify, create, delete, move)
            # -r: recursive, -e: events to watch, -q: quiet (don't print events)
            inotifywait -r -e modify,create,delete,move -q "${watch_paths[@]}"
        fi
        
        # When a change is detected, run the conversion
        convert_all_files
    else
        echo -e "${RED}No directories to watch. Exiting.${NC}"
        exit 1
    fi
done
