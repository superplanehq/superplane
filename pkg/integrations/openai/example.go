package openai

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_text_prompt.json
var exampleOutputTextPromptBytes []byte

var exampleOutputTextPromptOnce sync.Once
var exampleOutputTextPrompt map[string]any

func (c *CreateResponse) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTextPromptOnce, exampleOutputTextPromptBytes, &exampleOutputTextPrompt)
}

//go:embed example_output_get_file.json
var exampleOutputGetFileBytes []byte

var exampleOutputGetFileOnce sync.Once
var exampleOutputGetFile map[string]any

func (c *GetFile) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetFileOnce, exampleOutputGetFileBytes, &exampleOutputGetFile)
}

//go:embed example_output_download_file.json
var exampleOutputDownloadFileBytes []byte

var exampleOutputDownloadFileOnce sync.Once
var exampleOutputDownloadFile map[string]any

func (c *DownloadFile) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDownloadFileOnce, exampleOutputDownloadFileBytes, &exampleOutputDownloadFile)
}

//go:embed example_output_download_container_file.json
var exampleOutputDownloadContainerFileBytes []byte

var exampleOutputDownloadContainerFileOnce sync.Once
var exampleOutputDownloadContainerFile map[string]any

func (c *DownloadContainerFile) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDownloadContainerFileOnce, exampleOutputDownloadContainerFileBytes, &exampleOutputDownloadContainerFile)
}
