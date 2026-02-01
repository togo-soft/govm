package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func HomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return homeDir
}

func WorkspaceDir() string {
	homeDir := HomeDir()
	return filepath.Join(homeDir, ".govm")
}

// Delete files matching the specified pattern
func removeFiles(pattern string) error {
	// Split directory and filename pattern
	dir, filePattern := filepath.Split(pattern)

	// If directory is empty, set to current directory
	if dir == "" {
		dir = "."
	} else {
		// Remove trailing path separator
		dir = strings.TrimSuffix(dir, string(filepath.Separator))
	}

	// If file pattern is empty, cancel call
	if filePattern == "" {
		return nil
	}

	// Convert file pattern to regular expression
	re, err := regexp.Compile(filePattern)
	if err != nil {
		return fmt.Errorf("cannot compile pattern %s: %s", pattern, err)
	}

	var filesRemoved []string

	// Traverse directory
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Get relative path (relative to specified directory)
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}

		// Check if filename matches the pattern
		if re.MatchString(filepath.Base(relPath)) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete %s: %v", path, err)
			}

			filesRemoved = append(filesRemoved, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to traverse directory: %v", err)
	}

	// Output results
	for _, file := range filesRemoved {
		fmt.Printf("  - %s\n", file)
	}

	return nil
}
