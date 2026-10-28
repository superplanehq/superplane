package billinghttp

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func (s *Server) adminRegistrationStats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	start, startErr := time.Parse(time.RFC3339Nano, query.Get("start"))
	end, endErr := time.Parse(time.RFC3339Nano, query.Get("end"))
	if startErr != nil || endErr != nil || !start.Before(end) {
		http.Error(w, "Choose a valid start and end date", http.StatusBadRequest)
		return
	}
	offset := 0
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			http.Error(w, "Invalid page offset", http.StatusBadRequest)
			return
		}
		offset = parsed
	}
	var report billingmodels.RegistrationStats
	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		var err error
		report, err = billingmodels.DescribeRegistrationStats(tx, start, end, offset)
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		log.WithError(err).Error("admin: failed to load registration stats")
		http.Error(w, "Failed to load statistics", http.StatusInternalServerError)
		return
	}
	respondJSON(w, report)
}
