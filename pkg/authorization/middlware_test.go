package authorization

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
)

func createTestRequest(method, path string, userID, orgID string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := context.WithValue(req.Context(), userIDKey, userID)
	if orgID != "" {
		ctx = context.WithValue(ctx, orgIDKey, orgID)
	}
	return req.WithContext(ctx)
}

func Test__AuthorizationMiddleware_NonAPIRequests(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	middleware := AuthorizationMiddleware(authService)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name string
		path string
	}{
		{"root path", "/"},
		{"health check", "/health"},
		{"static assets", "/static/file.js"},
		{"non-api path", "/login"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(http.MethodGet, tt.path, r.User.String(), "")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "success", rec.Body.String())
		})
	}
}

func Test__AuthorizationMiddleware_MissingUserID(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	middleware := AuthorizationMiddleware(authService)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called")
	}))

	tests := []struct {
		name    string
		context context.Context
	}{
		{"no userID in context", context.Background()},
		{"empty userID", context.WithValue(context.Background(), userIDKey, "")},
		{"non-string userID", context.WithValue(context.Background(), userIDKey, 123)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil)
			req = req.WithContext(tt.context)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "Unauthorized: missing user ID")
		})
	}
}

func Test__AuthorizationMiddleware_CanvasOperations(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	canvasID := uuid.New().String()
	orgID := uuid.New().String()

	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		method         string
		path           string
		setupUser      func()
		expectedStatus int
	}{
		{
			name:   "GET canvas list - allowed with org viewer role",
			method: http.MethodGet,
			path:   "/api/v1/canvases",
			setupUser: func() {
				err := authService.AssignRole(userID, RoleOrgViewer, orgID, DomainOrg)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST canvas create - allowed with org admin role",
			method: http.MethodPost,
			path:   "/api/v1/canvases",
			setupUser: func() {
				err := authService.AssignRole(userID, RoleOrgAdmin, orgID, DomainOrg)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST canvas create - forbidden with viewer role",
			method: http.MethodPost,
			path:   "/api/v1/canvases",
			setupUser: func() {
				err := authService.RemoveRole(userID, RoleOrgAdmin, orgID, DomainOrg)
				require.NoError(t, err)
				err = authService.AssignRole(userID, RoleOrgViewer, orgID, DomainOrg)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "GET canvas detail - allowed with org viewer role",
			method: http.MethodGet,
			path:   fmt.Sprintf("/api/v1/canvases/%s", canvasID),
			setupUser: func() {
				// Already has viewer role from previous test
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "DELETE canvas - forbidden without proper role",
			method: http.MethodDelete,
			path:   "/api/v1/canvases",
			setupUser: func() {
				// User only has viewer role
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupUser()

			req := createTestRequest(tt.method, tt.path, userID, orgID)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func Test__AuthorizationMiddleware_EventSourceOperations(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	canvasID := uuid.New().String()
	eventSourceID := uuid.New().String()

	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		method         string
		path           string
		setupUser      func()
		expectedStatus int
	}{
		{
			name:   "GET event sources list - allowed with viewer role",
			method: http.MethodGet,
			path:   fmt.Sprintf("/api/v1/canvases/%s/event-sources", canvasID),
			setupUser: func() {
				err := authService.AssignRole(userID, RoleCanvasViewer, canvasID, DomainCanvas)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST create event source - forbidden with viewer role",
			method: http.MethodPost,
			path:   fmt.Sprintf("/api/v1/canvases/%s/event-sources", canvasID),
			setupUser: func() {
				// User only has viewer role from previous test
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "POST create event source - allowed with admin role",
			method: http.MethodPost,
			path:   fmt.Sprintf("/api/v1/canvases/%s/event-sources", canvasID),
			setupUser: func() {
				err := authService.AssignRole(userID, RoleCanvasAdmin, canvasID, DomainCanvas)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "PUT update event source - allowed with admin role",
			method: http.MethodPut,
			path:   fmt.Sprintf("/api/v1/canvases/%s/event-sources/%s", canvasID, eventSourceID),
			setupUser: func() {
				// User already has admin role
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "DELETE event source - allowed with owner role",
			method: http.MethodDelete,
			path:   fmt.Sprintf("/api/v1/canvases/%s/event-sources/%s", canvasID, eventSourceID),
			setupUser: func() {
				err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, DomainCanvas)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupUser()

			req := createTestRequest(tt.method, tt.path, userID, "")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func Test__AuthorizationMiddleware_StageOperations(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	canvasID := uuid.New().String()
	stageID := uuid.New().String()

	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	viewerID := uuid.New().String()
	adminID := uuid.New().String()

	tests := []struct {
		name           string
		method         string
		path           string
		userID         string
		setupUser      func()
		expectedStatus int
	}{
		{
			name:   "GET stages list - allowed with viewer role",
			method: http.MethodGet,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages", canvasID),
			userID: viewerID,
			setupUser: func() {
				err := authService.AssignRole(viewerID, RoleCanvasViewer, canvasID, DomainCanvas)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST create stage - forbidden with viewer role",
			method: http.MethodPost,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages", canvasID),
			userID: viewerID,
			setupUser: func() {
				// User already has viewer role
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "POST create stage - allowed with admin role",
			method: http.MethodPost,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages", canvasID),
			userID: adminID,
			setupUser: func() {
				err := authService.AssignRole(adminID, RoleCanvasAdmin, canvasID, DomainCanvas)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "PATCH update stage - allowed with admin role",
			method: http.MethodPatch,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages/%s", canvasID, stageID),
			userID: adminID,
			setupUser: func() {
				// User already has admin role
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "DELETE stage - allowed with admin role",
			method: http.MethodDelete,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages/%s", canvasID, stageID),
			userID: adminID,
			setupUser: func() {
				// User already has admin role
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupUser()

			req := createTestRequest(tt.method, tt.path, tt.userID, "")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func Test__AuthorizationMiddleware_StageEventOperations(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	canvasID := uuid.New().String()
	stageID := uuid.New().String()
	eventID := uuid.New().String()

	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		method         string
		path           string
		setupUser      func()
		expectedStatus int
	}{
		{
			name:   "GET stage events - allowed with viewer role",
			method: http.MethodGet,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages/%s/events", canvasID, stageID),
			setupUser: func() {
				err := authService.AssignRole(userID, RoleCanvasViewer, canvasID, DomainCanvas)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST approve stage event - forbidden with viewer role",
			method: http.MethodPost,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages/%s/events/%s/approve", canvasID, stageID, eventID),
			setupUser: func() {
				// User only has viewer role
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "POST approve stage event - allowed with admin role",
			method: http.MethodPost,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages/%s/events/%s/approve", canvasID, stageID, eventID),
			setupUser: func() {
				err := authService.AssignRole(userID, RoleCanvasAdmin, canvasID, DomainCanvas)
				require.NoError(t, err)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "GET approve endpoint (wrong method) - uses read permission",
			method: http.MethodGet,
			path:   fmt.Sprintf("/api/v1/canvases/%s/stages/%s/events/%s/approve", canvasID, stageID, eventID),
			setupUser: func() {
				// User has admin role which inherits viewer permissions
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupUser()

			req := createTestRequest(tt.method, tt.path, userID, "")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func Test__AuthorizationMiddleware_MissingOrgID(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// test org-scoped endpoints without orgID in context
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "GET canvas list without orgID",
			method: http.MethodGet,
			path:   "/api/v1/canvases",
		},
		{
			name:   "POST create canvas without orgID",
			method: http.MethodPost,
			path:   "/api/v1/canvases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(tt.method, tt.path, userID, "") // Empty orgID
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "Unauthorized: organization ID required")
		})
	}
}

func Test__AuthorizationMiddleware_UnrecognizedPaths(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	orgID := uuid.New().String()

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("passed through"))
	}))

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "unknown resource type",
			method: http.MethodGet,
			path:   "/api/v1/unknown/resource",
		},
		{
			name:   "invalid path structure",
			method: http.MethodGet,
			path:   "/api/v1/canvases/abc/unknown/xyz",
		},
		{
			name:   "deeply nested unknown path",
			method: http.MethodPost,
			path:   "/api/v1/canvases/abc/stages/def/unknown/ghi/jkl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(tt.method, tt.path, userID, orgID)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "passed through", rec.Body.String())
		})
	}
}

func Test__AuthorizationMiddleware_RoleInheritance(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	canvasID := uuid.New().String()
	orgID := uuid.New().String()

	// Setup roles
	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("canvas owner can perform all actions in its canvas", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasOwner, canvasID, DomainCanvas)
		require.NoError(t, err)

		tests := []struct {
			method string
			path   string
		}{
			{http.MethodGet, fmt.Sprintf("/api/v1/canvases/%s/stages", canvasID)},                         // read (viewer)
			{http.MethodPost, fmt.Sprintf("/api/v1/canvases/%s/stages", canvasID)},                        // create (admin)
			{http.MethodPut, fmt.Sprintf("/api/v1/canvases/%s/stages/123", canvasID)},                     // update (admin)
			{http.MethodDelete, fmt.Sprintf("/api/v1/canvases/%s/stages/123", canvasID)},                  // delete (admin)
			{http.MethodPost, fmt.Sprintf("/api/v1/canvases/%s/stages/123/events/456/approve", canvasID)}, // approve (admin)
		}

		for _, test := range tests {
			req := createTestRequest(test.method, test.path, userID, "")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "Owner should be able to %s %s", test.method, test.path)
		}
	})

	t.Run("canvas owner cannot perform actions on other canvas that has no permission", func(t *testing.T) {
		randomCanvasID := uuid.New().String()

		tests := []struct {
			method string
			path   string
		}{
			{http.MethodGet, fmt.Sprintf("/api/v1/canvases/%s/stages", randomCanvasID)},                         // read (viewer)
			{http.MethodPost, fmt.Sprintf("/api/v1/canvases/%s/stages", randomCanvasID)},                        // create (admin)
			{http.MethodPut, fmt.Sprintf("/api/v1/canvases/%s/stages/123", randomCanvasID)},                     // update (admin)
			{http.MethodDelete, fmt.Sprintf("/api/v1/canvases/%s/stages/123", randomCanvasID)},                  // delete (admin)
			{http.MethodPost, fmt.Sprintf("/api/v1/canvases/%s/stages/123/events/456/approve", randomCanvasID)}, // approve (admin)
		}

		for _, test := range tests {
			req := createTestRequest(test.method, test.path, userID, "")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code, "Owner should not be able to %s %s", test.method, test.path)
		}
	})

	t.Run("org owner can perform all org actions", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleOrgOwner, orgID, DomainOrg)
		require.NoError(t, err)

		tests := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/v1/canvases"},    // read (viewer)
			{http.MethodPost, "/api/v1/canvases"},   // create (admin)
			{http.MethodPut, "/api/v1/canvases"},    // update (admin)
			{http.MethodDelete, "/api/v1/canvases"}, // delete (admin)
		}

		for _, test := range tests {
			req := createTestRequest(test.method, test.path, userID, orgID)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "Org owner should be able to %s %s", test.method, test.path)
		}
	})
}

func Test__AuthorizationMiddleware_ConcurrentRequests(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	numUsers := 5
	users := make([]string, numUsers)
	canvases := make([]string, numUsers)

	for i := 0; i < numUsers; i++ {
		users[i] = uuid.New().String()
		canvases[i] = uuid.New().String()

		err := authService.SetupCanvasRoles(canvases[i])
		require.NoError(t, err)

		if i%2 == 0 {
			err = authService.AssignRole(users[i], RoleCanvasAdmin, canvases[i], DomainCanvas)
		} else {
			err = authService.AssignRole(users[i], RoleCanvasViewer, canvases[i], DomainCanvas)
		}
		require.NoError(t, err)
	}

	// run concurrent requests
	done := make(chan bool)
	for i := 0; i < numUsers; i++ {
		go func(idx int) {
			defer func() { done <- true }()

			// admin users can create, viewers cannot
			method := http.MethodPost
			expectedStatus := http.StatusOK
			if idx%2 != 0 {
				expectedStatus = http.StatusForbidden
			}

			path := fmt.Sprintf("/api/v1/canvases/%s/stages", canvases[idx])
			req := createTestRequest(method, path, users[idx], "")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, expectedStatus, rec.Code)
		}(i)
	}

	// wait for all goroutines to complete
	for i := 0; i < numUsers; i++ {
		<-done
	}
}

func Test__parseAPIPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		method       string
		wantResource string
		wantAction   string
		wantCanvasID string
		wantDomain   string
		wantErr      bool
	}{
		// Canvas operations
		{
			name:         "canvas list",
			path:         "/api/v1/canvases",
			method:       http.MethodGet,
			wantResource: "canvas",
			wantAction:   "read",
			wantDomain:   DomainOrg,
		},
		{
			name:         "canvas create",
			path:         "/api/v1/canvases",
			method:       http.MethodPost,
			wantResource: "canvas",
			wantAction:   "create",
			wantDomain:   DomainOrg,
		},
		{
			name:         "canvas detail",
			path:         "/api/v1/canvases/abc123",
			method:       http.MethodGet,
			wantResource: "canvas",
			wantAction:   "read",
			wantCanvasID: "abc123",
			wantDomain:   DomainOrg,
		},
		// Event source operations
		{
			name:         "event source list",
			path:         "/api/v1/canvases/abc123/event-sources",
			method:       http.MethodGet,
			wantResource: "eventsource",
			wantAction:   "read",
			wantCanvasID: "abc123",
		},
		{
			name:         "event source detail",
			path:         "/api/v1/canvases/abc123/event-sources/es456",
			method:       http.MethodGet,
			wantResource: "eventsource",
			wantAction:   "read",
			wantCanvasID: "abc123",
		},
		{
			name:         "event source create",
			path:         "/api/v1/canvases/abc123/event-sources",
			method:       http.MethodPost,
			wantResource: "eventsource",
			wantAction:   "create",
			wantCanvasID: "abc123",
		},
		{
			name:         "event source update",
			path:         "/api/v1/canvases/abc123/event-sources/es456",
			method:       http.MethodPut,
			wantResource: "eventsource",
			wantAction:   "update",
			wantCanvasID: "abc123",
		},
		{
			name:         "event source delete",
			path:         "/api/v1/canvases/abc123/event-sources/es456",
			method:       http.MethodDelete,
			wantResource: "eventsource",
			wantAction:   "delete",
			wantCanvasID: "abc123",
		},
		// Stage operations
		{
			name:         "stage list",
			path:         "/api/v1/canvases/abc123/stages",
			method:       http.MethodGet,
			wantResource: "stage",
			wantAction:   "read",
			wantCanvasID: "abc123",
		},
		{
			name:         "stage create",
			path:         "/api/v1/canvases/abc123/stages",
			method:       http.MethodPost,
			wantResource: "stage",
			wantAction:   "create",
			wantCanvasID: "abc123",
		},
		{
			name:         "stage update with PUT",
			path:         "/api/v1/canvases/abc123/stages/stage789",
			method:       http.MethodPut,
			wantResource: "stage",
			wantAction:   "update",
			wantCanvasID: "abc123",
		},
		{
			name:         "stage update with PATCH",
			path:         "/api/v1/canvases/abc123/stages/stage789",
			method:       http.MethodPatch,
			wantResource: "stage",
			wantAction:   "update",
			wantCanvasID: "abc123",
		},
		{
			name:         "stage delete",
			path:         "/api/v1/canvases/abc123/stages/stage789",
			method:       http.MethodDelete,
			wantResource: "stage",
			wantAction:   "delete",
			wantCanvasID: "abc123",
		},
		// Stage event operations
		{
			name:         "stage events list",
			path:         "/api/v1/canvases/abc123/stages/stage789/events",
			method:       http.MethodGet,
			wantResource: "stageevent",
			wantAction:   "read",
			wantCanvasID: "abc123",
		},
		{
			name:         "stage events create",
			path:         "/api/v1/canvases/abc123/stages/stage789/events",
			method:       http.MethodPost,
			wantResource: "stageevent",
			wantAction:   "create",
			wantCanvasID: "abc123",
		},
		{
			name:         "stage event approve",
			path:         "/api/v1/canvases/abc123/stages/stage789/events/evt123/approve",
			method:       http.MethodPost,
			wantResource: "stageevent",
			wantAction:   "approve",
			wantCanvasID: "abc123",
		},
		// Error cases
		{
			name:    "unrecognized path",
			path:    "/api/v1/unknown/path",
			method:  http.MethodGet,
			wantErr: true,
		},
		{
			name:    "invalid path structure",
			path:    "/api/v1/canvases/abc/unknown/xyz",
			method:  http.MethodGet,
			wantErr: true,
		},
		{
			name:    "missing api prefix",
			path:    "/canvases/abc123",
			method:  http.MethodGet,
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			method:  http.MethodGet,
			wantErr: true,
		},
		{
			name:    "just api prefix",
			path:    "/api/v1/",
			method:  http.MethodGet,
			wantErr: true,
		},
		// Edge cases with special characters
		{
			name:         "canvas ID with hyphens",
			path:         "/api/v1/canvases/abc-123-def/stages",
			method:       http.MethodGet,
			wantResource: "stage",
			wantAction:   "read",
			wantCanvasID: "abc-123-def",
		},
		{
			name:         "canvas ID with underscores",
			path:         "/api/v1/canvases/abc_123_def/event-sources",
			method:       http.MethodGet,
			wantResource: "eventsource",
			wantAction:   "read",
			wantCanvasID: "abc_123_def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseAPIPath(tt.path, tt.method)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, info)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, info)

			assert.Equal(t, tt.wantResource, info.resourceType)
			assert.Equal(t, tt.wantAction, info.action)
			assert.Equal(t, tt.wantCanvasID, info.canvasID)
			if tt.wantDomain != "" {
				assert.Equal(t, tt.wantDomain, info.domainType)
			}
		})
	}
}

func Test__AuthorizationMiddleware_GroupPermissions(t *testing.T) {
	_ = support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	orgID := uuid.New().String()
	userID := uuid.New().String()
	groupName := "test-admins"

	// Setup organization and create group
	err = authService.SetupOrganizationRoles(orgID)
	require.NoError(t, err)
	err = authService.CreateGroup(orgID, groupName, RoleOrgAdmin)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("user without group cannot create canvas", func(t *testing.T) {
		req := createTestRequest(http.MethodPost, "/api/v1/canvases", userID, orgID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("user in admin group can create canvas", func(t *testing.T) {
		err := authService.AddUserToGroup(orgID, userID, groupName)
		require.NoError(t, err)

		req := createTestRequest(http.MethodPost, "/api/v1/canvases", userID, orgID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("removing user from group revokes permissions", func(t *testing.T) {
		err := authService.RemoveUserFromGroup(orgID, userID, groupName)
		require.NoError(t, err)

		req := createTestRequest(http.MethodPost, "/api/v1/canvases", userID, orgID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func Test__AuthorizationMiddleware_ComplexScenarios(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	org1 := uuid.New().String()
	org2 := uuid.New().String()
	canvas1 := uuid.New().String()
	canvas2 := uuid.New().String()
	userID := r.User.String()

	err = authService.SetupOrganizationRoles(org1)
	require.NoError(t, err)
	err = authService.SetupOrganizationRoles(org2)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvas1)
	require.NoError(t, err)
	err = authService.SetupCanvasRoles(canvas2)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("user with different roles in different orgs", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleOrgAdmin, org1, DomainOrg)
		require.NoError(t, err)
		err = authService.AssignRole(userID, RoleOrgViewer, org2, DomainOrg)
		require.NoError(t, err)

		req := createTestRequest(http.MethodPost, "/api/v1/canvases", userID, org1)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		req = createTestRequest(http.MethodPost, "/api/v1/canvases", userID, org2)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("user with different roles in different canvases", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasOwner, canvas1, DomainCanvas)
		require.NoError(t, err)
		err = authService.AssignRole(userID, RoleCanvasViewer, canvas2, DomainCanvas)
		require.NoError(t, err)

		req := createTestRequest(http.MethodDelete, fmt.Sprintf("/api/v1/canvases/%s/stages/123", canvas1), userID, "")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		req = createTestRequest(http.MethodDelete, fmt.Sprintf("/api/v1/canvases/%s/stages/123", canvas2), userID, "")
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func Test__AuthorizationMiddleware_SpecialCases(t *testing.T) {
	r := support.Setup(t)

	authService, err := NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	userID := r.User.String()
	canvasID := uuid.New().String()

	err = authService.SetupCanvasRoles(canvasID)
	require.NoError(t, err)

	middleware := AuthorizationMiddleware(authService)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("paths with trailing slashes", func(t *testing.T) {
		err := authService.AssignRole(userID, RoleCanvasViewer, canvasID, DomainCanvas)
		require.NoError(t, err)

		paths := []string{
			fmt.Sprintf("/api/v1/canvases/%s/stages/", canvasID),
			fmt.Sprintf("/api/v1/canvases/%s/event-sources/", canvasID),
		}

		for _, path := range paths {
			req := createTestRequest(http.MethodGet, path, userID, "")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code, "Path with trailing slash should work: %s", path)
		}
	})

	t.Run("paths with query parameters", func(t *testing.T) {
		req := createTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/canvases/%s/stages?filter=active&sort=name", canvasID), userID, "")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "Path with query parameters should work")
	})

	t.Run("paths with URL fragments", func(t *testing.T) {
		req := createTestRequest(http.MethodGet, fmt.Sprintf("/api/v1/canvases/%s/stages#section", canvasID), userID, "")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "Path with URL fragment should work")
	})
}

func Test__resourceInfo_String(t *testing.T) {
	tests := []struct {
		name     string
		info     resourceInfo
		expected string
	}{
		{
			name: "full resource info",
			info: resourceInfo{
				resourceType: "stage",
				action:       "create",
				canvasID:     "canvas-123",
				orgID:        "org-456",
			},
			expected: "resourceType: stage, action: create, canvasID: canvas-123, orgID: org-456",
		},
		{
			name: "partial resource info",
			info: resourceInfo{
				resourceType: "canvas",
				action:       "read",
				orgID:        "org-789",
			},
			expected: "resourceType: canvas, action: read, canvasID: , orgID: org-789",
		},
		{
			name:     "empty resource info",
			info:     resourceInfo{},
			expected: "resourceType: , action: , canvasID: , orgID: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.info.String())
		})
	}
}

func Test__getActionFromMethod(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{http.MethodGet, "read"},
		{http.MethodPost, "create"},
		{http.MethodPut, "update"},
		{http.MethodPatch, "update"},
		{http.MethodDelete, "delete"},
		{http.MethodOptions, ""},
		{http.MethodHead, ""},
		{http.MethodConnect, ""},
		{http.MethodTrace, ""},
		{"INVALID", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := getActionFromMethod(tt.method)
			assert.Equal(t, tt.want, got)
		})
	}
}
