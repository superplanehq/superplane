package public

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerArtifactUploadAndPublicDownload(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	signer := jwt.NewSigner("test")
	server := newRunnerArtifactTestServer(t, r, signer)

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Capture checkout", "Show the checkout state", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(database.Conn(), support.RandomName("line"), nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)

	const nodeID = "implementation-agent"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: nodeID,
		Name:   "Implementation Agent",
		Type:   models.NodeTypeComponent,
	}}, nil)
	require.NoError(t, database.Conn().Model(canvas).Update("factory_id", factoryModel.ID).Error)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), rootEvent)
	require.NoError(t, err)
	nodeExecution := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)

	now := time.Now()
	require.NoError(t, database.Conn().Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "Implement",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	token, err := runneraction.MintArtifactUploadToken(signer, runneraction.ArtifactUploadScope{
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		WorkOrderID:     order.ID,
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: nodeExecution.ID,
		NodeID:          nodeID,
	}, time.Minute)
	require.NoError(t, err)

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 16)...)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/artifacts", bytes.NewReader(png))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "image/png")
	request.Header.Set("Content-Disposition", `attachment; filename="checkout.png"`)
	request.Header.Set("X-SuperPlane-Artifact-Title", "Checkout")
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var uploaded runnerArtifactResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &uploaded))
	assert.Equal(t, "checkout.png", uploaded.Filename)
	assert.Equal(t, "image/png", uploaded.ContentType)
	assert.Equal(t, int64(len(png)), uploaded.SizeBytes)
	assert.Equal(t, "![Checkout]("+uploaded.PublicURL+")", uploaded.Markdown)

	fileID, err := uuid.Parse(uploaded.FileID)
	require.NoError(t, err)
	file, err := models.FindFile(database.Conn(), fileID)
	require.NoError(t, err)
	assert.Equal(t, models.FilePurposeArtifact, file.Purpose)
	assert.Equal(t, order.ID, *file.WorkOrderID)
	assert.Equal(t, models.FileStateReady, file.State)

	artifacts, err := order.ListArtifacts(database.Conn())
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, models.FactoryWorkOrderArtifactTypeFile, artifacts[0].Type)
	assert.Equal(t, uploaded.ArtifactID, artifacts[0].ID.String())

	publicURL, err := url.Parse(uploaded.PublicURL)
	require.NoError(t, err)
	download := httptest.NewRecorder()
	server.Router.ServeHTTP(download, httptest.NewRequest(http.MethodGet, publicURL.RequestURI(), nil))
	assert.Equal(t, http.StatusOK, download.Code)
	assert.Equal(t, "image/png", download.Header().Get("Content-Type"))
	assert.Equal(t, png, download.Body.Bytes())
}

func TestRunnerArtifactUploadFromPRDiscussion(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	signer := jwt.NewSigner("test")
	server := newRunnerArtifactTestServer(t, r, signer)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Adjust checkout", "Update the checkout UI", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/app",
		Number:     42,
		URL:        "https://github.com/acme/app/pull/42",
		Title:      "Adjust checkout",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	const nodeID = "address-pr-feedback"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: nodeID,
		Name:   "Address PR feedback",
		Type:   models.NodeTypeComponent,
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
	require.NoError(t, err)
	nodeExecution := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)
	_, err = runneraction.ResolveArtifactRunContext(db, run.ID)
	assert.ErrorIs(t, err, runneraction.ErrArtifactRunScopeNotFound)

	handler, err := factoryModel.CreatePRFeedbackHandler(
		db,
		canvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
		models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
	)
	require.NoError(t, err)
	_, err = pullRequest.CreateActivity(db, models.FactoryPullRequestActivityParams{
		RunID:             run.ID,
		FeedbackHandlerID: &handler.ID,
		Access:            models.FactoryPullRequestAccessExclusive,
	})
	require.NoError(t, err)

	runContext, err := runneraction.ResolveArtifactRunContext(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, runContext.WorkOrderID)
	assert.Nil(t, runContext.LineExecution)

	t.Setenv("JWT_SECRET", "test")
	environment := runneraction.AttachArtifactUploadEnv(core.ExecutionContext{
		ID:         nodeExecution.ID,
		RunID:      run.ID,
		WorkflowID: canvas.ID.String(),
		NodeID:     nodeID,
		BaseURL:    "http://files.test",
	}, nil, 60, true)
	require.True(t, runneraction.HasArtifactUploadToken(environment))
	var token string
	for _, variable := range environment {
		if variable.Name == runneraction.EnvSuperplaneArtifactToken {
			token = variable.Value
			break
		}
	}
	require.NotEmpty(t, token)

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 16)...)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/artifacts", bytes.NewReader(png))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "image/png")
	request.Header.Set("Content-Disposition", `attachment; filename="checkout-feedback.png"`)
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	events, err := order.ListEvents(db, 20, nil)
	require.NoError(t, err)
	var artifactAdded factory.WorkOrderArtifactAdded
	for _, event := range events {
		if event.Type != factory.EventTypeOrderArtifactAdded {
			continue
		}
		require.NoError(t, json.Unmarshal(event.Data, &artifactAdded))
		break
	}
	require.NotNil(t, artifactAdded.Automation)
	assert.Equal(t, canvas.ID, artifactAdded.Automation.AppID)
	assert.Equal(t, nodeID, artifactAdded.Automation.NodeID)
	assert.Equal(t, uuid.Nil, artifactAdded.Automation.LineID)
	assert.Nil(t, artifactAdded.Automation.StepIndex)
	require.NotNil(t, artifactAdded.Run)
	assert.Equal(t, run.ID, artifactAdded.Run.ID)

	require.NoError(t, db.Model(handler).Update("source", models.FactoryPRFeedbackHandlerSourcePullRequestChecks).Error)
	_, err = runneraction.ResolveArtifactRunContext(db, run.ID)
	assert.ErrorIs(t, err, runneraction.ErrArtifactRunScopeNotFound)
}

func newRunnerArtifactTestServer(
	t *testing.T,
	r *support.ResourceRegistry,
	signer *jwt.Signer,
) *Server {
	t.Helper()
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider(), nil)
	return server
}

func TestArtifactSignatureMatchesSupportedTypes(t *testing.T) {
	tests := []struct {
		contentType string
		prefix      []byte
	}{
		{contentType: "image/png", prefix: []byte("\x89PNG\r\n\x1a\nrest")},
		{contentType: "image/jpeg", prefix: []byte{0xff, 0xd8, 0xff, 0xe0}},
		{contentType: "image/webp", prefix: []byte("RIFF1234WEBP")},
		{contentType: "video/webm", prefix: []byte{0x1a, 0x45, 0xdf, 0xa3}},
		{contentType: "video/mp4", prefix: []byte("0000ftyp")},
	}
	for _, test := range tests {
		t.Run(test.contentType, func(t *testing.T) {
			assert.True(t, artifactSignatureMatches(test.contentType, test.prefix))
			assert.False(t, artifactSignatureMatches(test.contentType, []byte("not-media")))
		})
	}
}

func TestValidateArtifactUploadRequestEnforcesSizeAndMediaType(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 16)...)

	valid := httptest.NewRequest(http.MethodPost, "/api/v1/runner/artifacts", bytes.NewReader(png))
	valid.Header.Set("Content-Type", "image/png")
	valid.Header.Set("Content-Disposition", "attachment; filename*=UTF-8''screen.png")
	valid.ContentLength = int64(len(png))
	_, _, _, ok := validateArtifactUploadRequest(httptest.NewRecorder(), valid)
	assert.True(t, ok)

	oversized := httptest.NewRequest(http.MethodPost, "/api/v1/runner/artifacts", bytes.NewReader(png))
	oversized.Header = valid.Header.Clone()
	oversized.ContentLength = int64(100*1024*1024 + 1)
	recorder := httptest.NewRecorder()
	_, _, _, ok = validateArtifactUploadRequest(recorder, oversized)
	assert.False(t, ok)
	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)

	spoofed := httptest.NewRequest(http.MethodPost, "/api/v1/runner/artifacts", bytes.NewReader([]byte("not a png")))
	spoofed.Header = valid.Header.Clone()
	spoofed.ContentLength = 9
	recorder = httptest.NewRecorder()
	_, _, _, ok = validateArtifactUploadRequest(recorder, spoofed)
	assert.False(t, ok)
	assert.Equal(t, http.StatusUnsupportedMediaType, recorder.Code)
}

func TestArtifactMarkdownRendersImagesInlineAndVideosAsLinks(t *testing.T) {
	assert.Equal(t, "![Checkout](https://files.example/shot.png)", artifactMarkdown("Checkout", "image/png", "https://files.example/shot.png"))
	assert.Equal(t, "[Watch Checkout](https://files.example/demo.webm)", artifactMarkdown("Checkout", "video/webm", "https://files.example/demo.webm"))
	assert.Equal(t, `![Checkout\\](https://files.example/shot.png)`, artifactMarkdown(`Checkout\`, "image/png", "https://files.example/shot.png"))
	assert.Equal(t, `![Checkout\\\]](https://files.example/shot.png)`, artifactMarkdown(`Checkout\]`, "image/png", "https://files.example/shot.png"))
}

func TestDecodeArtifactTitleSupportsUTF8(t *testing.T) {
	assert.Equal(t, "Checkout → success", decodeArtifactTitle("Checkout%20%E2%86%92%20success"))
	assert.Equal(t, "注文確認", decodeArtifactTitle("%E6%B3%A8%E6%96%87%E7%A2%BA%E8%AA%8D"))
	assert.Equal(t, "100% complete", decodeArtifactTitle("100% complete"))
}

func TestArtifactFilenameMatchesContentType(t *testing.T) {
	assert.True(t, artifactFilenameMatchesContentType("screen.PNG", "image/png"))
	assert.True(t, artifactFilenameMatchesContentType("screen.jpeg", "image/jpeg"))
	assert.True(t, artifactFilenameMatchesContentType("demo.webm", "video/webm"))
	assert.False(t, artifactFilenameMatchesContentType("demo.png", "video/webm"))
	assert.False(t, artifactFilenameMatchesContentType("demo", "video/mp4"))
}

type artifactReadSeekCloser struct {
	*bytes.Reader
}

func (artifactReadSeekCloser) Close() error { return nil }

type artifactProvider struct {
	name      string
	content   []byte
	signedURL string
	signedTTL time.Duration
}

func (p *artifactProvider) Name() string { return p.name }
func (p *artifactProvider) Put(context.Context, string, io.Reader, blob.PutOptions) error {
	return nil
}
func (p *artifactProvider) Get(context.Context, string) (io.ReadCloser, error) {
	return artifactReadSeekCloser{Reader: bytes.NewReader(p.content)}, nil
}
func (p *artifactProvider) Head(context.Context, string) (*blob.ObjectInfo, error) {
	return &blob.ObjectInfo{Size: int64(len(p.content))}, nil
}
func (p *artifactProvider) Delete(context.Context, string) error { return nil }
func (p *artifactProvider) SignedGetURL(_ context.Context, _ string, ttl time.Duration) (string, error) {
	p.signedTTL = ttl
	return p.signedURL, nil
}

func TestServePublicArtifactSupportsFilesystemRanges(t *testing.T) {
	provider := &artifactProvider{name: blob.ProviderFilesystem, content: []byte("0123456789")}
	file := &models.File{
		ID:          uuid.New(),
		Filename:    "demo.webm",
		ContentType: "video/webm",
		SizeBytes:   10,
		StorageKey:  "artifact/demo",
		UpdatedAt:   time.Now(),
	}
	request := httptest.NewRequest(http.MethodGet, "/artifact", nil)
	request.Header.Set("Range", "bytes=2-5")
	response := httptest.NewRecorder()

	servePublicArtifact(response, request, provider, file)

	assert.Equal(t, http.StatusPartialContent, response.Code)
	assert.Equal(t, "2345", response.Body.String())
	assert.Equal(t, "bytes 2-5/10", response.Header().Get("Content-Range"))
}

func TestServePublicArtifactRedirectsGCSGets(t *testing.T) {
	provider := &artifactProvider{name: blob.ProviderGCS, signedURL: "https://storage.example/signed"}
	file := &models.File{Filename: "screen.png", ContentType: "image/png", SizeBytes: 8, StorageKey: "artifact/screen"}
	response := httptest.NewRecorder()

	servePublicArtifact(response, httptest.NewRequest(http.MethodGet, "/artifact", nil), provider, file)

	assert.Equal(t, http.StatusTemporaryRedirect, response.Code)
	assert.Equal(t, provider.signedURL, response.Header().Get("Location"))
	assert.Equal(t, 5*time.Minute, provider.signedTTL)
}

func TestServePublicArtifactAnswersHeadWithoutReadingObject(t *testing.T) {
	provider := &artifactProvider{name: blob.ProviderFilesystem}
	file := &models.File{Filename: "screen.png", ContentType: "image/png", SizeBytes: 42}
	response := httptest.NewRecorder()

	servePublicArtifact(response, httptest.NewRequest(http.MethodHead, "/artifact", nil), provider, file)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "42", response.Header().Get("Content-Length"))
	assert.Empty(t, response.Body.String())
}
