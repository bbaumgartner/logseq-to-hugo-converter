package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertLogseqToHugo(t *testing.T) {
	// Setup: paths to test files
	inputPath := "examples/journals/2026_01_17.md"
	expectedOutputDir := "2026-01-17_Frühlingspläne_2026"
	
	// Create a temporary directory for test output
	tempDir := t.TempDir()
	
	// Run the conversion
	outputPath, err := convertLogseqToHugo(inputPath, tempDir)
	if err != nil {
		t.Fatalf("convertLogseqToHugo() error = %v", err)
	}
	
	// Verify the output directory was created with the expected name
	expectedDirName := filepath.Base(expectedOutputDir)
	actualDirName := filepath.Base(outputPath)
	if actualDirName != expectedDirName {
		t.Errorf("Output directory name = %v, want %v", actualDirName, expectedDirName)
	}
	
	// Test 1: Verify index.md exists
	indexPath := filepath.Join(outputPath, "index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("index.md does not exist at %s", indexPath)
	}
	
	// Test 2: Compare index.md content with expected output
	actualContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read generated index.md: %v", err)
	}
	
	expectedIndexPath := filepath.Join(expectedOutputDir, "index.md")
	expectedContent, err := os.ReadFile(expectedIndexPath)
	if err != nil {
		t.Fatalf("Failed to read expected index.md: %v", err)
	}
	
	actualStr := strings.TrimSpace(string(actualContent))
	expectedStr := strings.TrimSpace(string(expectedContent))
	
	if actualStr != expectedStr {
		t.Errorf("index.md content mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedStr, actualStr)
	}
	
	// Test 3: Verify all expected image files exist
	expectedImages := []string{
		"featured.jpeg",
		"image_1768654728313_0.png",
		"image_1768655067995_0.png",
		"image_1768655164867_0.png",
		"image_1768655591886_0.png",
		"image_1768656457958_0.png",
	}
	
	for _, imgName := range expectedImages {
		actualImgPath := filepath.Join(outputPath, imgName)
		expectedImgPath := filepath.Join(expectedOutputDir, imgName)
		
		// Check if file exists
		actualInfo, err := os.Stat(actualImgPath)
		if os.IsNotExist(err) {
			t.Errorf("Expected image %s does not exist", imgName)
			continue
		}
		
		// Check if file size matches expected
		expectedInfo, err := os.Stat(expectedImgPath)
		if err != nil {
			t.Logf("Warning: Could not stat expected image %s: %v", imgName, err)
			continue
		}
		
		if actualInfo.Size() != expectedInfo.Size() {
			t.Errorf("Image %s size mismatch: got %d bytes, want %d bytes", 
				imgName, actualInfo.Size(), expectedInfo.Size())
		}
	}
	
	// Test 4: Verify no unexpected files were created
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}
	
	expectedFiles := append(expectedImages, "index.md")
	expectedFileMap := make(map[string]bool)
	for _, f := range expectedFiles {
		expectedFileMap[f] = true
	}
	
	for _, entry := range entries {
		if !expectedFileMap[entry.Name()] {
			t.Errorf("Unexpected file in output: %s", entry.Name())
		}
	}
}

func TestConvertLogseqToHugo_InvalidInput(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test with non-existent file
	_, err := convertLogseqToHugo("nonexistent.md", tempDir)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestConvertLogseqToHugo_NoBlogMarker(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a test file without blog marker
	testFile := filepath.Join(tempDir, "test.md")
	content := []byte("# Some heading\n\nSome content without blog marker\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	_, err := convertLogseqToHugo(testFile, tempDir)
	if err == nil {
		t.Error("Expected error for file without blog marker, got nil")
	}
	
	expectedErrMsg := "no list starting with 'type:: blog' found"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrMsg, err.Error())
	}
}
