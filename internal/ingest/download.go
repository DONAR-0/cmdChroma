package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const defaultDownloadTimeout = 5 * time.Minute

// IsURL returns true if the source string is an HTTP/HTTPS URL.
func IsURL(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

// DownloadFile downloads a URL to a temporary file and returns its path.
// The caller must remove the file after use. A non-nil onProgress callback
// is called periodically with (downloaded, total) bytes.
func DownloadFile(ctx context.Context, url string, onProgress func(downloaded, total int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: defaultDownloadTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	// Extract extension from the final URL path for temp file naming
	urlPath := path.Base(resp.Request.URL.Path)

	ext := ".parquet"
	if idx := strings.LastIndex(urlPath, "."); idx >= 0 {
		ext = urlPath[idx:]
	}

	tmpFile, err := os.CreateTemp("", "chroma-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	total := resp.ContentLength

	var (
		written   int64
		lastPrint time.Time
	)

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpFile.Name())

				return "", fmt.Errorf("failed to write temp file: %w", writeErr)
			}

			written += int64(n)

			if onProgress != nil && (time.Since(lastPrint) > time.Second || (total > 0 && written >= total)) {
				onProgress(written, total)

				lastPrint = time.Now()
			}
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())

			return "", fmt.Errorf("download failed: %w", readErr)
		}
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Final progress update
	if onProgress != nil {
		onProgress(written, total)
	}

	return tmpFile.Name(), nil
}

// FormatBytes returns a human-readable byte string (KB, MB, GB).
func FormatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
