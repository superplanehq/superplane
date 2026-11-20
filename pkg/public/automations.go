package public

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/grpc"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) registerAutomationRoutes(authenticate mux.MiddlewareFunc, authorizer *authorization.GatewayAuthorizer) {
	service := automations.Service{Backend: factoryactions.CanvasAutomationBackend{Deps: factoryactions.IntakeDependencies{
		Registry: s.registry, Encryptor: s.encryptor, AuthService: s.authService, WebhookBaseURL: s.WebhooksBaseURL,
	}}}
	for _, route := range []authorization.HTTPRoute{
		{Method: http.MethodGet, Pattern: authorization.AutomationsPath},
		{Method: http.MethodGet, Pattern: authorization.AutomationPath},
		{Method: http.MethodPost, Pattern: authorization.ValidateAutomationPath},
		{Method: http.MethodPost, Pattern: authorization.AutomationsPath},
		{Method: http.MethodPut, Pattern: authorization.AutomationPath},
		{Method: http.MethodDelete, Pattern: authorization.AutomationPath},
	} {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, err := authorizer.AuthorizeHTTP(r.Context(), r, route, mux.Vars(r))
			if err != nil {
				writeAutomationError(w, r, err)
				return
			}
			serveAutomation(w, r.WithContext(ctx), route, service)
		})
		s.Router.Handle(route.Pattern, authenticate(s.organizationGatewayHandler(handler))).Methods(route.Method)
	}
}

type automationWriteRequest struct {
	FactoryID string `json:"factory_id"`
	Stage     string `json:"stage"`
	YAML      string `json:"yaml"`
}

func serveAutomation(w http.ResponseWriter, r *http.Request, route authorization.HTTPRoute, service automations.Service) {
	ctx := r.Context()
	orgID, err := uuid.Parse(r.Header.Get("x-organization-id"))
	if err != nil {
		writeAutomationError(w, r, grpcerrors.Unauthenticated(err, "Sign in to an organization."))
		return
	}
	var id uuid.UUID
	if route.Pattern == authorization.AutomationPath {
		id, err = uuid.Parse(mux.Vars(r)["id"])
		if err != nil {
			writeAutomationError(w, r, grpcerrors.InvalidArgument(err, "Enter a valid automation ID."))
			return
		}
	}

	if r.Method == http.MethodGet {
		if route.Pattern == authorization.AutomationPath {
			result, err := service.Read(ctx, orgID, id)
			writeAutomationResponse(w, r, http.StatusOK, result, err)
			return
		}
		factoryID, err := uuid.Parse(r.URL.Query().Get("factory_id"))
		if err != nil {
			writeAutomationError(w, r, grpcerrors.InvalidArgument(err, "Enter a valid factory_id."))
			return
		}
		result, err := service.List(ctx, automations.Placement{OrganizationID: orgID, FactoryID: factoryID})
		writeAutomationResponse(w, r, http.StatusOK, map[string]any{"automations": result}, err)
		return
	}
	if r.Method == http.MethodDelete {
		writeAutomationResponse(w, r, http.StatusNoContent, nil, service.Delete(ctx, orgID, id))
		return
	}

	input, err := decodeAutomationWrite(w, r)
	if err != nil {
		writeAutomationError(w, r, err)
		return
	}
	actor, err := uuid.Parse(r.Header.Get("x-user-id"))
	if err != nil {
		writeAutomationError(w, r, grpcerrors.Unauthenticated(err, "Sign in to an organization."))
		return
	}
	if r.Method == http.MethodPut {
		if input.FactoryID != "" || input.Stage != "" {
			writeAutomationError(w, r, grpcerrors.InvalidArgument(nil, "Omit factory_id and stage when you update an automation."))
			return
		}
		result, err := service.Update(ctx, orgID, id, actor, input.YAML)
		writeAutomationResponse(w, r, http.StatusOK, result, err)
		return
	}

	factoryID, err := uuid.Parse(input.FactoryID)
	if err != nil {
		writeAutomationError(w, r, grpcerrors.InvalidArgument(err, "Enter a valid factory_id."))
		return
	}
	placement := automations.Placement{OrganizationID: orgID, FactoryID: factoryID, Stage: input.Stage}
	if route.Pattern == authorization.ValidateAutomationPath {
		doc, err := service.Validate(ctx, placement, input.YAML)
		if err != nil {
			writeAutomationError(w, r, err)
			return
		}
		_ = writeJSON(w, http.StatusOK, map[string]any{"definition": doc.Definition})
		return
	}

	result, err := service.Create(ctx, placement, actor, input.YAML)
	writeAutomationResponse(w, r, http.StatusCreated, result, err)
}

func decodeAutomationWrite(w http.ResponseWriter, r *http.Request) (automationWriteRequest, error) {
	var input automationWriteRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			return input, grpcerrors.ResourceExhausted(err, "Use a request body smaller than 1 MiB.")
		}
		return input, grpcerrors.InvalidArgument(err, "Send one JSON object with the YAML source as a string.")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return input, grpcerrors.InvalidArgument(err, "Send one JSON object with the YAML source as a string.")
	}
	return input, nil
}

func writeAutomationResponse(w http.ResponseWriter, r *http.Request, code int, result any, err error) {
	if err != nil {
		writeAutomationError(w, r, err)
		return
	}
	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}
	_ = writeJSON(w, code, result)
}

func writeAutomationError(w http.ResponseWriter, r *http.Request, err error) {
	var diagnostic *automationspec.Diagnostic
	if errors.As(err, &diagnostic) {
		_ = writeJSON(w, http.StatusBadRequest, map[string]any{
			"message": diagnostic.Message, "diagnostics": []*automationspec.Diagnostic{diagnostic},
		})
		return
	}
	safe := status.Convert(grpc.SanitizeError(r.Context(), err))
	var fallback *factoryactions.AutomationFormFallback
	if r.Method == http.MethodGet && errors.As(err, &fallback) {
		_ = writeJSON(w, runtime.HTTPStatusFromCode(safe.Code()), map[string]any{
			"message": safe.Message(), "can_edit_in_form": true, "live_version_id": fallback.LiveVersionID,
		})
		return
	}
	if safe.Code() == codes.Internal {
		log.WithError(err).Error("automation API request failed")
	}
	_ = writeJSON(w, runtime.HTTPStatusFromCode(safe.Code()), map[string]string{"message": safe.Message()})
}
