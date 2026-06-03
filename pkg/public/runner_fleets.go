package public

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
)

type runnerFleetsResponse struct {
	Configured bool           `json:"configured"`
	Fleets     []runner.Fleet `json:"fleets"`
}

func (s *Server) handleRunnerFleets(w http.ResponseWriter, r *http.Request) {
	broker, err := runner.NewBrokerClient(s.registry.HTTPContext())
	if err != nil {
		respondJSON(w, runnerFleetsResponse{
			Configured: false,
			Fleets:     []runner.Fleet{},
		})
		return
	}

	fleets, err := broker.ListFleets()
	if err != nil {
		log.Errorf("runner: failed to list fleets: %v", err)
		http.Error(w, "Failed to list runner fleets", http.StatusBadGateway)
		return
	}

	if fleets == nil {
		fleets = []runner.Fleet{}
	}

	respondJSON(w, runnerFleetsResponse{
		Configured: true,
		Fleets:     fleets,
	})
}
