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
	expectedFilename := "index.de.md" // German language

	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Run the conversion using the real convertFile function
	outputs, err := convertFile(inputPath, tempDir)
	if err != nil {
		t.Fatalf("convertFile() error = %v", err)
	}

	if len(outputs) == 0 {
		t.Fatalf("convertFile() returned no outputs")
	}

	output := outputs[0]

	// Verify the output directory was created with the expected name
	expectedDirName := filepath.Base(expectedOutputDir)
	actualDirName := filepath.Base(output.Dir)
	if actualDirName != expectedDirName {
		t.Errorf("Output directory name = %v, want %v", actualDirName, expectedDirName)
	}

	// Verify the correct filename was created
	if output.Filename != expectedFilename {
		t.Errorf("Output filename = %v, want %v", output.Filename, expectedFilename)
	}

	// Test 1: Verify the index file exists
	indexPath := filepath.Join(output.Dir, expectedFilename)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("%s does not exist at %s", expectedFilename, indexPath)
	}

	// Test 2: Compare content with expected output
	actualContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read generated %s: %v", expectedFilename, err)
	}

	expectedIndexPath := filepath.Join(expectedOutputDir, expectedFilename)
	expectedContent, err := os.ReadFile(expectedIndexPath)
	if err != nil {
		t.Fatalf("Failed to read expected %s: %v", expectedFilename, err)
	}

	actualStr := strings.TrimSpace(string(actualContent))
	expectedStr := strings.TrimSpace(string(expectedContent))

	if actualStr != expectedStr {
		t.Errorf("%s content mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedFilename, expectedStr, actualStr)
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
		actualImgPath := filepath.Join(output.Dir, imgName)
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
	entries, err := os.ReadDir(output.Dir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	expectedFiles := append(expectedImages, expectedFilename)
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
	_, err := convertFile("nonexistent.md", tempDir)
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

	_, err := convertFile(testFile, tempDir)
	if err == nil {
		t.Error("Expected error for file without blog marker, got nil")
	}

	expectedErrMsg := "no blog post found with 'type:: blog' marker"
	if err != nil && !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestConvertLogseqToHugo_StatusNotOnline(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file with blog marker but status is "draft"
	testFile := filepath.Join(tempDir, "test.md")
	content := []byte(`- [[Blog]]
	- type:: blog
	  status:: draft
	  date:: 2026-01-17
	  title:: Test Post
	  author:: test
	- This is a test post
`)
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// convertFile should skip the draft post and return empty outputs
	outputs, err := convertFile(testFile, tempDir)
	if err != nil {
		t.Fatalf("convertFile() error = %v, expected no error", err)
	}

	if len(outputs) != 0 {
		t.Errorf("Expected no output for blog with status 'draft', got %d outputs", len(outputs))
	}
}

func TestConvertLogseqToHugo_RenanExample(t *testing.T) {
	// Test conversion of the Renan.md example file which uses top-level metadata format
	tempDir := t.TempDir()

	inputPath := "examples/pages/Renan.md"
	expectedOutputDir := "2024-06-14_Renan"
	expectedFilename := "index.en.md" // English language

	// Run the conversion using the real convertFile function
	outputs, err := convertFile(inputPath, tempDir)
	if err != nil {
		t.Fatalf("convertFile() error = %v", err)
	}

	if len(outputs) == 0 {
		t.Fatalf("convertFile() returned no outputs")
	}

	output := outputs[0]

	// Verify the output directory was created with the expected name
	expectedDirName := filepath.Base(expectedOutputDir)
	actualDirName := filepath.Base(output.Dir)
	if actualDirName != expectedDirName {
		t.Errorf("Output directory name = %v, want %v", actualDirName, expectedDirName)
	}

	// Verify the correct filename was created
	if output.Filename != expectedFilename {
		t.Errorf("Output filename = %v, want %v", output.Filename, expectedFilename)
	}

	// Verify the index file exists
	indexPath := filepath.Join(output.Dir, expectedFilename)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("%s does not exist at %s", expectedFilename, indexPath)
	}

	// Read and verify the generated content
	actualContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read generated %s: %v", expectedFilename, err)
	}

	expectedIndexPath := filepath.Join(expectedOutputDir, expectedFilename)
	expectedContent, err := os.ReadFile(expectedIndexPath)
	if err != nil {
		t.Fatalf("Failed to read expected %s: %v", expectedFilename, err)
	}

	actualStr := strings.TrimSpace(string(actualContent))
	expectedStr := strings.TrimSpace(string(expectedContent))

	if actualStr != expectedStr {
		t.Errorf("%s content mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedFilename, expectedStr, actualStr)
	}
}

func TestConvertLogseqToHugo_SKSExample(t *testing.T) {
	// Setup: paths to test files
	inputPath := "examples/journals/2026_01_23.md"
	expectedOutputDir := "2025-09-13_SKS"
	expectedFilename := "index.de.md" // German language

	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Run the conversion using the real convertFile function
	outputs, err := convertFile(inputPath, tempDir)
	if err != nil {
		t.Fatalf("convertFile() error = %v", err)
	}

	if len(outputs) == 0 {
		t.Fatalf("convertFile() returned no outputs")
	}

	output := outputs[0]

	// Verify the output directory was created with the expected name
	expectedDirName := filepath.Base(expectedOutputDir)
	actualDirName := filepath.Base(output.Dir)
	if actualDirName != expectedDirName {
		t.Errorf("Output directory name = %v, want %v", actualDirName, expectedDirName)
	}

	// Verify the correct filename was created
	if output.Filename != expectedFilename {
		t.Errorf("Output filename = %v, want %v", output.Filename, expectedFilename)
	}

	// Verify the index file exists
	indexPath := filepath.Join(output.Dir, expectedFilename)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("%s does not exist at %s", expectedFilename, indexPath)
	}

	// Compare content with expected output
	actualContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read generated %s: %v", expectedFilename, err)
	}

	expectedIndexPath := filepath.Join(expectedOutputDir, expectedFilename)
	expectedContent, err := os.ReadFile(expectedIndexPath)
	if err != nil {
		t.Fatalf("Failed to read expected %s: %v", expectedFilename, err)
	}

	actualStr := strings.TrimSpace(string(actualContent))
	expectedStr := strings.TrimSpace(string(expectedContent))

	if actualStr != expectedStr {
		t.Errorf("%s content mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedFilename, expectedStr, actualStr)
	}
}

func TestConvertLogseqToHugo_DeepNesting(t *testing.T) {
	// Setup: paths to test files
	inputPath := "test-nesting.md"
	expectedOutputDir := "2025-01-20_Deep_Nesting_Test"
	expectedFilename := "index.de.md" // Default to German when no language specified

	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Run the conversion using the real convertFile function
	outputs, err := convertFile(inputPath, tempDir)
	if err != nil {
		t.Fatalf("convertFile() error = %v", err)
	}

	if len(outputs) == 0 {
		t.Fatalf("convertFile() returned no outputs")
	}

	output := outputs[0]

	// Verify the output directory was created with the expected name
	expectedDirName := filepath.Base(expectedOutputDir)
	actualDirName := filepath.Base(output.Dir)
	if actualDirName != expectedDirName {
		t.Errorf("Output directory name = %v, want %v", actualDirName, expectedDirName)
	}

	// Verify the correct filename was created
	if output.Filename != expectedFilename {
		t.Errorf("Output filename = %v, want %v", output.Filename, expectedFilename)
	}

	// Verify the index file exists
	indexPath := filepath.Join(output.Dir, expectedFilename)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatalf("%s does not exist at %s", expectedFilename, indexPath)
	}

	// Compare content with expected output
	actualContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read generated %s: %v", expectedFilename, err)
	}

	expectedIndexPath := filepath.Join(expectedOutputDir, expectedFilename)
	expectedContent, err := os.ReadFile(expectedIndexPath)
	if err != nil {
		t.Fatalf("Failed to read expected %s: %v", expectedFilename, err)
	}

	actualStr := strings.TrimSpace(string(actualContent))
	expectedStr := strings.TrimSpace(string(expectedContent))

	if actualStr != expectedStr {
		t.Errorf("%s content mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedFilename, expectedStr, actualStr)
	}
}

func TestConvertLogseqToHugo_MultiplePosts(t *testing.T) {
	// Setup: paths to test files
	inputPath := "test-multiple.md"
	expectedOutputDirs := []string{
		"2025-01-21_First_Post",
		"2025-01-22_Second_Post",
	}
	expectedFilename := "index.de.md" // Default to German when no language specified

	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Run the conversion - convertFile handles multiple posts
	outputs, err := convertFile(inputPath, tempDir)
	if err != nil {
		t.Fatalf("convertFile() error = %v", err)
	}

	// Verify we got exactly 2 outputs
	if len(outputs) != 2 {
		t.Fatalf("Expected 2 outputs, got %d", len(outputs))
	}

	// Test each blog post
	for i, expectedOutputDir := range expectedOutputDirs {
		// Verify the output directory was created with the expected name
		expectedDirName := filepath.Base(expectedOutputDir)
		actualDirName := filepath.Base(outputs[i].Dir)
		if actualDirName != expectedDirName {
			t.Errorf("Output directory %d name = %v, want %v", i+1, actualDirName, expectedDirName)
		}

		// Verify the correct filename was created
		if outputs[i].Filename != expectedFilename {
			t.Errorf("Output %d filename = %v, want %v", i+1, outputs[i].Filename, expectedFilename)
		}

		// Verify the index file exists
		indexPath := filepath.Join(outputs[i].Dir, expectedFilename)
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Fatalf("%s does not exist at %s", expectedFilename, indexPath)
		}

		// Compare content with expected output
		actualContent, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatalf("Failed to read generated %s for post %d: %v", expectedFilename, i+1, err)
		}

		expectedIndexPath := filepath.Join(expectedOutputDir, expectedFilename)
		expectedContent, err := os.ReadFile(expectedIndexPath)
		if err != nil {
			t.Fatalf("Failed to read expected %s for post %d: %v", expectedFilename, i+1, err)
		}

		actualStr := strings.TrimSpace(string(actualContent))
		expectedStr := strings.TrimSpace(string(expectedContent))

		if actualStr != expectedStr {
			t.Errorf("Post %d %s content mismatch.\nExpected:\n%s\n\nActual:\n%s", i+1, expectedFilename, expectedStr, actualStr)
		}
	}
}
