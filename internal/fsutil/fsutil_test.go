package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirPreservesFilePermissions(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")

	testFiles := []struct {
		name string
		path string
		mode os.FileMode
	}{
		{
			name: "bin executable",
			path: filepath.Join("bin", "go"),
			mode: 0o755,
		},
		{
			name: "lib shell script",
			path: filepath.Join("lib", "time", "update.bash"),
			mode: 0o755,
		},
		{
			name: "lib wasm helper",
			path: filepath.Join("lib", "wasm", "go_js_wasm_exec"),
			mode: 0o755,
		},
		{
			name: "lib non executable archive",
			path: filepath.Join("lib", "time", "zoneinfo.zip"),
			mode: 0o644,
		},
		{
			name: "pkg tool executable",
			path: filepath.Join("pkg", "tool", "linux_amd64", "compile"),
			mode: 0o755,
		},
	}

	for _, tc := range testFiles {
		t.Run(tc.name, func(t *testing.T) {
			fullPath := filepath.Join(src, tc.path)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				t.Fatalf("create parent directories: %v", err)
			}
			if err := os.WriteFile(fullPath, []byte(tc.name), tc.mode); err != nil {
				t.Fatalf("create source file: %v", err)
			}
			if err := os.Chmod(fullPath, tc.mode); err != nil {
				t.Fatalf("set source mode: %v", err)
			}
		})
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() error = %v", err)
	}

	for _, tc := range testFiles {
		t.Run("copied "+tc.name, func(t *testing.T) {
			info, err := os.Stat(filepath.Join(dst, tc.path))
			if err != nil {
				t.Fatalf("stat copied file: %v", err)
			}
			if got := info.Mode().Perm(); got != tc.mode {
				t.Fatalf("copied mode = %o, want %o", got, tc.mode)
			}
		})
	}
}
