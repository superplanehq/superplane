package public

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

const (
	defaultPolarWebhookPage  = 1
	defaultPolarWebhookLimit = 50
	maxPolarWebhookLimit     = 100
	polarWebhookPath         = "/api/v1/polar/webhooks"
)

type adminPolarWebhookDelivery struct {
	ID             string `json:"id"`
	CreatedAt      string `json:"created_at"`
	Succeeded      bool   `json:"succeeded"`
	HTTPCode       *int   `json:"http_code"`
	Response       string `json:"response"`
	EventType      string `json:"event_type"`
	EventID        string `json:"event_id"`
	EventSucceeded *bool  `json:"event_succeeded"`
	Payload        string `json:"payload"`
	APIVersion     string `json:"api_version"`
}

type adminPolarWebhookEndpoint struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	APIVersion string `json:"api_version"`
	Format     string `json:"format"`
	Current    bool   `json:"current"`
}

type adminPolarWebhookEndpointsResponse struct {
	Configured bool                        `json:"configured"`
	APIVersion string                      `json:"api_version"`
	Endpoints  []adminPolarWebhookEndpoint `json:"endpoints"`
}

type adminPolarWebhooksResponse struct {
	Configured bool                        `json:"configured"`
	Items      []adminPolarWebhookDelivery `json:"items"`
	Total      int                         `json:"total"`
	Page       int                         `json:"page"`
	Limit      int                         `json:"limit"`
}

func (s *Server) adminListPolarWebhooks(w http.ResponseWriter, r *http.Request) {
	if !polar.Configured() {
		respondJSON(w, adminPolarWebhooksResponse{
			Configured: false,
			Items:      []adminPolarWebhookDelivery{},
			Page:       defaultPolarWebhookPage,
			Limit:      defaultPolarWebhookLimit,
		})
		return
	}

	succeeded, err := parseSucceededQuery(r.URL.Query().Get("succeeded"))
	if err != nil {
		http.Error(w, "Invalid succeeded filter", http.StatusBadRequest)
		return
	}

	page := defaultPolarWebhookPage
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		page = v
	}
	limit := defaultPolarWebhookLimit
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxPolarWebhookLimit {
		limit = maxPolarWebhookLimit
	}

	result, err := polar.NewClientFromEnv().ListWebhookDeliveries(r.Context(), polar.WebhookDeliveryFilter{
		Succeeded: succeeded,
		EventType: strings.TrimSpace(r.URL.Query().Get("event_type")),
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		writePolarAdminError(w, r, err, "Failed to load Polar webhook deliveries")
		return
	}

	items := make([]adminPolarWebhookDelivery, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAdminPolarWebhookDelivery(item))
	}

	respondJSON(w, adminPolarWebhooksResponse{
		Configured: true,
		Items:      items,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
	})
}

func (s *Server) adminListPolarWebhookEndpoints(w http.ResponseWriter, r *http.Request) {
	if !polar.Configured() {
		respondJSON(w, adminPolarWebhookEndpointsResponse{
			Configured: false,
			APIVersion: polar.APIVersion(),
			Endpoints:  []adminPolarWebhookEndpoint{},
		})
		return
	}

	endpoints, err := polar.NewClientFromEnv().ListWebhookEndpoints(r.Context())
	if err != nil {
		writePolarAdminError(w, r, err, "Failed to load Polar webhook endpoints")
		return
	}

	items := make([]adminPolarWebhookEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		items = append(items, adminPolarWebhookEndpoint{
			ID:         endpoint.ID,
			URL:        endpoint.URL,
			APIVersion: endpoint.APIVersion,
			Format:     endpoint.Format,
			Current:    s.isPolarWebhookEndpoint(endpoint.URL),
		})
	}

	respondJSON(w, adminPolarWebhookEndpointsResponse{
		Configured: true,
		APIVersion: polar.APIVersion(),
		Endpoints:  items,
	})
}

func (s *Server) adminRedeliverPolarWebhook(w http.ResponseWriter, r *http.Request) {
	if !polar.Configured() {
		http.Error(w, "Polar is not configured", http.StatusBadRequest)
		return
	}

	eventID := strings.TrimSpace(mux.Vars(r)["eventId"])
	if eventID == "" {
		http.Error(w, "Webhook event id is required", http.StatusBadRequest)
		return
	}

	if err := polar.NewClientFromEnv().RedeliverWebhookEvent(r.Context(), eventID); err != nil {
		writePolarAdminError(w, r, err, "Failed to redeliver Polar webhook event")
		return
	}

	respondJSON(w, map[string]string{"status": "accepted"})
}

// isPolarWebhookEndpoint reports whether a Polar endpoint delivers to this
// installation. One Polar organization can serve several installations.
func (s *Server) isPolarWebhookEndpoint(endpointURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(endpointURL))
	if err != nil || parsed.Host == "" {
		return false
	}
	if strings.TrimRight(parsed.Path, "/") != polarWebhookPath {
		return false
	}
	for _, base := range []string{s.BaseURL, s.WebhooksBaseURL} {
		installation, err := url.Parse(strings.TrimSpace(base))
		if err == nil && installation.Host != "" && strings.EqualFold(installation.Host, parsed.Host) {
			return true
		}
	}
	return false
}

func toAdminPolarWebhookDelivery(item polar.WebhookDelivery) adminPolarWebhookDelivery {
	return adminPolarWebhookDelivery{
		ID:             item.ID,
		CreatedAt:      item.CreatedAt,
		Succeeded:      item.Succeeded,
		HTTPCode:       item.HTTPCode,
		Response:       item.Response,
		EventType:      item.EventType,
		EventID:        item.EventID,
		EventSucceeded: item.EventSucceeded,
		Payload:        item.Payload,
		APIVersion:     item.APIVersion,
	}
}

func parseSucceededQuery(raw string) (*bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func writePolarAdminError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	switch {
	case polar.IsUnauthorized(err):
		recordPolarAdminServerError(r, err)
		http.Error(w, "Polar rejected the access token. Add webhooks:read and webhooks:write scopes.", http.StatusBadGateway)
	case polar.IsUnsupportedAPIVersion(err):
		recordPolarAdminServerError(r, err)
		log.WithError(err).Error(fallback)
		http.Error(w, "Polar does not support API version "+polar.APIVersion()+". Upgrade the pinned Polar API version.", http.StatusBadGateway)
	case polar.IsNotFound(err):
		http.Error(w, "Polar webhook event was not found.", http.StatusNotFound)
	case polar.IsRateLimited(err):
		recordPolarAdminServerError(r, err)
		http.Error(w, "Polar rate-limited the request. Try again later.", http.StatusBadGateway)
	default:
		recordPolarAdminServerError(r, err)
		log.WithError(err).Error(fallback)
		http.Error(w, fallback, http.StatusBadGateway)
	}
}

func recordPolarAdminServerError(r *http.Request, err error) {
	middleware.SetServerError(r.Context(), err, nil)
}
