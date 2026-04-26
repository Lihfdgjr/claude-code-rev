package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"claudecode/internal/core"
)

const (
	filesUploadEndpoint = "https://api.anthropic.com/v1/files"
	filesUploadMaxBytes = 10 * 1024 * 1024
	filesUploadBeta     = "files-api-2025-04-14"
	anthropicVersionHdr = "2023-06-01"
)

type filesUploadTool struct{}

// NewFilesUpload returns a Tool that uploads a local file to the Anthropic
// Files API and returns the resulting file id.
func NewFilesUpload() core.Tool { return &filesUploadTool{} }

func (t *filesUploadTool) Name() string { return "FilesUpload" }

func (t *filesUploadTool) Description() string {
	return "Upload a local file (cap 10 MiB) to the Anthropic Files API and return its file id. Requires ANTHROPIC_API_KEY in env."
}

func (t *filesUploadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {"type": "string", "description": "Absolute path to the file to upload."},
			"purpose": {"type": "string", "description": "File purpose. Defaults to user_data."}
		},
		"required": ["file_path"],
		"additionalProperties": false
	}`)
}

func (t *filesUploadTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		FilePath string `json:"file_path"`
		Purpose  string `json:"purpose"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if args.FilePath == "" {
		return "", fmt.Errorf("file_path required")
	}
	if !filepath.IsAbs(args.FilePath) {
		return "", fmt.Errorf("file_path must be absolute")
	}
	if args.Purpose == "" {
		args.Purpose = "user_data"
	}

	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	info, err := os.Stat(args.FilePath)
	if err != nil {
		return "", fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("file_path is a directory")
	}
	if info.Size() > filesUploadMaxBytes {
		return "", fmt.Errorf("file too large: %d bytes exceeds %d cap", info.Size(), filesUploadMaxBytes)
	}

	f, err := os.Open(args.FilePath)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, filesUploadMaxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	if int64(len(data)) > filesUploadMaxBytes {
		return "", fmt.Errorf("file too large after read")
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	if err := mw.WriteField("purpose", args.Purpose); err != nil {
		return "", fmt.Errorf("write field purpose: %w", err)
	}

	filename := filepath.Base(args.FilePath)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	hdr.Set("Content-Type", detectContentType(filename, data))
	part, err := mw.CreatePart(hdr)
	if err != nil {
		return "", fmt.Errorf("create part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("write part: %w", err)
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, filesUploadEndpoint, &body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersionHdr)
	req.Header.Set("anthropic-beta", filesUploadBeta)
	req.Header.Set("content-type", mw.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("files api %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Filename   string `json:"filename"`
		MimeType   string `json:"mime_type"`
		SizeBytes  int64  `json:"size_bytes"`
		CreatedAt  string `json:"created_at"`
		Downloadable bool `json:"downloadable"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if parsed.ID == "" {
		return "", fmt.Errorf("response missing id: %s", strings.TrimSpace(string(respBody)))
	}

	return fmt.Sprintf("file_id=%s filename=%s size=%d mime_type=%s", parsed.ID, parsed.Filename, parsed.SizeBytes, parsed.MimeType), nil
}

func detectContentType(filename string, data []byte) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".json":
		return "application/json"
	case ".txt", ".md":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	}
	if ct := http.DetectContentType(data); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
