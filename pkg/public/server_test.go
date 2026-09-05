package public

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type fakePublicUsageService struct {
	checkAccountResponse *usagepb.CheckAccountLimitsResponse
	checkAccountErr      error
	checkAccount         func(*usagepb.AccountState) *usagepb.CheckAccountLimitsResponse
}

func (s *fakePublicUsageService) Enabled() bool {
	return true
}

func (s *fakePublicUsageService) SetupAccount(context.Context, string) (*usagepb.SetupAccountResponse, error) {
	return &usagepb.SetupAccountResponse{}, nil
}

func (s *fakePublicUsageService) SetupOrganization(context.Context, string, string, usage.SetupOrganizationDetails) (*usagepb.SetupOrganizationResponse, error) {
	return &usagepb.SetupOrganizationResponse{}, nil
}

func (s *fakePublicUsageService) DescribeAccountLimits(context.Context, string) (*usagepb.DescribeAccountLimitsResponse, error) {
	return &usagepb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakePublicUsageService) DescribeOrganizationLimits(context.Context, string) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	return &usagepb.DescribeOrganizationLimitsResponse{}, nil
}

func (s *fakePublicUsageService) DescribeOrganizationUsage(context.Context, string) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return &usagepb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakePublicUsageService) CheckAccountLimits(
	_ context.Context,
	_ string,
	state *usagepb.AccountState,
) (*usagepb.CheckAccountLimitsResponse, error) {
	if s.checkAccountErr != nil {
		return nil, s.checkAccountErr
	}

	if s.checkAccount != nil {
		return s.checkAccount(state), nil
	}

	if s.checkAccountResponse != nil {
		return s.checkAccountResponse, nil
	}

	return &usagepb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakePublicUsageService) CheckOrganizationLimits(
	context.Context,
	string,
	*usagepb.OrganizationState,
	*usagepb.CanvasState,
) (*usagepb.CheckOrganizationLimitsResponse, error) {
	return &usagepb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

var _ usage.Service = (*fakePublicUsageService)(nil)

func Test__HealthCheckEndpoint(t *testing.T) {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	registry, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	signer := jwt.NewSigner("test")
	oidcProvider := support.NewOIDCProvider()
	gitProvider := inmemory.NewProvider()
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, oidcProvider, gitProvider, "", "", "", "test", "/app/templates", authService, nil, false)
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/health",
	})

	require.Equal(t, 200, response.Code)
}

func Test__OpenAPIEndpoints(t *testing.T) {
	checkSwaggerFiles(t)

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	signer := jwt.NewSigner("test")
	registry, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	oidcProvider := support.NewOIDCProvider()
	gitProvider := inmemory.NewProvider()
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, oidcProvider, gitProvider, "", "", "", "test", "/app/templates", authService, nil, false)
	require.NoError(t, err)

	server.RegisterOpenAPIHandler()

	t.Run("OpenAPI JSON spec is accessible", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs/superplane.swagger.json",
		})

		require.Equal(t, 200, response.Code)
		require.NotEmpty(t, response.Body.String())
		require.Contains(t, response.Header().Get("Content-Type"), "application/json")

		var jsonData map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &jsonData)
		require.NoError(t, err, "Response should be valid JSON")

		assert.Contains(t, jsonData, "swagger", "Should contain 'swagger' field")
		assert.Contains(t, jsonData, "paths", "Should contain 'paths' field")
	})

	t.Run("Swagger UI HTML is accessible", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs",
		})

		require.Equal(t, 200, response.Code)
		require.NotEmpty(t, response.Body.String())
		require.Contains(t, response.Header().Get("Content-Type"), "text/html")

		require.Contains(t, response.Body.String(), "<html")
		require.Contains(t, response.Body.String(), "swagger-ui")
		require.Contains(t, response.Body.String(), "SwaggerUIBundle")
	})

	t.Run("OpenAPI spec is accessible via directory path", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs/superplane.swagger.json",
		})

		require.Equal(t, 200, response.Code)
		require.NotEmpty(t, response.Body.String())
		require.Contains(t, response.Header().Get("Content-Type"), "application/json")

		var jsonData map[string]interface{}
		err := json.Unmarshal(response.Body.Bytes(), &jsonData)
		require.NoError(t, err, "Response should be valid JSON")
	})

	t.Run("Non-existent file returns 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/docs/non-existent-file.json",
		})

		require.Equal(t, 404, response.Code)
	})
}

func Test__GRPCGatewayRegistration(t *testing.T) {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	signer := jwt.NewSigner("test")
	registry, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)
	oidcProvider := support.NewOIDCProvider()
	gitProvider := inmemory.NewProvider()
	server, err := NewServer(&crypto.NoOpEncryptor{}, registry, signer, oidcProvider, gitProvider, "", "", "", "test", "/app/templates", authService, nil, false)
	require.NoError(t, err)

	registerTestGRPCGateway(t, server, authService, registry, &crypto.NoOpEncryptor{}, oidcProvider, gitProvider, nil)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/api/v1/canvases/is-alive",
	})

	require.Equal(t, "", response.Body.String())
	require.Equal(t, 200, response.Code)
}

func Test__HandleWebhook_DoesNotRunNodesForSoftDeletedOrganization(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		r.GitProvider,
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

	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStateReady,
		Secret: []byte("secret"),
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	nodeID := "start-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: nodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
		},
		[]models.Edge{},
	)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", nodeID).
		Update("webhook_id", webhookID).
		Error)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	response := execRequest(server, requestParams{
		method: "POST",
		path:   "/webhooks/" + webhookID.String(),
		body:   []byte(`{"ok": true}`),
	})

	require.Equal(t, http.StatusNotFound, response.Code)

	eventCount, err := models.CountCanvasEvents(database.Conn(), canvas.ID, nodeID)
	require.NoError(t, err)
	assert.Zero(t, eventCount)
}

type canvasesGatewayStubServer struct {
	pbCanvases.UnimplementedCanvasesServer
	createCanvasCalled bool
}

func (s *canvasesGatewayStubServer) CreateCanvas(
	context.Context,
	*pbCanvases.CreateCanvasRequest,
) (*pbCanvases.CreateCanvasResponse, error) {
	s.createCanvasCalled = true
	return &pbCanvases.CreateCanvasResponse{}, nil
}

func Test__GRPCGatewayRejectsUnknownFields(t *testing.T) {
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, newGRPCGatewayMarshaler()),
	)

	server := &canvasesGatewayStubServer{}
	err := pbCanvases.RegisterCanvasesHandlerServer(context.Background(), mux, server)
	require.NoError(t, err)

	requestBody := `{
  "name": "unknown-field-test",
  "hello": "what"
}`

	request := httptest.NewRequest(http.MethodPost, "/api/v1/canvases", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	var statusBody struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &statusBody))
	require.Equal(t, int32(3), statusBody.Code)
	require.Contains(t, statusBody.Message, `unknown field "hello"`)
	require.False(t, server.createCanvasCalled)
}

// Helper function to check if the required Swagger files exist
func checkSwaggerFiles(t *testing.T) {
	apiDir := os.Getenv("SWAGGER_BASE_PATH")

	// Check if the directory exists
	dirInfo, err := os.Stat(apiDir)
	require.NoError(t, err, "api/swagger directory should exist")
	require.True(t, dirInfo.IsDir(), "api/swagger should be a directory")

	// Check for the OpenAPI spec JSON file
	specPath := filepath.Join(apiDir, "superplane.swagger.json")
	fileInfo, err := os.Stat(specPath)
	require.NoError(t, err, "superplane.swagger.json should exist")
	require.False(t, fileInfo.IsDir(), "superplane.swagger.json should be a file")
	require.Greater(t, fileInfo.Size(), int64(0), "superplane.swagger.json should not be empty")

	// Check for the Swagger UI HTML file
	htmlPath := filepath.Join(apiDir, "swagger-ui.html")
	fileInfo, err = os.Stat(htmlPath)
	require.NoError(t, err, "swagger-ui.html should exist")
	require.False(t, fileInfo.IsDir(), "swagger-ui.html should be a file")
	require.Greater(t, fileInfo.Size(), int64(0), "swagger-ui.html should not be empty")

	// Check that the JSON file is valid
	jsonData, err := os.ReadFile(specPath)
	require.NoError(t, err, "Should be able to read swagger JSON file")

	var data map[string]interface{}
	err = json.Unmarshal(jsonData, &data)
	require.NoError(t, err, "superplane.swagger.json should contain valid JSON")

	// Check that the HTML file contains expected content
	htmlData, err := os.ReadFile(htmlPath)
	require.NoError(t, err, "Should be able to read swagger UI HTML file")
	require.Contains(t, string(htmlData), "swagger-ui", "HTML should contain swagger-ui reference")
}

type requestParams struct {
	method       string
	path         string
	body         []byte
	signature    string
	authToken    string
	authCookie   string
	contentType  string
	customSource bool
}

func execRequest(server *Server, params requestParams) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(params.method, params.path, bytes.NewReader(params.body))

	if params.contentType != "" {
		req.Header.Add("Content-Type", params.contentType)
	}

	// Set the appropriate signature header based on the path
	if params.signature != "" {
		if params.customSource {
			req.Header.Add("X-Signature-256", params.signature)
		} else {
			req.Header.Add("X-Semaphore-Signature-256", params.signature)
		}
	}

	if params.authToken != "" {
		req.Header.Add("Authorization", "Bearer "+params.authToken)
	}

	if params.authCookie != "" {
		req.AddCookie(&http.Cookie{Name: "account_token", Value: params.authCookie})
	}

	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	return res
}

// mockAuthService wraps a real auth service and allows us to inject errors
type mockAuthService struct {
	*authorization.AuthService
	setupOrgError error
}

func (m *mockAuthService) SetupOrganization(tx *gorm.DB, orgID, ownerID string) error {
	if m.setupOrgError != nil {
		return m.setupOrgError
	}
	return m.AuthService.SetupOrganization(tx, orgID, ownerID)
}

func Test__CreateInitialWorkspaceSerializesRetries(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.DB(t.Context()),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "github-owner-id", "github-owner", "GitHub Owner", ""),
	))

	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		r.GitProvider,
		"",
		"localhost",
		"",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)

	attemptID := uuid.New()
	body, err := json.Marshal(initialWorkspaceRequest{Owner: "GitHub Owner", AttemptID: attemptID.String()})
	require.NoError(t, err)

	const requestCount = 24
	responses := make([]*httptest.ResponseRecorder, requestCount)
	start := make(chan struct{})
	var requests sync.WaitGroup
	for index := range requestCount {
		requests.Add(1)
		go func() {
			defer requests.Done()
			<-start
			request := httptest.NewRequest(http.MethodPost, "/account/onboarding", bytes.NewReader(body))
			request = request.WithContext(accountContext(r.Account))
			responses[index] = httptest.NewRecorder()
			server.createInitialWorkspace(responses[index], request)
		}()
	}
	close(start)
	requests.Wait()

	var first initialWorkspaceResponse
	for index, response := range responses {
		require.Equal(t, http.StatusOK, response.Code)
		var result initialWorkspaceResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		if index == 0 {
			first = result
		}
		assert.Equal(t, first, result)
	}

	organizations, err := models.ListOrganizationsCreatedByAccount(database.DB(t.Context()), r.Account.ID)
	require.NoError(t, err)
	matchingWorkspaces := 0
	var initialWorkspace *models.Factory
	for _, organization := range organizations {
		workspaces, listErr := models.ListFactories(database.DB(t.Context()), organization.ID)
		require.NoError(t, listErr)
		for _, workspace := range workspaces {
			if workspace.HasInitialOnboardingAttempt(attemptID) {
				matchingWorkspaces++
				initialWorkspace = &workspace
			}
		}
	}
	assert.Equal(t, 1, matchingWorkspaces)
	require.NotNil(t, initialWorkspace)

	completedAt := time.Now()
	require.NoError(t, database.DB(t.Context()).Model(initialWorkspace).Update("onboarding_completed_at", completedAt).Error)
	retry := httptest.NewRequest(http.MethodPost, "/account/onboarding", bytes.NewReader(body))
	retry = retry.WithContext(accountContext(r.Account))
	retryResponse := httptest.NewRecorder()
	server.createInitialWorkspace(retryResponse, retry)
	require.Equal(t, http.StatusOK, retryResponse.Code)
	var retryResult initialWorkspaceResponse
	require.NoError(t, json.Unmarshal(retryResponse.Body.Bytes(), &retryResult))
	assert.Equal(t, first, retryResult)
}

func Test__InitialOrganizationNameUsesEmailWhenAccountNameEmpty(t *testing.T) {
	account := &models.Account{Name: "  ", Email: "dev@superplane.local"}
	assert.Equal(t, "dev", initialOrganizationName(account, ""))
	assert.Equal(t, "dev", initialOrganizationName(account, "dev"))
	assert.True(t, accountMayNameOrganization(nil, account, "dev"))
}

func Test__CreateInitialWorkspaceUsesAccountNameWithoutGitHub(t *testing.T) {
	r := support.Setup(t)
	account, err := models.CreateAccount("Ada Lovelace", "ada-onboarding@superplane.local")
	require.NoError(t, err)

	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		r.GitProvider,
		"",
		"localhost",
		"",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)

	attemptID := uuid.New()
	body, err := json.Marshal(initialWorkspaceRequest{Owner: "", AttemptID: attemptID.String()})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/account/onboarding", bytes.NewReader(body))
	request = request.WithContext(accountContext(account))
	response := httptest.NewRecorder()
	server.createInitialWorkspace(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	organization, err := models.FindOrganizationByName("Ada Lovelace")
	require.NoError(t, err)
	workspaces, err := models.ListFactories(database.DB(t.Context()), organization.ID)
	require.NoError(t, err)
	require.Len(t, workspaces, 1)
}

func Test__OrganizationCreationSerializesLimitChecks(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.DB(t.Context()),
		models.NewAccountLinkedAccount(r.Account.ID, models.ProviderGitHub, "github-owner-id", "github-owner", "GitHub Owner", ""),
	))
	initialOrganizations, err := models.CountOrganizationsByBillingAccount(database.DB(t.Context()), r.Account.ID.String())
	require.NoError(t, err)
	maxOrganizations := int32(initialOrganizations + 1)
	var limitChecks atomic.Int32
	concurrentLimitChecks := make(chan struct{})

	usageService := &fakePublicUsageService{
		checkAccount: func(state *usagepb.AccountState) *usagepb.CheckAccountLimitsResponse {
			if limitChecks.Add(1) == 1 {
				select {
				case <-concurrentLimitChecks:
				case <-time.After(100 * time.Millisecond):
				}
			} else {
				close(concurrentLimitChecks)
			}
			response := &usagepb.CheckAccountLimitsResponse{
				Allowed: state.GetOrganizations() <= maxOrganizations,
				Limits:  &usagepb.AccountLimits{MaxOrganizations: maxOrganizations},
			}
			if response.Allowed {
				return response
			}
			response.Violations = []*usagepb.LimitViolation{{
				Limit:           usagepb.LimitName_LIMIT_NAME_MAX_ORGANIZATIONS,
				ConfiguredLimit: int64(maxOrganizations),
				CurrentValue:    int64(state.GetOrganizations()),
			}}
			return response
		},
	}
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		r.GitProvider,
		"",
		"localhost",
		"",
		"test",
		"/app/templates",
		r.AuthService,
		usageService,
		false,
	)
	require.NoError(t, err)

	onboardingBody, err := json.Marshal(initialWorkspaceRequest{Owner: "GitHub Owner", AttemptID: uuid.NewString()})
	require.NoError(t, err)
	organizationBody, err := json.Marshal(OrganizationCreationRequest{Name: "Manual Organization"})
	require.NoError(t, err)

	responses := []*httptest.ResponseRecorder{httptest.NewRecorder(), httptest.NewRecorder()}
	start := make(chan struct{})
	var requests sync.WaitGroup
	requests.Add(2)
	go func() {
		defer requests.Done()
		<-start
		request := httptest.NewRequest(http.MethodPost, "/account/onboarding", bytes.NewReader(onboardingBody))
		server.createInitialWorkspace(responses[0], request.WithContext(accountContext(r.Account)))
	}()
	go func() {
		defer requests.Done()
		<-start
		request := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(organizationBody))
		server.createOrganization(responses[1], request.WithContext(accountContext(r.Account)))
	}()
	close(start)
	requests.Wait()

	assert.ElementsMatch(t, []int{http.StatusOK, http.StatusTooManyRequests}, []int{responses[0].Code, responses[1].Code})
	organizations, err := models.CountOrganizationsByBillingAccount(database.DB(t.Context()), r.Account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, initialOrganizations+1, organizations)
}

func Test__CreateOrganization(t *testing.T) {
	t.Run("organization creation fails due to RBAC setup failure", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		//
		// Set up account
		//
		signer := jwt.NewSigner("test")
		account, err := models.CreateAccount("test@example.com", "Test User")
		require.NoError(t, err)
		token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		//
		// Initial server and dependencies.
		// Here, we use a mocked auth service that will fail to setup organization.
		//
		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		mockedAuthService := &mockAuthService{
			AuthService:   authService,
			setupOrgError: errors.New("simulated authorization setup failure"),
		}

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		gitProvider := inmemory.NewProvider()
		server, err := NewServer(encryptor, r, signer, oidcProvider, gitProvider, "", "localhost", "", "test", "/app/templates", mockedAuthService, nil, false)
		require.NoError(t, err)

		//
		// Request to create organization returns 500
		//
		body, err := json.Marshal(OrganizationCreationRequest{Name: "Test Organization"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Contains(t, response.Body.String(), "Failed to set up organization roles")

		//
		// Organization and user records to not exist
		//
		_, err = models.FindOrganizationByName("Test Organization")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindAnyUserByEmail(account.Email)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("organization is created successfully", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		//
		// Set up account
		//
		account, err := models.CreateAccount("success@example.com", "Success User")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		//
		// Initial server and dependencies.
		// Here, we use the real authentication service, which should not fail.
		//
		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		gitProvider := inmemory.NewProvider()
		server, err := NewServer(encryptor, r, signer, oidcProvider, gitProvider, "", "localhost", "", "test", "/app/templates", authService, nil, false)
		require.NoError(t, err)

		//
		// Request to create organization should succeed
		//
		body, err := json.Marshal(OrganizationCreationRequest{Name: "Success Organization"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		//
		// Verify organization and user records were created,
		// and RBAC policies were set up for the organization and user.
		//
		var responseData map[string]interface{}
		err = json.Unmarshal(response.Body.Bytes(), &responseData)
		require.NoError(t, err)
		orgID := responseData["id"].(string)

		org, err := models.FindOrganizationByID(orgID)
		require.NoError(t, err)
		assert.Equal(t, "Success Organization", org.Name)

		user, err := models.FindActiveUserByEmail(orgID, account.Email)
		require.NoError(t, err)
		assert.Equal(t, account.Email, user.GetEmail())

		roles, err := authService.GetUserRolesForOrg(context.Background(), user.ID.String(), orgID)
		require.NoError(t, err)
		assert.NotEmpty(t, roles)
	})

	t.Run("organizations with a duplicate name are both created and get distinct slugs", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		account, err := models.CreateAccount("duplicate@example.com", "Duplicate User")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		gitProvider := inmemory.NewProvider()
		server, err := NewServer(encryptor, r, signer, oidcProvider, gitProvider, "", "localhost", "", "test", "/app/templates", authService, nil, false)
		require.NoError(t, err)

		body, err := json.Marshal(OrganizationCreationRequest{Name: "Duplicate Organization"})
		require.NoError(t, err)

		//
		// The first creation succeeds.
		//
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		require.Equal(t, http.StatusOK, response.Code)

		var firstData map[string]interface{}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &firstData))
		firstOrg, err := models.FindOrganizationByID(firstData["id"].(string))
		require.NoError(t, err)

		//
		// Names are no longer required to be unique, so a second organization
		// with the same name is also created. Only the slug must stay unique,
		// so the second organization gets a distinct slug.
		//
		response = execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		require.Equal(t, http.StatusOK, response.Code)

		var secondData map[string]interface{}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &secondData))
		secondOrg, err := models.FindOrganizationByID(secondData["id"].(string))
		require.NoError(t, err)

		assert.Equal(t, "Duplicate Organization", firstOrg.Name)
		assert.Equal(t, "Duplicate Organization", secondOrg.Name)
		assert.NotEqual(t, firstOrg.ID, secondOrg.ID)
		assert.NotEqual(t, firstOrg.Slug, secondOrg.Slug)
	})

	t.Run("organization creation returns 429 when account limit is reached", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		account, err := models.CreateAccount("limited@example.com", "Limited User")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		gitProvider := inmemory.NewProvider()
		usageService := &fakePublicUsageService{
			checkAccountResponse: &usagepb.CheckAccountLimitsResponse{
				Allowed: false,
				Violations: []*usagepb.LimitViolation{
					{
						Limit:           usagepb.LimitName_LIMIT_NAME_MAX_ORGANIZATIONS,
						ConfiguredLimit: 0,
						CurrentValue:    1,
					},
				},
			},
		}
		server, err := NewServer(
			encryptor,
			r,
			signer,
			oidcProvider,
			gitProvider,
			"",
			"localhost",
			"",
			"test",
			"/app/templates",
			authService,
			usageService,
			false,
		)
		require.NoError(t, err)

		body, err := json.Marshal(OrganizationCreationRequest{Name: "Blocked Organization"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "POST",
			path:        "/organizations",
			body:        body,
			authCookie:  token,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusTooManyRequests, response.Code)
		assert.Contains(t, response.Body.String(), "account organization limit exceeded")

		_, err = models.FindOrganizationByName("Blocked Organization")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func Test__GetOrganizationCreationStatus(t *testing.T) {
	t.Run("returns allowed when the account can create another organization", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		account, err := models.CreateAccount("status-ok@example.com", "Status Ok")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		usageService := &fakePublicUsageService{
			checkAccountResponse: &usagepb.CheckAccountLimitsResponse{
				Allowed: true,
				Limits: &usagepb.AccountLimits{
					MaxOrganizations: 3,
				},
			},
		}
		server, err := NewServer(
			encryptor,
			r,
			signer,
			oidcProvider,
			inmemory.NewProvider(),
			"",
			"localhost",
			"",
			"test",
			"/app/templates",
			authService,
			usageService,
			false,
		)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/account/limits",
			authCookie: token,
		})

		require.Equal(t, http.StatusOK, response.Code)

		var data organizationCreationStatusResponse
		err = json.Unmarshal(response.Body.Bytes(), &data)
		require.NoError(t, err)
		assert.True(t, data.Allowed)
		assert.True(t, data.UsageEnabled)
		assert.Equal(t, int32(0), data.CurrentOrganizations)
		assert.Equal(t, int32(3), data.MaxOrganizations)
		assert.Empty(t, data.Message)
	})

	t.Run("returns blocked when the account has reached the max organizations limit", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		account, err := models.CreateAccount("status-blocked@example.com", "Status Blocked")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		usageService := &fakePublicUsageService{
			checkAccountResponse: &usagepb.CheckAccountLimitsResponse{
				Allowed: false,
				Limits: &usagepb.AccountLimits{
					MaxOrganizations: 1,
				},
				Violations: []*usagepb.LimitViolation{
					{
						Limit:           usagepb.LimitName_LIMIT_NAME_MAX_ORGANIZATIONS,
						ConfiguredLimit: 1,
						CurrentValue:    2,
					},
				},
			},
		}
		server, err := NewServer(
			encryptor,
			r,
			signer,
			oidcProvider,
			inmemory.NewProvider(),
			"",
			"localhost",
			"",
			"test",
			"/app/templates",
			authService,
			usageService,
			false,
		)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/account/limits",
			authCookie: token,
		})

		require.Equal(t, http.StatusOK, response.Code)

		var data organizationCreationStatusResponse
		err = json.Unmarshal(response.Body.Bytes(), &data)
		require.NoError(t, err)
		assert.False(t, data.Allowed)
		assert.True(t, data.UsageEnabled)
		assert.Equal(t, int32(1), data.MaxOrganizations)
		assert.Equal(t, "account organization limit exceeded", data.Message)
	})

	t.Run("returns 503 with diagnostic context when the usage service is unavailable", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		account, err := models.CreateAccount("status-unavailable@example.com", "Status Unavailable")
		require.NoError(t, err)
		signer := jwt.NewSigner("test")
		token, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		authService, err := authorization.NewAuthService()
		require.NoError(t, err)

		encryptor := &crypto.NoOpEncryptor{}
		r, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
		require.NoError(t, err)
		oidcProvider := support.NewOIDCProvider()
		usageService := &fakePublicUsageService{
			checkAccountErr: status.Error(codes.Unavailable, "usage service unreachable"),
		}
		server, err := NewServer(
			encryptor,
			r,
			signer,
			oidcProvider,
			inmemory.NewProvider(),
			"",
			"localhost",
			"",
			"test",
			"/app/templates",
			authService,
			usageService,
			false,
		)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/account/limits",
			authCookie: token,
		})

		require.Equal(t, http.StatusServiceUnavailable, response.Code)
		assert.Contains(t, response.Body.String(), "Usage service unavailable")
	})
}
