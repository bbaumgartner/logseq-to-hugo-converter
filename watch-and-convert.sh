#!/bin/bash

# watch-and-convert.sh
# Watches a directory for changes and converts all .md files using the logseq-to-hugo converter

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if fswatch is installed
if ! command -v fswatch &> /dev/null; then
    echo -e "${RED}Error: fswatch is not installed${NC}"
    echo "Please install it with: brew install fswatch"
    exit 1
fi

# Check parameters
if [ "$#" -ne 2 ]; then
    echo -e "${RED}Usage: $0 <input_directory> <output_directory>${NC}"
    echo "Example: $0 ./logseq/journals ./hugo/content/posts"
    exit 1
fi

INPUT_DIR="$1"
OUTPUT_DIR="$2"

# Validate input directory exists
if [ ! -d "$INPUT_DIR" ]; then
    echo -e "${RED}Error: Input directory '$INPUT_DIR' does not exist${NC}"
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

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
echo ""

# Function to convert all markdown files
convert_all_files() {
    echo -e "${YELLOW}Change detected! Waiting 3 seconds for additional changes...${NC}"
    sleep 3
    
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
        ((file_count++))
        echo -e "\n${YELLOW}Processing:${NC} $md_file"
        
        # Run the converter
        if go run main.go "$md_file" "$OUTPUT_DIR" 2>&1; then
            ((success_count++))
        else
            ((error_count++))
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
    echo ""
    echo -e "${YELLOW}Watching for changes... (Press Ctrl+C to stop)${NC}"
}

# Initial conversion on startup
echo -e "${YELLOW}Running initial conversion...${NC}"
convert_all_files

# Watch for changes and trigger conversion with debouncing
# The -1 flag makes fswatch exit after first event, so we can debounce in our loop
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
        fswatch -1 -r "${watch_paths[@]}" > /dev/null
        
        # When a change is detected, run the conversion
        convert_all_files
    else
        echo -e "${RED}No directories to watch. Exiting.${NC}"
        exit 1
    fi
done
