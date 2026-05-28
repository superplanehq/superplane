package ndjson

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/supergit/internal/storage"
)

type CommitMetadata struct {
	TargetBranch    string `json:"target_branch"`
	BaseBranch      string `json:"base_branch"`
	ExpectedHeadSHA string `json:"expected_head_sha"`
	CommitMessage   string `json:"commit_message"`
	Author          struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	Files []CommitFile `json:"files"`
}

type CommitFile struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	ContentID string `json:"content_id"`
	Mode      string `json:"mode"`
}

type BlobChunk struct {
	ContentID string `json:"content_id"`
	Data      string `json:"data"`
	EOF       bool   `json:"eof"`
}

type envelope struct {
	Metadata  *CommitMetadata `json:"metadata,omitempty"`
	BlobChunk *BlobChunk      `json:"blob_chunk,omitempty"`
}

func ParseCommitBody(body io.Reader, limits storage.Limits) (storage.CommitOptions, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var metadata *CommitMetadata
	blobs := map[string][]byte{}

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var item envelope
		if err := json.Unmarshal(line, &item); err != nil {
			return storage.CommitOptions{}, fmt.Errorf("invalid ndjson line: %w", err)
		}

		switch {
		case item.Metadata != nil:
			if metadata != nil {
				return storage.CommitOptions{}, fmt.Errorf("duplicate metadata line")
			}
			metadata = item.Metadata
		case item.BlobChunk != nil:
			chunk := item.BlobChunk
			if strings.TrimSpace(chunk.ContentID) == "" {
				return storage.CommitOptions{}, fmt.Errorf("blob_chunk content_id is required")
			}

			decoded, err := base64.StdEncoding.DecodeString(chunk.Data)
			if err != nil {
				return storage.CommitOptions{}, fmt.Errorf("invalid blob_chunk data for %q: %w", chunk.ContentID, err)
			}

			blobs[chunk.ContentID] = append(blobs[chunk.ContentID], decoded...)
			if chunk.EOF {
				if limits.MaxFileBytes > 0 && int64(len(blobs[chunk.ContentID])) > limits.MaxFileBytes {
					return storage.CommitOptions{}, storage.ErrFileTooLarge
				}
			}
		default:
			return storage.CommitOptions{}, fmt.Errorf("ndjson line must contain metadata or blob_chunk")
		}
	}

	if err := scanner.Err(); err != nil {
		return storage.CommitOptions{}, err
	}

	if metadata == nil {
		return storage.CommitOptions{}, fmt.Errorf("metadata line is required")
	}

	operations := make([]storage.FileOperation, 0, len(metadata.Files))
	var totalBytes int64

	for _, file := range metadata.Files {
		operation := strings.TrimSpace(strings.ToLower(file.Operation))
		switch operation {
		case "delete":
			operations = append(operations, storage.FileOperation{
				Path:   file.Path,
				Delete: true,
			})
		case "upsert", "add", "update":
			content, ok := blobs[file.ContentID]
			if !ok {
				return storage.CommitOptions{}, fmt.Errorf("missing blob content for %q", file.ContentID)
			}
			totalBytes += int64(len(content))
			operations = append(operations, storage.FileOperation{
				Path:      file.Path,
				Content:   bytes.NewReader(content),
				SizeBytes: int64(len(content)),
			})
		default:
			return storage.CommitOptions{}, fmt.Errorf("unsupported file operation %q", file.Operation)
		}
	}

	if limits.MaxCommitBytes > 0 && totalBytes > limits.MaxCommitBytes {
		return storage.CommitOptions{}, storage.ErrCommitTooLarge
	}

	return storage.CommitOptions{
		Branch:          metadata.TargetBranch,
		BaseBranch:      metadata.BaseBranch,
		ExpectedHeadSHA: metadata.ExpectedHeadSHA,
		Message:         metadata.CommitMessage,
		Author: storage.CommitAuthor{
			Name:  metadata.Author.Name,
			Email: metadata.Author.Email,
		},
		Operations: operations,
	}, nil
}
