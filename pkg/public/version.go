package public

import (
	"encoding/json"
	"net/http"

	"github.com/superplanehq/superplane/pkg/buildinfo"
)

type versionResponse struct {
	Version string `json:"version"`
}

// serveVersion reports the server release version. It is public and
// unauthenticated so that clients (mainly the CLI) can detect version skew
// against self-hosted installations before or after authentication.
func serveVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(versionResponse{Version: buildinfo.Version})
}
