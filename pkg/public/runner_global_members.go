package public

import (
	"cmp"
	"net/http"
	"slices"
	"strings"

	log "github.com/sirupsen/logrus"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/taskowner"
	"github.com/superplanehq/superplane/pkg/models"
)

func (s *Server) handleRunnerListMembers(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeMembersRead)
	if !ok {
		return
	}
	candidates, err := models.ListTaskOwnerCandidates(database.DB(r.Context()), scope.OrganizationID, scope.FactoryID)
	if err != nil {
		log.WithError(err).Error("list runner members")
		http.Error(w, "Members request failed", http.StatusInternalServerError)
		return
	}
	s.addTaskOwnerRoles(r.Context(), scope.OrganizationID, candidates)

	members := taskowner.Members(candidates)
	slices.SortStableFunc(members, func(left, right taskowner.Member) int {
		return cmp.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	_ = writeJSON(w, http.StatusOK, map[string]any{"members": members})
}
