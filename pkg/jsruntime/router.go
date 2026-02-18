package jsruntime

import (
	"net/http"
	"strings"
)

// NewRouter returns an http.Handler that routes JS component API requests.
func NewRouter(handler *APIHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/js-components")
		path = strings.TrimSuffix(path, "/")

		switch {
		case path == "" && r.Method == http.MethodGet:
			handler.ListComponents(w, r)
		case path == "" && r.Method == http.MethodPost:
			handler.SaveComponent(w, r)
		case path == "" && r.Method == http.MethodDelete:
			handler.DeleteComponent(w, r)
		case path == "/generate" && r.Method == http.MethodPost:
			handler.Generate(w, r)
		case path == "/validate" && r.Method == http.MethodPost:
			handler.ValidateSource(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}
