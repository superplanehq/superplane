package public

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/badges"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
)

func (s *Server) handlePublicBadge(w http.ResponseWriter, r *http.Request) {
	token := badgeToken(mux.Vars(r)["token"])
	factory, err := models.FindFactoryByPublicBadgeToken(database.DB(r.Context()), token)
	if err != nil || factory == nil || !factory.PublicBadgeEnabled {
		if err != nil && !errors.Is(err, models.ErrFactoryNotFound) {
			log.Errorf("Failed to load public badge: %v", err)
		}
		http.NotFound(w, r)
		return
	}

	period := badgePeriodDays(r.URL.Query().Get("period"))
	size := badgeSize(r.URL.Query().Get("size"))
	now := time.Now().In(time.Local)
	windows, err := factoryactions.ComputeFactoryVelocityWindows(
		database.DB(r.Context()),
		factory,
		period,
		factory.OnboardingConfigValue().AppRepository,
		now,
	)
	if err != nil {
		log.Errorf("Failed to build public badge: %v", err)
		http.Error(w, "Badge unavailable", http.StatusInternalServerError)
		return
	}

	body, err := badges.Render(badgeInput(windows, period, size, now, factory.PublicBadgeShowCost))
	if err != nil {
		log.Errorf("Failed to render public badge: %v", err)
		http.Error(w, "Badge unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func badgeInput(
	windows factoryactions.FactoryVelocityWindows,
	period int,
	size badges.Size,
	now time.Time,
	showCost bool,
) badges.Input {
	days := make([]badges.DayPoint, len(windows.Days))
	for i, day := range windows.Days {
		days[i] = badges.DayPoint{
			Date:             day.Date,
			SuperplaneMerged: day.SuperplaneMerged,
			PeopleMerged:     day.PeopleMerged,
		}
	}
	in := badges.Input{
		PeriodDays:               period,
		SuperplaneMerged:         windows.Current.SuperplaneMerged,
		PeopleMerged:             windows.Current.PeopleMerged,
		SuperplaneWaste:          windows.Current.Waste,
		PreviousSuperplaneMerged: windows.Previous.SuperplaneMerged,
		PreviousPeopleMerged:     windows.Previous.PeopleMerged,
		HasPrevious:              windows.HasPrevious,
		Days:                     days,
		Updated:                  now,
		Size:                     size,
	}
	if showCost {
		cents := windows.Current.CostCents
		in.CostCents = &cents
	}
	return in
}

func badgeToken(raw string) string {
	return strings.TrimSuffix(strings.TrimSpace(raw), ".svg")
}

func badgePeriodDays(raw string) int {
	switch strings.TrimSpace(raw) {
	case "7":
		return 7
	case "14":
		return 14
	default:
		return 30
	}
}

func badgeSize(raw string) badges.Size {
	switch strings.TrimSpace(raw) {
	case "large":
		return badges.SizeLarge
	case "wide":
		return badges.SizeWide
	default:
		return badges.SizeSmall
	}
}
