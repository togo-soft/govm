package main

import (
	"os"
	"path/filepath"
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
