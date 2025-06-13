package authorization

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	canvasListPattern        = regexp.MustCompile(`^canvases$`)
	canvasDetailPattern      = regexp.MustCompile(`^canvases/([^/]+)$`)
	eventSourcePattern       = regexp.MustCompile(`^canvases/([^/]+)/event-sources(?:/[^/]+)?$`)
	stagePattern             = regexp.MustCompile(`^canvases/([^/]+)/stages(?:/[^/]+)?$`)
	stageEventPattern        = regexp.MustCompile(`^canvases/([^/]+)/stages/[^/]+/events$`)
	stageEventApprovePattern = regexp.MustCompile(`^canvases/([^/]+)/stages/[^/]+/events/[^/]+/approve$`)
)

// define custom types to avoid context key warnings
type contextKey string

const (
	userIDKey contextKey = "userID"
	orgIDKey  contextKey = "orgID"
)

type resourceInfo struct {
	resourceType string // "canvas", "eventsource", "stage", "stageevent"
	action       string // "create", "read", "update", "delete", "approve"
	canvasID     string // Canvas ID for canvas-scoped resources
	orgID        string // Organization ID for org-scoped resources
	domainType   string // "canvas" or "org"
}

func (r *resourceInfo) String() string {
	return fmt.Sprintf("resourceType: %s, action: %s, canvasID: %s, orgID: %s", r.resourceType, r.action, r.canvasID, r.orgID)
}

func (r *resourceInfo) SetOrgID(orgID string) {
	r.orgID = orgID
}

func getActionFromMethod(method string) string {
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return ""
	}
}

func parseAPIPath(path string, method string) (*resourceInfo, error) {
	path = strings.TrimPrefix(path, "/api/v1/")

	switch {
	case canvasListPattern.MatchString(path):
		return &resourceInfo{
			resourceType: "canvas",
			action:       getActionFromMethod(method),
			domainType:   DomainOrg,
		}, nil

	case canvasDetailPattern.MatchString(path):
		matches := canvasDetailPattern.FindStringSubmatch(path)
		return &resourceInfo{
			resourceType: "canvas",
			action:       "read",
			canvasID:     matches[1],
			domainType:   DomainOrg,
		}, nil

	case eventSourcePattern.MatchString(path):
		matches := eventSourcePattern.FindStringSubmatch(path)
		return &resourceInfo{
			resourceType: "eventsource",
			action:       getActionFromMethod(method),
			canvasID:     matches[1],
		}, nil

	case stageEventApprovePattern.MatchString(path):
		matches := stageEventApprovePattern.FindStringSubmatch(path)
		if method == http.MethodPost {
			return &resourceInfo{
				resourceType: "stageevent",
				action:       "approve",
				canvasID:     matches[1],
			}, nil
		}

	case stageEventPattern.MatchString(path):
		matches := stageEventPattern.FindStringSubmatch(path)
		return &resourceInfo{
			resourceType: "stageevent",
			action:       getActionFromMethod(method),
			canvasID:     matches[1],
		}, nil

	case stagePattern.MatchString(path):
		matches := stagePattern.FindStringSubmatch(path)
		return &resourceInfo{
			resourceType: "stage",
			action:       getActionFromMethod(method),
			canvasID:     matches[1],
		}, nil
	}

	return nil, fmt.Errorf("unrecognized API path: %s", path)
}

func checkPermission(authService Authorization, userID string, info *resourceInfo) (bool, error) {
	if info.domainType == DomainOrg {
		if info.orgID == "" {
			return false, fmt.Errorf("organization ID required")
		}
		return authService.CheckOrganizationPermission(userID, info.orgID, info.resourceType, info.action)
	}

	if info.canvasID == "" {
		return false, fmt.Errorf("canvas ID required")
	}
	return authService.CheckCanvasPermission(userID, info.canvasID, info.resourceType, info.action)
}

// AuthorizationMiddleware returns an HTTP middleware that checks authorization
func AuthorizationMiddleware(authService Authorization) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			userID, ok := ctx.Value(userIDKey).(string)
			if !ok || userID == "" {
				http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
				return
			}

			orgID, _ := ctx.Value(orgIDKey).(string)

			info, err := parseAPIPath(r.URL.Path, r.Method)
			if err != nil {
				log.Warnf("Failed to parse API path %s: %v", r.URL.Path, err)
				next.ServeHTTP(w, r)
				return
			}

			info.SetOrgID(orgID)

			allowed, err := checkPermission(authService, userID, info)
			if err != nil {
				if err.Error() == "organization ID required" || err.Error() == "canvas ID required" {
					http.Error(w, fmt.Sprintf("Unauthorized: %s", err.Error()), http.StatusUnauthorized)
					return
				}
				log.Errorf("Error checking permission: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
