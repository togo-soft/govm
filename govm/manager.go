package govm

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"codeberg.org/application/govm/internal/archive"
	"codeberg.org/application/govm/internal/download"
	"codeberg.org/application/govm/internal/fsutil"
)

type Manager struct {
	Versions  Versions
	Data      *LocalData
	OS        string
	Arch      string
	workspace string
}

func NewManager() *Manager {
	return &Manager{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		workspace: fsutil.WorkspaceDir(),
	}
}

func (m *Manager) Init() error {
	if err := os.MkdirAll(m.workspace, 0740); err != nil {
		slog.Error("could not create workspace directory", "reason", err)
		return err
	}

	if err := m.readLocalData(); err != nil {
		slog.Error("could not read local data", "reason", err)
		return err
	}

	for _, sub := range []string{"versions", "downloads", "go"} {
		if err := os.MkdirAll(filepath.Join(m.workspace, sub), 0740); err != nil {
			slog.Error("could not create directory", "dir", sub, "reason", err)
			return err
		}
	}

	if err := m.Sync(); err != nil {
		slog.Error("could not sync go versions", "reason", err)
		return err
	}

	versions, err := m.readLocalVersions()
	if err != nil {
		slog.Error("could not read local versions", "reason", err)
		return err
	}
	m.Versions = versions
	return m.readLocalData()
}

func (m *Manager) Sync() error {
	if time.Since(m.Data.LastCheckedAt) < time.Hour {
		return nil
	}

	versions, err := m.fetchRemoteVersions()
	if err != nil {
		return err
	}
	m.Versions = versions
	m.Data.LastCheckedAt = time.Now()

	if err := m.walkInstalledVersions(); err != nil {
		return err
	}
	return m.saveAll()
}

func (m *Manager) fetchRemoteVersions() (Versions, error) {
	req, err := http.NewRequest(http.MethodGet, "https://golang.google.cn/dl/?mode=json&include=all", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GoClient-govm")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var versions Versions
	if err := json.Unmarshal(body, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (m *Manager) FilterVersions(stable bool) []string {
	var vs []string
	for _, version := range m.Versions {
		for _, fileData := range version.Files {
			if stable && !version.Stable {
				continue
			}
			if fileData.Os != m.OS {
				continue
			}
			if fileData.Arch != m.Arch {
				continue
			}
			if fileData.Kind != "archive" {
				continue
			}
			vs = append(vs, strings.TrimPrefix(version.Version, "go"))
		}
	}
	return vs
}

func (m *Manager) IsValidVersion(version string) bool {
	return slices.Contains(m.FilterVersions(false), version)
}

func (m *Manager) FindVersionFile(version string) *VersionFile {
	for _, v := range m.Versions {
		ver := strings.TrimPrefix(v.Version, "go")
		if ver != version {
			continue
		}
		for _, file := range v.Files {
			if file.Os == m.OS && file.Arch == m.Arch && file.Kind == "archive" {
				return file
			}
		}
	}
	return nil
}

func (m *Manager) Install(version, siteURL string) error {
	versionFile := m.FindVersionFile(version)
	if versionFile == nil {
		return fmt.Errorf("version file not found for %s", version)
	}

	downloadsDir := filepath.Join(m.workspace, "downloads")
	downloadedFile := filepath.Join(downloadsDir, versionFile.Filename)

	if _, err := os.Stat(downloadedFile); err != nil {
		if err := os.MkdirAll(downloadsDir, 0755); err != nil {
			return fmt.Errorf("failed to create downloads directory: %w", err)
		}

		downloadURL := strings.TrimRight(siteURL, "/") + "/" + versionFile.Filename

		file, err := download.File(downloadURL, downloadsDir)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", downloadURL, err)
		}
		downloadedFile = file

		if err := download.VerifySHA256(downloadedFile, versionFile.Sha256); err != nil {
			os.Remove(downloadedFile)
			return fmt.Errorf("failed to verify checksum: %w", err)
		}
		slog.Info("file downloaded", "file", versionFile.Filename)
	} else {
		slog.Info("file found in downloads, skipping download", "file", versionFile.Filename)
	}

	versionsDir := filepath.Join(m.workspace, "versions")
	versionDir := filepath.Join(versionsDir, version)

	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove existing version directory: %w", err)
	}
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	if err := archive.Extract(downloadedFile, versionDir); err != nil {
		os.RemoveAll(versionDir)
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	currentDir := filepath.Join(m.workspace, "go")
	if err := fsutil.CopyDir(versionDir, currentDir); err != nil {
		return fmt.Errorf("failed to copy to .govm/go directory: %w", err)
	}

	if err := m.walkInstalledVersions(); err != nil {
		return fmt.Errorf("failed to update installed versions: %w", err)
	}

	m.Data.CurrentVersion = version

	if err := m.writeLocalData(); err != nil {
		return fmt.Errorf("failed to save local data: %w", err)
	}

	slog.Info("version installed and set as current", "version", version)
	return nil
}

func (m *Manager) Uninstall(version string) error {
	wasInstalled := m.Data.IsInstalled(version)

	// Remove version directory
	versionsDir := filepath.Join(m.workspace, "versions")
	versionDir := filepath.Join(versionsDir, version)
	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}
	slog.Info("removed version directory", "version", version)

	// Remove corresponding download files
	downloadsDir := filepath.Join(m.workspace, "downloads")
	dirEntries, err := os.ReadDir(downloadsDir)
	if err == nil {
		for _, entry := range dirEntries {
			if !entry.IsDir() {
				filename := entry.Name()
				fileVersion := ExtractVersionFromFilename(filename)
				if fileVersion == version {
					filePath := filepath.Join(downloadsDir, filename)
					if err := os.Remove(filePath); err != nil {
						slog.Error("failed to remove file from downloads", "file", filename, "reason", err)
					} else {
						slog.Info("removed file from downloads", "file", filename)
					}
					break
				}
			}
		}
	}

	// If removing current version, clear it
	if m.Data.CurrentVersion == version {
		m.Data.CurrentVersion = ""
		currentDir := filepath.Join(m.workspace, "go")
		os.RemoveAll(currentDir)
		os.MkdirAll(currentDir, 0755)
	}

	if err := m.walkInstalledVersions(); err != nil {
		return fmt.Errorf("failed to update installed versions: %w", err)
	}

	if err := m.writeLocalData(); err != nil {
		return fmt.Errorf("failed to save local data: %w", err)
	}

	if !wasInstalled {
		return fmt.Errorf("version %s not installed", version)
	}

	slog.Info("version removed", "version", version)
	return nil
}
