package urlparse

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	arxivNewIDPattern = regexp.MustCompile(`^\d{4}\.\d{4,5}(v\d+)?$`)
	arxivOldIDPattern = regexp.MustCompile(`^[A-Za-z-]+(\.[A-Za-z]{2})?/\d{7}(v\d+)?$`)
)

// NormalizeArxivInput recognizes an arXiv ID or arXiv URL and returns a
// canonical arXiv abs URL plus the extracted ID.
func NormalizeArxivInput(input string) (canonicalURL, id string, ok bool) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", "", false
	}

	if len(s) >= len("arxiv:") && strings.EqualFold(s[:len("arxiv:")], "arxiv:") {
		s = strings.TrimSpace(s[len("arxiv:"):])
	}

	if IsURL(s) {
		id, ok := extractArxivIDFromURL(s)
		if !ok {
			return "", "", false
		}
		return "https://arxiv.org/abs/" + id, id, true
	}

	s = strings.TrimSuffix(s, ".pdf")
	if isArxivID(s) {
		return "https://arxiv.org/abs/" + s, s, true
	}

	return "", "", false
}

// IsArxivInput reports whether input is an arXiv ID or arXiv URL.
func IsArxivInput(input string) bool {
	_, _, ok := NormalizeArxivInput(input)
	return ok
}

func isArxivID(s string) bool {
	return arxivNewIDPattern.MatchString(s) || arxivOldIDPattern.MatchString(s)
}

func extractArxivIDFromURL(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "http://arxiv.org/") &&
		!strings.HasPrefix(lower, "https://arxiv.org/") &&
		!strings.HasPrefix(lower, "http://www.arxiv.org/") &&
		!strings.HasPrefix(lower, "https://www.arxiv.org/") &&
		!strings.HasPrefix(lower, "http://export.arxiv.org/") &&
		!strings.HasPrefix(lower, "https://export.arxiv.org/") {
		return "", false
	}

	// Avoid pulling in net/url just for this simple path extraction; arXiv IDs
	// cannot contain '?' or '#'.
	path := trimmed
	if idx := strings.Index(path, "://"); idx >= 0 {
		path = path[idx+3:]
		if slash := strings.Index(path, "/"); slash >= 0 {
			path = path[slash+1:]
		} else {
			return "", false
		}
	}
	if q := strings.IndexAny(path, "?#"); q >= 0 {
		path = path[:q]
	}
	path = strings.Trim(path, "/")

	for _, prefix := range []string{"abs/", "pdf/", "html/", "e-print/"} {
		if strings.HasPrefix(path, prefix) {
			id := strings.TrimPrefix(path, prefix)
			id = strings.TrimSuffix(id, ".pdf")
			if isArxivID(id) {
				return id, true
			}
		}
	}

	return "", false
}

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
