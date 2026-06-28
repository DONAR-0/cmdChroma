package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://huggingface.co/datasets/foo/data.parquet", true},
		{"http://example.com/file.jsonl", true},
		{"/local/path/file.parquet", false},
		{"./relative/file.jsonl", false},
		{"file.jsonl", false},
		{"", false},
		{"ftp://example.com/file", false},
	}
	for _, tt := range tests {
		got := IsURL(tt.input)
		require.Equal(t, tt.want, got, "IsURL(%q)", tt.input)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.input)
		require.Equal(t, tt.want, got, "FormatBytes(%d)", tt.input)
	}
}

func TestDownloadFile(t *testing.T) {
	content := []byte(`{"id":"1","text":"hello"}
{"id":"2","text":"world"}
`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	path, err := DownloadFile(context.Background(), server.URL, nil)
	require.NoError(t, err)
	require.NotEmpty(t, path)

	defer func() { _ = os.Remove(path) }()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, data)
}

func TestDownloadFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := DownloadFile(context.Background(), server.URL, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "404")
}

func TestDownloadFile_WithProgress(t *testing.T) {
	content := []byte(`{"id":"1","text":"hello"}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "25")
		_, _ = w.Write(content)
	}))
	defer server.Close()

	var calls []int64

	path, err := DownloadFile(context.Background(), server.URL, func(downloaded, total int64) {
		calls = append(calls, downloaded)

		require.Equal(t, int64(25), total)
	})
	require.NoError(t, err)

	defer func() { _ = os.Remove(path) }()

	require.NotEmpty(t, calls)
	require.Equal(t, int64(25), calls[len(calls)-1])
}
