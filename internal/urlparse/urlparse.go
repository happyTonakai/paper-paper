package urlparse

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FetchURL fetches content from a URL, trying arxiv2text first, then falling back to HTTP.
func FetchURL(url string) (string, error) {
	// Try arxiv2text first
	if content, err := tryArxiv2Text(url); err == nil && content != "" {
		return content, nil
	}

	// Fallback to HTTP fetch
	return httpFetch(url)
}

func tryArxiv2Text(url string) (string, error) {
	// Check if arxiv2text is available
	path, err := exec.LookPath("arxiv2text")
	if err != nil {
		return "", fmt.Errorf("arxiv2text not found in PATH")
	}

	cmd := exec.Command(path, url)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("arxiv2text failed: %w", err)
	}

	return string(output), nil
}

func httpFetch(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(body), nil
}

// LoadFile loads content from a file path.
func LoadFile(path string) (string, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return string(data), nil
}

// IsURL checks if a string looks like a URL.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// IsFilePath checks if a string looks like a file path.
func IsFilePath(s string) bool {
	return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~")
}
