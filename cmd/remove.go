package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func (m *Command) removeCommand() {
	var version string
	var vm = NewVersionManager()
	cmd := &cobra.Command{
		Use:     "remove [version]",
		Short:   "remove golang version",
		Example: "govm remove 1.24.11\ngovm remove -v 1.24.11",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return vm.Initialized()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// If version not specified via flag, try to get it from positional argument
			if version == "" && len(args) > 0 {
				version = args[0]
			}
			if vm.LocalData.IsInstalled(version) {
				return nil
			}
			return errors.New("version not installed")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return vm.UninstallVersion(version)
		},
	}

	cmd.Flags().StringVarP(&version, "version", "v", "", "golang version to remove")

	m.cmd.AddCommand(cmd)
}

// UninstallVersion removes an installed Go version
func (vm *VersionManager) UninstallVersion(version string) error {
	// Remove version directory from versions
	versionsDir := filepath.Join(WorkspaceDir(), "versions")
	versionDir := filepath.Join(versionsDir, version)

	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}
	log.Info("removed version directory", "version", version)

	// Try to remove corresponding file from downloads if it exists
	downloadsDir := filepath.Join(WorkspaceDir(), "downloads")
	dirEntries, err := os.ReadDir(downloadsDir)
	if err == nil {
		for _, entry := range dirEntries {
			if !entry.IsDir() {
				filename := entry.Name()
				// Extract version from filename
				fileVersion := extractVersionFromFilename(filename)
				if fileVersion == version {
					filePath := filepath.Join(downloadsDir, filename)
					if err := os.Remove(filePath); err != nil {
						log.Error("failed to remove file from downloads", "file", filename, "reason", err)
					} else {
						log.Info("removed file from downloads", "file", filename)
					}
					break
				}
			}
		}
	}

	// If removing current version, clear CurrentVersion and current directory
	if vm.LocalData.CurrentVersion == version {
		vm.LocalData.CurrentVersion = ""
		// Clear current directory
		currentDir := filepath.Join(WorkspaceDir(), "current")
		os.RemoveAll(currentDir)
		os.MkdirAll(currentDir, 0755)
	}

	// Update installed versions list (scan versions directory)
	if err := vm.walkInstalledGoVersions(); err != nil {
		return fmt.Errorf("failed to update installed versions: %w", err)
	}

	// Save local data
	if err := vm.writeLocalData(); err != nil {
		return fmt.Errorf("failed to save local data: %w", err)
	}

	log.Info("version removed", "version", version)
	return nil
}
