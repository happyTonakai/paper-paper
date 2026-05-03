package urlparse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://arxiv.org/abs/2301.00001", true},
		{"http://example.com", true},
		{"/path/to/file", false},
		{"./relative", false},
		{"~/home/file", false},
		{"just text", false},
	}

	for _, tt := range tests {
		result := IsURL(tt.input)
		if result != tt.expected {
			t.Errorf("IsURL(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestIsFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/absolute/path", true},
		{"./relative", true},
		{"../parent", true},
		{"~/home/file", true},
		{"https://example.com", false},
		{"just text", false},
	}

	for _, tt := range tests {
		result := IsFilePath(tt.input)
		if result != tt.expected {
			t.Errorf("IsFilePath(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestLoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "This is test paper content"
	os.WriteFile(testFile, []byte(content), 0644)

	result, err := LoadFile(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestLoadFileWithTilde(t *testing.T) {
	// Create a file in temp dir
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "tilde test content"
	os.WriteFile(testFile, []byte(content), 0644)

	// Set HOME to tmpDir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	result, err := LoadFile("~/test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestHTTPFetchInvalidURL(t *testing.T) {
	_, err := httpFetch("http://localhost:1")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
