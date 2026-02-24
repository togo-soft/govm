package download

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type progressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	filename string
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.current += int64(n)

	if pr.total <= 0 {
		currentMB := float64(pr.current) / (1024 * 1024)
		fmt.Printf("\r[===>] Downloaded: %.1f MB", currentMB)
	} else {
		percent := float64(pr.current) / float64(pr.total) * 100
		barWidth := 30
		filledWidth := int(percent / 100 * float64(barWidth))
		if filledWidth > barWidth {
			filledWidth = barWidth
		}
		if filledWidth < 0 {
			filledWidth = 0
		}

		bar := strings.Repeat("=", filledWidth) + strings.Repeat(" ", barWidth-filledWidth)
		totalMB := float64(pr.total) / (1024 * 1024)
		currentMB := float64(pr.current) / (1024 * 1024)

		fmt.Printf("\r[%s] %.1f MB / %.1f MB (%.1f%%)",
			bar, currentMB, totalMB, percent)
	}

	if err == io.EOF {
		fmt.Println()
	}

	return
}

// File downloads a file from url to destDir and returns the path to the downloaded file.
func File(url, destDir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status code: %d", resp.StatusCode)
	}

	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	filePath := filepath.Join(destDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	slog.Info("downloading", "file", filename)

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

	slog.Info("download completed", "file", filename)
	return filePath, nil
}

// VerifySHA256 verifies file SHA256 checksum.
func VerifySHA256(filePath, expectedSum string) error {
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
	filename := filepath.Base(filePath)

	if actual == expectedSum {
		slog.Info("SHA256 verification passed", "file", filename)
		return nil
	}

	fmt.Printf("✗ SHA256 verification FAILED: %s\n", filename)
	fmt.Printf("  Expected: %s\n", expectedSum)
	fmt.Printf("  Got:      %s\n", actual)
	slog.Error("SHA256 verification failed", "file", filename, "expected", expectedSum, "actual", actual)

	return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedSum, actual)
}
