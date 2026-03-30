package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var githubBlobRe = regexp.MustCompile(`^https://github\.com/([^/]+/[^/]+)/blob/(.+)$`)

// isURL returns true if s is an HTTPS URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "https://")
}

// toRawURL converts a GitHub blob URL to its raw.githubusercontent.com equivalent.
// raw.githubusercontent.com and other HTTPS URLs are returned unchanged.
// Non-HTTPS URLs return an error.
func toRawURL(u string) (string, error) {
	if !strings.HasPrefix(u, "https://") {
		return "", fmt.Errorf("only HTTPS URLs are supported")
	}
	if m := githubBlobRe.FindStringSubmatch(u); m != nil {
		return "https://raw.githubusercontent.com/" + m[1] + "/" + m[2], nil
	}
	return u, nil
}

// downloadToCache downloads the JS file at rawURL into cacheDir and returns
// the local file path. Re-downloading the same URL overwrites the cached copy.
// The cache filename is derived from the last path segment of the URL.
func downloadToCache(rawURL, cacheDir string) (string, error) {
	downloadURL, err := toRawURL(rawURL)
	if err != nil {
		return "", err
	}

	// Derive filename from the last segment of the URL path.
	urlPath := strings.Split(downloadURL, "?")[0]
	urlPath = strings.TrimRight(urlPath, "/")
	parts := strings.Split(urlPath, "/")
	filename := sanitizeFilename(parts[len(parts)-1])
	if filename == "" {
		return "", fmt.Errorf("could not derive a filename from URL")
	}
	if !strings.HasSuffix(filename, ".js") {
		filename += ".js"
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %s", resp.Status)
	}

	const maxSize = 1 << 20 // 1 MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if len(body) > maxSize {
		return "", fmt.Errorf("file too large (max 1 MB)")
	}

	localPath := filepath.Join(cacheDir, filename)
	if err := os.WriteFile(localPath, body, 0o644); err != nil {
		return "", fmt.Errorf("write cache: %w", err)
	}

	return localPath, nil
}

// sanitizeFilename keeps only characters safe for a cross-platform filename.
func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
