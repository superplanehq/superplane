package tasks

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const artifactUploadTimeout = 15 * time.Minute

var artifactContentTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".webm": "video/webm",
	".mp4":  "video/mp4",
}

// uploadTaskArtifactFile stores the file on the task and returns its id.
// The file is not on the task until an artifact refers to it.
func uploadTaskArtifactFile(ctx core.CommandContext, workspaceID, taskID, path string) (string, error) {
	contentType, ok := artifactContentTypes[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return "", fmt.Errorf("file type is not supported; use PNG, JPEG, WebP, WebM, or MP4")
	}
	file, err := os.Open(path) // #nosec G304 -- CLI user-provided path
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	body := openapi_client.NewFilesCreateWorkOrderFileBody()
	body.SetFilename(filepath.Base(path))
	body.SetContentType(contentType)
	body.SetPurpose("artifact")
	created, _, err := ctx.API.FilesAPI.
		FilesCreateWorkOrderFile(ctx.Context, workspaceID, taskID).
		Body(*body).
		Execute()
	if err != nil {
		return "", err
	}
	createdFile := created.GetFile()
	fileID := createdFile.GetId()
	if fileID == "" {
		return "", fmt.Errorf("the server did not return a file id")
	}
	if err := putFileContent(ctx, fileID, contentType, file, info.Size()); err != nil {
		return "", err
	}
	return fileID, nil
}

func putFileContent(ctx core.CommandContext, fileID, contentType string, content io.Reader, size int64) error {
	config := ctx.API.GetConfig()
	if config == nil {
		return fmt.Errorf("api client config is required")
	}
	baseURL, err := config.ServerURLWithContext(ctx.Context, "FilesAPIService.FilesCreateWorkOrderFile")
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/v1/files/%s/content", strings.TrimRight(baseURL, "/"), url.PathEscape(fileID))
	request, err := http.NewRequestWithContext(ctx.Context, http.MethodPut, endpoint, content)
	if err != nil {
		return err
	}
	request.ContentLength = size
	for name, value := range config.DefaultHeader {
		request.Header.Set(name, value)
	}
	request.Header.Set("Content-Type", contentType)

	client := &http.Client{Timeout: artifactUploadTimeout}
	if config.HTTPClient != nil {
		client = &http.Client{Transport: config.HTTPClient.Transport, Timeout: artifactUploadTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return fmt.Errorf("upload file: %s: %s", response.Status, strings.TrimSpace(string(message)))
}
