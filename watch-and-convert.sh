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

echo -e "${GREEN}Starting file watcher...${NC}"
echo -e "Input directory: ${YELLOW}$INPUT_DIR${NC}"
echo -e "Output directory: ${YELLOW}$OUTPUT_DIR${NC}"
echo ""

# Function to convert all markdown files
convert_all_files() {
    echo -e "${YELLOW}Change detected! Waiting 3 seconds for additional changes...${NC}"
    sleep 3
    
    echo -e "${GREEN}Converting all markdown files...${NC}"
    
    # Find all .md files in the input directory and subdirectories
    file_count=0
    success_count=0
    error_count=0
    
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
    done < <(find "$INPUT_DIR" -type f -name "*.md" -print0)
    
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
    # Wait for any change in the input directory
    fswatch -1 -r "$INPUT_DIR" > /dev/null
    
    # When a change is detected, run the conversion
    convert_all_files
done
