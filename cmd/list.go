package main

import (
	"archive/tar"
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"codefloe.com/apps/govm/request"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

type Versions []*Version

type Version struct {
	Version string         `json:"version"`
	Stable  bool           `json:"stable"`
	Files   []*VersionFile `json:"files"`
}

type VersionFile struct {
	Filename string `json:"filename"`
	Os       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Sha256   string `json:"sha256"`
	Size     int    `json:"size"`
	Kind     string `json:"kind"`
}

type ShowVersion struct {
	Version   string `json:"version"`
	Stable    bool   `json:"stable"`
	SHA256    string `json:"sha256"`
	Size      int    `json:"size"`
	Installed bool   `json:"installed"`
}

type GoVMData struct {
	// last check remote time
	LastCheckedAt time.Time `json:"last_checked_at"`
	// local installed go version list
	InstalledVersions []string `json:"installed_versions"`
	// current using go version
	CurrentVersion string `json:"current_version"`
}

func (gd *GoVMData) IsInstalled(version string) bool {
	for _, iv := range gd.InstalledVersions {
		if iv == version {
			return true
		}
	}
	return false
}

// IsVersionDownloaded checks if a version file exists in downloads directory
func (gd *GoVMData) IsVersionDownloaded(version string, filename string) bool {
	// Check if file exists in downloads
	downloadsDir := filepath.Join(WorkspaceDir(), "downloads")
	filePath := filepath.Join(downloadsDir, filename)
	_, err := os.Stat(filePath)
	return err == nil
}

func (vm *VersionManager) IsValidVersion(version string) bool {
	versions := vm.filterGoVersions(false)
	for _, ver := range versions {
		if ver == version {
			return true
		}
	}
	return false
}

type VersionManager struct {
	Versions  Versions  `json:"versions"`
	LocalData *GoVMData `json:"local_data"`
	LocalOS   string    `json:"local_os"`
	LocalArch string    `json:"local_arch"`
}

func NewVersionManager() *VersionManager {
	return &VersionManager{
		LocalOS:   runtime.GOOS,
		LocalArch: runtime.GOARCH,
	}
}

func (vm *VersionManager) Initialized() error {
	// create workspace directory
	if err := os.MkdirAll(WorkspaceDir(), 0740); err != nil {
		log.Error("could not create workspace directory", "reason", err)
		return err
	}
	// read or create local data file
	if err := vm.ReadLocalData(); err != nil {
		log.Error("could not read local data", "reason", err)
		return err
	}
	// create versions directory
	if err := os.MkdirAll(filepath.Join(WorkspaceDir(), "versions"), 0740); err != nil {
		log.Error("could not create versions directory", "reason", err)
		return err
	}
	// create downloads directory
	if err := os.MkdirAll(filepath.Join(WorkspaceDir(), "downloads"), 0740); err != nil {
		log.Error("could not create downloads directory", "reason", err)
		return err
	}
	// create current directory
	if err := os.MkdirAll(filepath.Join(WorkspaceDir(), "current"), 0740); err != nil {
		log.Error("could not create current directory", "reason", err)
		return err
	}
	// sync go versions
	if err := vm.SyncGoVersions(); err != nil {
		log.Error("could not sync go versions", "reason", err)
		return err
	}

	versions, err := vm.ReadLocalGoVersions()
	if err != nil {
		slog.Error("could not read local versions", "reason", err)
		return err
	}
	vm.Versions = versions
	return vm.ReadLocalData()
}

func (vm *VersionManager) SyncGoVersions() error {
	// check if you need refresh remote go versions
	if time.Now().Sub(vm.LocalData.LastCheckedAt) < time.Hour {
		return nil
	}
	// sync remote go versions
	versions, err := vm.ReadRemoteGoVersions()
	if err != nil {
		return err
	}
	vm.Versions = versions
	vm.LocalData.LastCheckedAt = time.Now()

	if err := vm.walkInstalledGoVersions(); err != nil {
		return err
	}
	return vm.writeFile()
}

func (vm *VersionManager) walkInstalledGoVersions() error {
	dir := filepath.Join(WorkspaceDir(), "versions")
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		// versions directory might not exist yet, that's ok
		return nil
	}

	// Clear the list first
	vm.LocalData.InstalledVersions = nil

	// Scan version directories
	for _, entry := range dirEntries {
		if entry.IsDir() {
			vm.LocalData.InstalledVersions = append(vm.LocalData.InstalledVersions, entry.Name())
		}
	}
	return nil
}

// extractVersionFromFilename extracts version number from Go distribution filename
// Examples: "go1.25.6.tar.gz" -> "1.25.6"
//
//	"go1.25.6.windows-amd64.zip" -> "1.25.6"
//	"go1.25.6.linux-amd64.tar.gz" -> "1.25.6"
//	"go1.21rc1.tar.gz" -> "1.21rc1"
func extractVersionFromFilename(filename string) string {
	// Remove extension
	name := filename
	if strings.HasSuffix(name, ".tar.gz") {
		name = strings.TrimSuffix(name, ".tar.gz")
	} else if strings.HasSuffix(name, ".zip") {
		name = strings.TrimSuffix(name, ".zip")
	} else {
		return ""
	}

	// Remove "go" prefix
	if strings.HasPrefix(name, "go") {
		name = name[2:]
	}

	// Extract version number using regex
	// Matches: major.minor[.patch][prerelease]
	// Examples: 1.25.6, 1.25, 1.25.6rc1, 1.21beta1, 1.10alpha1
	pattern := regexp.MustCompile(`^(\d+\.\d+(?:\.\d+)?(?:[a-z]+\d+)?)`)
	matches := pattern.FindStringSubmatch(name)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func (vm *VersionManager) writeFile() error {
	if err := vm.writeLocalData(); err != nil {
		return err
	}
	if err := vm.writeLocalGoVersions(); err != nil {
		return err
	}
	return nil
}

func (vm *VersionManager) ReadLocalData() error {
	filename := filepath.Join(WorkspaceDir(), "local.json")
	content, err := os.ReadFile(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content = []byte("{}")
	}
	var localData = new(GoVMData)
	err = json.Unmarshal(content, localData)
	if err != nil {
		return err
	}
	vm.LocalData = localData
	return nil
}

func (vm *VersionManager) writeLocalData() error {
	filename := filepath.Join(WorkspaceDir(), "local.json")
	content, err := json.Marshal(vm.LocalData)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, content, 0600)
}

func (vm *VersionManager) ReadLocalGoVersions() (Versions, error) {
	filename := filepath.Join(WorkspaceDir(), "versions.json")
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var versions Versions
	err = json.Unmarshal(content, &versions)
	if err != nil {
		return nil, err
	}
	return versions, nil
}

func (vm *VersionManager) writeLocalGoVersions() error {
	filename := filepath.Join(WorkspaceDir(), "versions.json")
	content, err := json.Marshal(vm.Versions)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, content, 0600)
}

func (vm *VersionManager) ReadRemoteGoVersions() (Versions, error) {
	var versions Versions
	requester := request.NewRequester()
	err := requester.Request(&request.Data{
		Method: http.MethodGet,
		URL:    "https://golang.google.cn/dl/?mode=json&include=all",
		Header: map[string]string{
			"User-Agent": "GoClient-govm",
		},
		ExpectedCode: http.StatusOK,
		Bind:         &versions,
	})
	return versions, err
}

func (vm *VersionManager) filterGoVersions(stable bool) []string {
	var vs []string
	for _, version := range vm.Versions {
		for _, fileData := range version.Files {
			if stable && !version.Stable {
				continue
			}
			if fileData.Os != vm.LocalOS {
				continue
			}
			if fileData.Arch != vm.LocalArch {
				continue
			}
			if fileData.Kind != "archive" {
				continue
			}
			x := version.Version
			if strings.HasPrefix(x, "go") {
				x = x[2:]
			}
			vs = append(vs, x)
		}
	}
	return vs
}

// normalizeVersion converts Go version format to semantic version format
// e.g. "1.10beta1" -> "1.10.0-beta1", "1.21rc1" -> "1.21.0-rc1", "1.21" -> "1.21.0"
func normalizeVersion(version string) string {
	// Pattern to match Go version format: X.Y or X.Y.Z followed by optional prerelease
	// e.g. "1.21", "1.21.0", "1.10beta1", "1.21rc1", "1.21beta2"
	pattern := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:(a|alpha|b|beta|rc)(\d+))?(.*)$`)
	matches := pattern.FindStringSubmatch(version)

	if matches == nil {
		// If it doesn't match the pattern, return as-is with v prefix
		return "v" + version
	}

	major := matches[1]
	minor := matches[2]
	patch := matches[3]
	preReleaseType := matches[4]
	preReleaseNum := matches[5]
	suffix := matches[6]

	// Set patch to 0 if not provided
	if patch == "" {
		patch = "0"
	}

	result := major + "." + minor + "." + patch

	// Normalize prerelease identifiers
	if preReleaseType != "" {
		switch preReleaseType {
		case "a", "alpha":
			preReleaseType = "alpha"
		case "b", "beta":
			preReleaseType = "beta"
		}
		result += "-" + preReleaseType + preReleaseNum
	}

	result += suffix
	return "v" + result
}

func (m *Command) listCommand() {
	var stable bool
	var vm = NewVersionManager()
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "list golang versions",
		Example: `govm list --stable`,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   true,
			DisableNoDescFlag:   true,
			DisableDescriptions: true,
			HiddenDefaultCmd:    true,
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return vm.Initialized()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			versions := vm.filterGoVersions(stable)
			// Normalize versions to semantic version format for proper sorting
			normalizedVersions := make(map[string]string) // maps normalized to original
			versionList := make([]string, 0, len(versions))

			for _, v := range versions {
				normalized := normalizeVersion(v)
				normalizedVersions[normalized] = v
				versionList = append(versionList, normalized)
			}

			semver.Sort(versionList)
			for _, normalized := range versionList {
				// Get original version for display
				displayVersion := normalizedVersions[normalized]
				if vm.LocalData.IsInstalled(displayVersion) {
					color.Green(displayVersion)
				} else {
					color.White(displayVersion)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&stable, "stable", "s", false, "show stable versions")

	m.cmd.AddCommand(cmd)
}

// findVersionFile finds the appropriate VersionFile for current platform
func (vm *VersionManager) findVersionFile(version string) *VersionFile {
	for _, v := range vm.Versions {
		// Handle both "go1.24.11" and "1.24.11" formats
		ver := v.Version
		if strings.HasPrefix(ver, "go") {
			ver = ver[2:]
		}
		if ver != version {
			continue
		}
		for _, file := range v.Files {
			if file.Os == vm.LocalOS && file.Arch == vm.LocalArch && file.Kind == "archive" {
				return file
			}
		}
	}
	return nil
}

// downloadFile downloads a file from URL and returns path to downloaded file
// progressReader wraps an io.Reader and displays download progress
type progressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	filename string
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.current += int64(n)

	// Calculate progress
	percent := float64(pr.current) / float64(pr.total) * 100
	barWidth := 30
	filledWidth := int(percent / 100 * float64(barWidth))
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	// Create progress bar
	bar := strings.Repeat("=", filledWidth) + strings.Repeat(" ", barWidth-filledWidth)

	// Format file size
	totalMB := float64(pr.total) / (1024 * 1024)
	currentMB := float64(pr.current) / (1024 * 1024)

	// Print progress (using \r to overwrite the same line)
	fmt.Printf("\r[%s] %.1f MB / %.1f MB (%.1f%%)",
		bar, currentMB, totalMB, percent)

	if err == io.EOF {
		fmt.Println() // New line when done
	}

	return
}

func downloadFile(url string, destDir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status code: %d", resp.StatusCode)
	}

	// Extract filename from URL
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	filePath := filepath.Join(destDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Show download info
	log.Info("downloading", "file", filename)

	// Create progress reader
	pr := &progressReader{
		reader:   resp.Body,
		total:    resp.ContentLength,
		filename: filename,
	}

	_, err = io.Copy(file, pr)
	if err != nil {
		os.Remove(filePath)
		return "", err
	}

	log.Info("download completed", "file", filename)
	return filePath, nil
}

// verifySha256 verifies file SHA256 checksum and displays result
func verifySha256(filePath string, expectedSum string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actual := hex.EncodeToString(hash.Sum(nil))

	// Extract filename from path for display
	filename := filepath.Base(filePath)

	// Display verification result
	if actual == expectedSum {
		fmt.Printf("✓ SHA256 verification passed: %s\n", filename)
		log.Info("sha256 verification passed", "file", filename)
		return nil
	}

	// Verification failed - show details
	fmt.Printf("✗ SHA256 verification FAILED: %s\n", filename)
	fmt.Printf("  Expected: %s\n", expectedSum)
	fmt.Printf("  Got:      %s\n", actual)
	log.Error("sha256 verification failed", "file", filename, "expected", expectedSum, "actual", actual)

	return fmt.Errorf("sha256 mismatch: expected %s, got %s", expectedSum, actual)
}

// extractArchive extracts tar.gz or zip archive to destination
func extractArchive(src string, dest string) error {
	if strings.HasSuffix(src, ".tar.gz") {
		return extractTarGz(src, dest)
	} else if strings.HasSuffix(src, ".zip") {
		return extractZip(src, dest)
	}
	return fmt.Errorf("unsupported archive format: %s", src)
}

// copyDir recursively copies a directory to destination
func copyDir(src string, dest string) error {
	// Remove destination if it exists
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
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			file, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer file.Close()

			outFile, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, file); err != nil {
				return err
			}
		}
	}
	return nil
}

// extractTarGz extracts tar.gz archive
func extractTarGz(src string, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	tr := tar.NewReader(file)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Remove "go/" prefix if present (Go official distribution structure)
		name := header.Name
		if strings.HasPrefix(name, "go/") {
			name = strings.TrimPrefix(name, "go/")
		} else if name == "go" {
			// Skip the top-level go directory itself
			continue
		}

		// Skip empty paths
		if name == "" {
			continue
		}

		path := filepath.Join(dest, name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}
	return nil
}

// extractZip extracts zip archive
func extractZip(src string, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		// Remove "go/" prefix if present (Go official distribution structure)
		name := file.Name
		if strings.HasPrefix(name, "go/") {
			name = strings.TrimPrefix(name, "go/")
		} else if name == "go" {
			// Skip the top-level go directory itself
			continue
		}

		// Skip empty paths
		if name == "" {
			continue
		}

		path := filepath.Join(dest, name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(f, rc)
		f.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// InstallVersion downloads and installs a Go version
func (vm *VersionManager) InstallVersion(version string, siteURL string) error {
	// Find version file metadata
	versionFile := vm.findVersionFile(version)
	if versionFile == nil {
		return fmt.Errorf("version file not found for %s", version)
	}

	// Check if file exists in downloads directory
	downloadsDir := filepath.Join(WorkspaceDir(), "downloads")
	downloadedFile := filepath.Join(downloadsDir, versionFile.Filename)

	if _, err := os.Stat(downloadedFile); err != nil {
		// File doesn't exist, download it
		if err := os.MkdirAll(downloadsDir, 0755); err != nil {
			return fmt.Errorf("failed to create downloads directory: %w", err)
		}

		// Construct download URL
		downloadURL := strings.TrimRight(siteURL, "/") + "/" + versionFile.Filename

		file, err := downloadFile(downloadURL, downloadsDir)
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", downloadURL, err)
		}
		downloadedFile = file

		// Verify SHA256 only for newly downloaded files
		if err := verifySha256(downloadedFile, versionFile.Sha256); err != nil {
			os.Remove(downloadedFile)
			return fmt.Errorf("failed to verify checksum: %w", err)
		}
		log.Info("file downloaded", "file", versionFile.Filename)
	} else {
		log.Info("file found in downloads, skipping download", "file", versionFile.Filename)
	}

	// Extract to versions/{version} directory
	versionsDir := filepath.Join(WorkspaceDir(), "versions")
	versionDir := filepath.Join(versionsDir, version)

	// Remove existing version directory if it exists
	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove existing version directory: %w", err)
	}
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Extract archive to versions/{version}
	if err := extractArchive(downloadedFile, versionDir); err != nil {
		os.RemoveAll(versionDir)
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Copy to current directory
	currentDir := filepath.Join(WorkspaceDir(), "current")
	if err := copyDir(versionDir, currentDir); err != nil {
		return fmt.Errorf("failed to copy to current directory: %w", err)
	}

	// Update installed versions list (scan versions directory)
	if err := vm.walkInstalledGoVersions(); err != nil {
		return fmt.Errorf("failed to update installed versions: %w", err)
	}

	// Update current version
	vm.LocalData.CurrentVersion = version

	// Save local data
	if err := vm.writeLocalData(); err != nil {
		return fmt.Errorf("failed to save local data: %w", err)
	}

	log.Info("version installed and set as current", "version", version)
	return nil
}
