package archive

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGzPreservesFilePermissions(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	archivePath := filepath.Join(base, "go-test.tar.gz")
	dest := filepath.Join(base, "out")

	testFiles := []struct {
		name string
		path string
		mode int64
		body string
	}{
		{
			name: "bin executable",
			path: filepath.Join("go", "bin", "go"),
			mode: 0o755,
			body: "go binary",
		},
		{
			name: "lib shell script",
			path: filepath.Join("go", "lib", "time", "update.bash"),
			mode: 0o755,
			body: "#!/bin/sh\necho update\n",
		},
		{
			name: "lib non executable archive",
			path: filepath.Join("go", "lib", "time", "zoneinfo.zip"),
			mode: 0o644,
			body: "archive",
		},
		{
			name: "pkg tool executable",
			path: filepath.Join("go", "pkg", "tool", "linux_amd64", "compile"),
			mode: 0o755,
			body: "compiler",
		},
	}

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}

	gzw := gzip.NewWriter(file)
	tw := tar.NewWriter(gzw)

	for _, tc := range testFiles {
		header := &tar.Header{
			Name: tc.path,
			Mode: tc.mode,
			Size: int64(len(tc.body)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write header for %s: %v", tc.name, err)
		}
		if _, err := tw.Write([]byte(tc.body)); err != nil {
			t.Fatalf("write body for %s: %v", tc.name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive file: %v", err)
	}

	if err := Extract(archivePath, dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	for _, tc := range testFiles {
		t.Run(tc.name, func(t *testing.T) {
			trimmedPath := filepath.Join(dest, tc.path[len("go/"):])
			info, err := os.Stat(trimmedPath)
			if err != nil {
				t.Fatalf("stat extracted file: %v", err)
			}
			if got := info.Mode().Perm(); got != os.FileMode(tc.mode) {
				t.Fatalf("extracted mode = %o, want %o", got, tc.mode)
			}
		})
	}
}
