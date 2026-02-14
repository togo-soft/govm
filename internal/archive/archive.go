package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract extracts a tar.gz or zip archive to destination.
func Extract(src, dest string) error {
	if strings.HasSuffix(src, ".tar.gz") {
		return extractTarGz(src, dest)
	} else if strings.HasSuffix(src, ".zip") {
		return extractZip(src, dest)
	}
	return fmt.Errorf("unsupported archive format: %s", src)
}

func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
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
			continue
		}

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
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}
			file.Close()
			if err := os.Chmod(path, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractZip(src, dest string) error {
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
			continue
		}

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

		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
		if err := os.Chmod(path, file.FileInfo().Mode()); err != nil {
			return err
		}
	}
	return nil
}
