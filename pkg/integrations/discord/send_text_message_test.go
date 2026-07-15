package discord

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SendTextMessage__Setup(t *testing.T) {
	component := &SendTextMessage{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing channel -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"content": "Hello"},
		})

		require.ErrorContains(t, err, "channel is required")
	})

	t.Run("no content or embed -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "123456789"},
		})

		require.ErrorContains(t, err, "either content, embed (title/description), or files is required")
	})

	t.Run("only empty file entries -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "123456789", "files": []any{map[string]any{"source": "url", "url": ""}, map[string]any{"source": "url", "url": ""}}},
		})

		require.ErrorContains(t, err, "either content, embed (title/description), or files is required")
	})

	t.Run("too many files -> error", func(t *testing.T) {
		files := make([]any, 11)
		for i := range files {
			files[i] = map[string]any{"source": "url", "url": fmt.Sprintf("https://example.com/file-%d.png", i)}
		}

		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "123456789", "files": files},
		})

		require.ErrorContains(t, err, "at most 10 files")
	})

	t.Run("invalid file URL -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "123456789", "files": []any{map[string]any{"source": "url", "url": "ftp://example.com/file.png"}}},
		})

		require.ErrorContains(t, err, "must be an http(s) URL")
	})

	t.Run("unresolved expression file URL is allowed", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"id":"123456789","name":"general","type":0}`), nil
		})

		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"channel": "123456789", "files": []any{map[string]any{"source": "url", "url": "{{ nodes.download.outputs.url }}"}}},
		})

		require.NoError(t, err)
	})

	t.Run("invalid embed color -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"channel":    "123456789",
				"embedTitle": "Title",
				"embedColor": "not-a-color",
			},
		})

		require.ErrorContains(t, err, "invalid embed color")
	})

	t.Run("valid content -> validates channel and stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "/channels/123456789")
			return jsonResponse(http.StatusOK, `{
				"id": "123456789",
				"name": "general",
				"type": 0
			}`), nil
		})

		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"channel": "123456789",
				"content": "Hello, Discord!",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendTextMessageMetadata)
		require.True(t, ok)
		assert.False(t, stored.HasEmbed)
		assert.Equal(t, "123456789", stored.Channel.ID)
		assert.Equal(t, "general", stored.Channel.Name)
	})

	t.Run("valid embed -> stores metadata", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
				"id": "123456789",
				"name": "general",
				"type": 0
			}`), nil
		})

		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
			},
			Metadata: metadata,
			Configuration: map[string]any{
				"channel":          "123456789",
				"embedTitle":       "My Embed",
				"embedDescription": "A description",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendTextMessageMetadata)
		require.True(t, ok)
		assert.True(t, stored.HasEmbed)
	})
}

func Test__SendTextMessage__Execute(t *testing.T) {
	component := &SendTextMessage{}

	t.Run("valid configuration -> sends message and emits", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "/channels/123456789/messages")
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "Bot test-bot-token", req.Header.Get("Authorization"))

			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload CreateMessageRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "Hello, Discord!", payload.Content)

			return jsonResponse(http.StatusOK, `{
				"id": "1234567890",
				"type": 0,
				"content": "Hello, Discord!",
				"channel_id": "123456789",
				"author": {"id": "999888777", "username": "TestBot", "bot": true},
				"timestamp": "2025-01-16T12:00:00.000Z"
			}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration: map[string]any{
				"channel": "123456789",
				"content": "Hello, Discord!",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "discord.message.sent", execState.Type)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "1234567890", data["id"])
		assert.Equal(t, "Hello, Discord!", data["content"])
	})

	t.Run("message with file URL -> fetches and uploads as attachment", func(t *testing.T) {
		// The artifact link is fetched through the workflow HTTP context (SSRF
		// policy applies); only the Discord upload uses the bot transport.
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("png-bytes")),
				},
			},
		}

		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			// Multipart message upload to Discord.
			assert.Contains(t, req.URL.String(), "/channels/123456789/messages")
			assert.Equal(t, "Bot test-bot-token", req.Header.Get("Authorization"))
			mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
			require.NoError(t, err)
			assert.Equal(t, "multipart/form-data", mediaType)

			reader := multipart.NewReader(req.Body, params["boundary"])
			form, err := reader.ReadForm(1 << 20)
			require.NoError(t, err)

			require.Len(t, form.Value["payload_json"], 1)
			var payload CreateMessageRequest
			require.NoError(t, json.Unmarshal([]byte(form.Value["payload_json"][0]), &payload))
			assert.Equal(t, "Here is the artifact", payload.Content)

			require.Len(t, form.File["files[0]"], 1)
			assert.Equal(t, "screenshot.png", form.File["files[0]"][0].Filename)

			return jsonResponse(http.StatusOK, `{
				"id": "1234567890",
				"type": 0,
				"content": "Here is the artifact",
				"channel_id": "123456789",
				"author": {"id": "999888777", "username": "TestBot", "bot": true},
				"timestamp": "2025-01-16T12:00:00.000Z"
			}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpContext,
			Configuration: map[string]any{
				"channel": "123456789",
				"content": "Here is the artifact",
				"files":   []any{map[string]any{"source": "url", "url": "https://artifacts.example.com/agents/abc/artifacts/screenshot.png?sig=xyz"}},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "discord.message.sent", execState.Type)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "artifacts.example.com", httpContext.Requests[0].URL.Host)
	})

	t.Run("file fetch failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("{}")),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpContext,
			Configuration: map[string]any{
				"channel": "123456789",
				"files":   []any{map[string]any{"source": "url", "url": "https://artifacts.example.com/missing.png"}},
			},
		})

		require.ErrorContains(t, err, "failed to fetch file")
	})

	t.Run("all file entries empty at execution with no content -> error", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           &contexts.HTTPContext{},
			Configuration: map[string]any{
				"channel": "123456789",
				"files":   []any{map[string]any{"source": "url", "url": ""}},
			},
		})

		require.ErrorContains(t, err, "nothing to send")
	})

	t.Run("message with embed -> sends correctly", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)

			var payload CreateMessageRequest
			require.NoError(t, json.Unmarshal(body, &payload))
			assert.Equal(t, "Hello!", payload.Content)
			require.Len(t, payload.Embeds, 1)
			assert.Equal(t, "Test Title", payload.Embeds[0].Title)
			assert.Equal(t, "Test Description", payload.Embeds[0].Description)
			assert.Equal(t, 5793266, payload.Embeds[0].Color) // #5865F2

			return jsonResponse(http.StatusOK, `{
				"id": "1234567890",
				"type": 0,
				"content": "Hello!",
				"channel_id": "123456789",
				"author": {"id": "999888777", "username": "TestBot", "bot": true},
				"timestamp": "2025-01-16T12:00:00.000Z"
			}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration: map[string]any{
				"channel":          "123456789",
				"content":          "Hello!",
				"embedTitle":       "Test Title",
				"embedDescription": "Test Description",
				"embedColor":       "#5865F2",
			},
		})

		require.NoError(t, err)
	})

	t.Run("API failure -> error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusForbidden, `{"message": "Missing Access"}`), nil
		})

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration: map[string]any{
				"channel": "123456789",
				"content": "Hello",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message")
	})

	t.Run("missing channel -> error", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-bot-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			Configuration:  map[string]any{"content": "Hello"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel is required")
	})
}

func Test__SendTextMessage__InlineImageIsRenderable(t *testing.T) {
	// A minimal but valid PNG (1x1 transparent pixel).
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89,
	}
	pngB64 := base64.StdEncoding.EncodeToString(pngBytes)

	t.Run("content without mime or filename still gets a png extension from the bytes", func(t *testing.T) {
		// Mirrors an OpenAI container-file artifact: base64 content, no mimeType.
		file, err := resolveFileAttachment(&Client{BotToken: "t"}, &contexts.HTTPContext{}, FileAttachment{
			Source:   "content",
			Content:  pngB64,
			Encoding: "base64",
		}, 1)
		require.NoError(t, err)
		require.Equal(t, "file-2.png", file.Name)
		require.Equal(t, "image/png", file.ContentType)
		require.Equal(t, pngBytes, file.Content)
	})

	t.Run("filename without extension gets the sniffed png extension", func(t *testing.T) {
		file, err := resolveFileAttachment(&Client{BotToken: "t"}, &contexts.HTTPContext{}, FileAttachment{
			Source:   "content",
			Content:  pngB64,
			Encoding: "base64",
			Filename: "inventory",
		}, 0)
		require.NoError(t, err)
		require.Equal(t, "inventory.png", file.Name)
	})
}

func Test__SendTextMessage__StructuredFileEntries(t *testing.T) {
	t.Run("decode reads object entries", func(t *testing.T) {
		entries, err := decodeFileAttachments([]any{
			map[string]any{"source": "url", "url": "https://example.com/report.pdf"},
			map[string]any{"source": "content", "content": "a,b\n1,2\n", "encoding": "text", "mimeType": "text/csv"},
		})
		require.NoError(t, err)
		require.Len(t, entries, 2)
		require.Equal(t, "https://example.com/report.pdf", entries[0].URL)
		require.Equal(t, "content", entries[1].Source)
		require.Equal(t, "text/csv", entries[1].MimeType)
	})

	t.Run("decode rejects a non-object entry", func(t *testing.T) {
		_, err := decodeFileAttachments([]any{"https://example.com/report.pdf"})
		require.ErrorContains(t, err, "must be a file entry")
	})

	t.Run("content entry with base64 encoding decodes bytes", func(t *testing.T) {
		client := &Client{BotToken: "t"}
		file, err := resolveFileAttachment(client, &contexts.HTTPContext{}, FileAttachment{
			Source:   "content",
			Content:  " aGVsbG8= ",
			Encoding: "base64",
			MimeType: "image/png",
		}, 0)
		require.NoError(t, err)
		require.Equal(t, []byte("hello"), file.Content)
		require.Equal(t, "file-1.png", file.Name)
	})

	t.Run("content entry with text encoding keeps raw content and filename override", func(t *testing.T) {
		client := &Client{BotToken: "t"}
		file, err := resolveFileAttachment(client, &contexts.HTTPContext{}, FileAttachment{
			Source:   "content",
			Content:  "a,b\n1,2",
			Filename: "export.csv",
		}, 0)
		require.NoError(t, err)
		require.Equal(t, []byte("a,b\n1,2"), file.Content)
		require.Equal(t, "export.csv", file.Name)
	})

	t.Run("url entry without scheme fails with guidance", func(t *testing.T) {
		client := &Client{BotToken: "t"}
		_, err := resolveFileAttachment(client, &contexts.HTTPContext{}, FileAttachment{
			Source: "url",
			URL:    "iVBORw0KGgo=",
		}, 0)
		require.ErrorContains(t, err, "set the entry's source to content")
	})

	t.Run("validate rejects unknown source and encoding", func(t *testing.T) {
		require.ErrorContains(t, validateFiles([]FileAttachment{{Source: "ftp"}}), "source must be")
		require.ErrorContains(t, validateFiles([]FileAttachment{{Source: "content", Encoding: "hex"}}), "encoding must be")
	})

	t.Run("validate allows an expression-driven encoding", func(t *testing.T) {
		require.NoError(t, validateFiles([]FileAttachment{{
			Source:   "content",
			Content:  "{{ $['Text Prompt'].data.artifacts[0].content }}",
			Encoding: "{{ $['Text Prompt'].data.artifacts[0].encoding }}",
		}}))
	})
}

func Test__SendTextMessage__InlineFileSizeLimit(t *testing.T) {
	client := &Client{BotToken: "t"}
	oversized := strings.Repeat("a", maxMessageFileSize+1)
	_, err := sendMessage(client, &contexts.HTTPContext{}, SendTextMessageConfiguration{
		Channel: "chan",
		Files:   []FileAttachment{{Source: "content", Content: oversized, Filename: "big.txt"}},
	}, CreateMessageRequest{})
	require.ErrorContains(t, err, "per-file limit")
}

func Test__SendTextMessage__URLEntryIsNamedFromContent(t *testing.T) {
	// A presigned-style URL with no extension in its path: the attachment name
	// and type must come from the content, otherwise Discord shows a generic
	// download instead of the image.
	pngBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89,
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(pngBytes))},
		},
	}

	file, err := resolveFileAttachment(&Client{BotToken: "t"}, httpCtx,
		FileAttachment{Source: fileSourceURL, URL: "https://artifacts.example.com/agents/abc/download?sig=xyz"}, 0)
	require.NoError(t, err)
	assert.Equal(t, "download.png", file.Name)
	assert.Equal(t, "image/png", file.ContentType)
	assert.Equal(t, pngBytes, file.Content)
}

func Test__SendTextMessage__ConfigurationContract(t *testing.T) {
	component := &SendTextMessage{}

	t.Run("structured entries are valid", func(t *testing.T) {
		require.NoError(t, configuration.ValidateConfiguration(component.Configuration(), map[string]any{
			"channel": "123456789",
			"files": []any{
				map[string]any{"source": "url", "url": "https://example.com/a.png"},
				map[string]any{"source": "content", "content": "aGk=", "encoding": "base64", "mimeType": "image/png"},
			},
		}))
	})

	t.Run("non-object entries are rejected", func(t *testing.T) {
		err := configuration.ValidateConfiguration(component.Configuration(), map[string]any{
			"channel": "123456789",
			"files":   []any{"https://example.com/a.png"},
		})
		require.ErrorContains(t, err, "must be an object")
	})
}
