package fsutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// HomeDir returns the user's home directory.
func HomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return homeDir
}

// WorkspaceDir returns the govm workspace directory (~/.govm).
func WorkspaceDir() string {
	return filepath.Join(HomeDir(), ".govm")
}

// CopyDir recursively copies a directory to destination.
func CopyDir(src, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
			info, err := os.Stat(srcPath)
			if err != nil {
				return err
			}
			if err := os.Chmod(dstPath, info.Mode()); err != nil {
				return err
			}
		} else {
			file, err := os.Open(srcPath)
			if err != nil {
				return err
			}

			outFile, err := os.Create(dstPath)
			if err != nil {
				file.Close()
				return err
			}

			if _, err := io.Copy(outFile, file); err != nil {
				outFile.Close()
				file.Close()
				return err
			}
			file.Close()
			outFile.Close()

			info, err := os.Stat(srcPath)
			if err != nil {
				return err
			}
			if err := os.Chmod(dstPath, info.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveMatchingFiles deletes files matching the specified pattern.
func RemoveMatchingFiles(pattern string) error {
	dir, filePattern := filepath.Split(pattern)

	if dir == "" {
		dir = "."
	} else {
		dir = strings.TrimSuffix(dir, string(filepath.Separator))
	}

	if filePattern == "" {
		return nil
	}

	re, err := regexp.Compile(filePattern)
	if err != nil {
		return fmt.Errorf("cannot compile pattern %s: %s", pattern, err)
	}

	var filesRemoved []string

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}

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

	for _, file := range filesRemoved {
		fmt.Printf("  - %s\n", file)
	}

	return nil
}
