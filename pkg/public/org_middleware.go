package public

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

var orgIDKey contextKey = "org-id"

func OrganizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Header.Get("x-semaphore-org-id")
		if orgID == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		organizationID, err := uuid.Parse(orgID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		//
		// TODO
		// Check if organization exists and if it is not suspended/blocked before proceeding
		// or ensure that happens on the auth component.
		//

		ctx := context.WithValue(r.Context(), orgIDKey, organizationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
