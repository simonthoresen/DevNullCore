// gen-manifest walks a directory and produces a .bundle-manifest.json
// listing every bundled asset file with its SHA-256 checksum.
//
// Usage: go run ./cmd/gen-manifest <dir> > .bundle-manifest.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dev-null/internal/datadir"
)

// skipExts lists file extensions that are not bundled assets.
var skipExts = map[string]bool{
	".exe": true,
	".ps1": true,
	".log": true,
}

// skipNames lists file/directory names to skip entirely.
var skipNames = map[string]bool{
	".cache":                true,
	"state":                 true,
	".version":              true,
	".bundle-manifest.json": true,
	".bundle-version":       true,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gen-manifest <dir>\n")
		os.Exit(1)
	}
	root := os.Args[1]

	var files []datadir.ManifestFile
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if skipNames[name] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if skipExts[ext] {
			return nil
		}
		// Skip hidden files (dotfiles) other than known ones.
		if strings.HasPrefix(name, ".") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Normalize to forward slashes for cross-platform manifests.
		rel = filepath.ToSlash(rel)

		hash, err := datadir.FileHash(path)
		if err != nil {
			return fmt.Errorf("hash %s: %w", rel, err)
		}
		files = append(files, datadir.ManifestFile{
			Path:   rel,
			SHA256: hash,
		})
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	manifest := datadir.Manifest{
		Version: "generated",
		Files:   files,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
