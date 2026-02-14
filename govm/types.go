package govm

import (
	"regexp"
	"slices"
	"strings"
	"time"
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

type LocalData struct {
	LastCheckedAt     time.Time `json:"last_checked_at"`
	InstalledVersions []string  `json:"installed_versions"`
	CurrentVersion    string    `json:"current_version"`
}

func (d *LocalData) IsInstalled(version string) bool {
	return slices.Contains(d.InstalledVersions, version)
}

// ExtractVersionFromFilename extracts version number from Go distribution filename.
// Examples: "go1.25.6.tar.gz" -> "1.25.6", "go1.21rc1.tar.gz" -> "1.21rc1"
func ExtractVersionFromFilename(filename string) string {
	name := filename
	if strings.HasSuffix(name, ".tar.gz") {
		name = strings.TrimSuffix(name, ".tar.gz")
	} else if strings.HasSuffix(name, ".zip") {
		name = strings.TrimSuffix(name, ".zip")
	} else {
		return ""
	}

	name = strings.TrimPrefix(name, "go")

	pattern := regexp.MustCompile(`^(\d+\.\d+(?:\.\d+)?(?:[a-z]+\d+)?)`)
	matches := pattern.FindStringSubmatch(name)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// NormalizeVersion converts Go version format to semantic version format.
// e.g. "1.10beta1" -> "v1.10.0-beta1", "1.21rc1" -> "v1.21.0-rc1", "1.21" -> "v1.21.0"
func NormalizeVersion(version string) string {
	pattern := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:(a|alpha|b|beta|rc)(\d+))?(.*)$`)
	matches := pattern.FindStringSubmatch(version)

	if matches == nil {
		return "v" + version
	}

	major := matches[1]
	minor := matches[2]
	patch := matches[3]
	preReleaseType := matches[4]
	preReleaseNum := matches[5]
	suffix := matches[6]

	if patch == "" {
		patch = "0"
	}

	result := major + "." + minor + "." + patch

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
