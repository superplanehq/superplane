package hub

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

func (h *Hub) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/register", h.handleRegister)
	mux.HandleFunc("/api/v1/extensions/", h.handleExtensionFile)
	return mux
}

func (h *Hub) handleRegister(w http.ResponseWriter, r *http.Request) {
	workerID := strings.TrimSpace(r.URL.Query().Get(protocol.QueryWorkerID))
	token := strings.TrimSpace(r.URL.Query().Get(protocol.QueryToken))
	registration, err := h.registrationFromToken(workerID, token)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "validate registration token") {
			status = http.StatusUnauthorized
		}
		http.Error(w, err.Error(), status)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("upgrade websocket: %v", err), http.StatusBadRequest)
		return
	}

	h.registerWorker(registration, conn)
}

func claimString(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(text)
}

func (h *Hub) handleExtensionFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get(protocol.QueryToken))
	access, err := h.bundleAccessFromToken(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/extensions/")
	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[1] != "versions" {
		http.NotFound(w, r)
		return
	}

	extensionID := parts[0]
	versionID := parts[2]
	filename := parts[3]

	if access.ExtensionID != extensionID || access.VersionID != versionID {
		http.Error(w, "bundle token does not match requested artifact", http.StatusUnauthorized)
		return
	}

	var (
		content     []byte
		readErr     error
		contentType string
	)

	switch filename {
	case "bundle.js":
		content, readErr = h.storage.ReadVersionBundleJS(access.OrganizationID, extensionID, versionID)
		contentType = "application/javascript"
	case "manifest.json":
		content, readErr = h.storage.ReadVersionManifestJSON(access.OrganizationID, extensionID, versionID)
		contentType = "application/json"
	default:
		http.NotFound(w, r)
		return
	}

	if readErr != nil {
		http.Error(w, readErr.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

func (h *Hub) registrationFromToken(workerID string, token string) (protocol.Registration, error) {
	if h.signer == nil {
		return protocol.Registration{}, fmt.Errorf("hub signer is not configured")
	}
	if workerID == "" {
		return protocol.Registration{}, fmt.Errorf("worker ID is required")
	}
	if token == "" {
		return protocol.Registration{}, fmt.Errorf("registration token is required")
	}

	claims, err := h.signer.ValidateAndGetClaims(token)
	if err != nil {
		return protocol.Registration{}, fmt.Errorf("validate registration token: %w", err)
	}

	registration := protocol.Registration{
		WorkerID:       workerID,
		OrganizationID: claimString(claims, "organizationId"),
		WorkerPoolID:   claimString(claims, "poolId"),
	}
	if err := registration.Validate(); err != nil {
		return protocol.Registration{}, err
	}

	return registration, nil
}

type bundleAccess struct {
	OrganizationID string
	ExtensionID    string
	VersionID      string
}

func (h *Hub) bundleAccessFromToken(token string) (bundleAccess, error) {
	if h.signer == nil {
		return bundleAccess{}, fmt.Errorf("hub signer is not configured")
	}
	if token == "" {
		return bundleAccess{}, fmt.Errorf("bundle token is required")
	}

	claims, err := h.signer.ValidateAndGetClaims(token)
	if err != nil {
		return bundleAccess{}, fmt.Errorf("validate bundle token: %w", err)
	}
	if claimString(claims, "sub") != bundleTokenSubject {
		return bundleAccess{}, fmt.Errorf("invalid bundle token subject")
	}

	access := bundleAccess{
		OrganizationID: claimString(claims, "organizationId"),
		ExtensionID:    claimString(claims, "extensionId"),
		VersionID:      claimString(claims, "versionId"),
	}
	if access.OrganizationID == "" || access.ExtensionID == "" || access.VersionID == "" {
		return bundleAccess{}, fmt.Errorf("bundle token claims are incomplete")
	}

	return access, nil
}
