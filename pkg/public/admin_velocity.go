package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
)

func (s *Server) adminGetOrganizationVelocity(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseAdminOrgID(w, r)
	if !ok {
		return
	}

	report, err := factoryactions.DescribeAdminOrganizationVelocity(
		r.Context(),
		orgID,
		r.URL.Query().Get("factory_id"),
		adminVelocityPeriodDays(r.URL.Query().Get("period_days")),
	)
	if err != nil {
		writeAdminVelocityError(w, err)
		return
	}

	body, err := json.Marshal(report)
	if err != nil {
		log.Errorf("admin: failed to encode organization velocity: %v", err)
		http.Error(w, "Failed to load organization velocity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func adminVelocityPeriodDays(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	days, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return days
}

func writeAdminVelocityError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrFactoryNotFound) {
		http.Error(w, "Workspace not found", http.StatusNotFound)
		return
	}

	log.Errorf("admin: failed to load organization velocity: %v", err)
	http.Error(w, "Failed to load organization velocity", http.StatusInternalServerError)
}
