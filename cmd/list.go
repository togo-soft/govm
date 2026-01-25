package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gitter.top/apps/govm/request"
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
}

func (gd *GoVMData) IsInstalled(version string) bool {
	for _, iv := range gd.InstalledVersions {
		if iv == version {
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

func (m *Command) listCommand() {
	var stable bool
	var vm = NewVersionManager()
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list golang versions",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   true,
			DisableNoDescFlag:   true,
			DisableDescriptions: true,
			HiddenDefaultCmd:    true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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
			// sync go versions
			if err := vm.SyncGoVersions(); err != nil {
				log.Error("could not sync go versions", "reason", err)
				return err
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			versions, err := vm.ReadLocalGoVersions()
			if err != nil {
				slog.Error("could not read local versions", "reason", err)
				return err
			}
			vm.Versions = versions
			return vm.ReadLocalData()
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
			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"Version", "Installed"})
			var showData [][]string
			for _, normalized := range versionList {
				// Get original version for display
				displayVersion := normalizedVersions[normalized]
				var installed = ""
				if vm.LocalData.IsInstalled(displayVersion) {
					installed = "true"
				}
				showData = append(showData, []string{
					displayVersion,
					installed,
				})
			}
			if err := table.Bulk(showData); err != nil {
				return err
			}
			return table.Render()
		},
	}

	cmd.Flags().BoolVarP(&stable, "stable", "s", false, "show stable versions")

	m.cmd.AddCommand(cmd)
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
		return err
	}
	for _, entry := range dirEntries {
		if entry.IsDir() {
			vm.LocalData.InstalledVersions = append(vm.LocalData.InstalledVersions, entry.Name())
		}
	}
	return nil
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
