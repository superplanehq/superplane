package fleetmanager

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/superplane/runner/fleet-manager/internal/ec2provision"
)

// Server exposes fleet-manager HTTP handlers (EC2 provisioning diagnostics only).
//
// EC2Launchers is the per-pool slice owned by this fleet-manager process. With one
// pool today it is length 1; multi-pool deploys (multi-arch / size class / etc.) push
// each additional pool onto the slice and the same handlers fan out across them.
type Server struct {
	Log          *slog.Logger
	EC2Launchers []*ec2provision.Launcher
}

// managedRunnerLister is the subset of *ec2provision.Launcher that aggregateManagedRunners
// uses. Defined here (not in ec2provision) so the handler tests can substitute fakes.
type managedRunnerLister interface {
	ListManagedRunners(ctx context.Context) ([]ec2provision.ManagedRunnerSummary, error)
	FleetID() string
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) adminManagedRunners(w http.ResponseWriter, r *http.Request) {
	if len(s.EC2Launchers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "ec2 diagnostics unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	views := make([]managedRunnerLister, len(s.EC2Launchers))
	for i, l := range s.EC2Launchers {
		views[i] = l
	}
	rows := aggregateManagedRunners(ctx, s.Log, views)
	writeJSON(w, http.StatusOK, rows)
}

// aggregateManagedRunners concatenates ListManagedRunners output across launchers. Per-
// launcher failures are logged with the owning fleet id and skipped — admin listing
// should still succeed for the healthy pools when one pool's EC2 call hiccups.
func aggregateManagedRunners(ctx context.Context, log *slog.Logger, launchers []managedRunnerLister) []ec2provision.ManagedRunnerSummary {
	out := make([]ec2provision.ManagedRunnerSummary, 0)
	for _, l := range launchers {
		rows, err := l.ListManagedRunners(ctx)
		if err != nil {
			if log != nil {
				log.Warn("admin managed runners: launcher list failed",
					slog.String("fleet_id", l.FleetID()),
					slog.Any("err", err))
			}
			continue
		}
		out = append(out, rows...)
	}
	return out
}

func (s *Server) adminEc2Console(w http.ResponseWriter, r *http.Request) {
	if len(s.EC2Launchers) == 0 {
		writeError(w, http.StatusServiceUnavailable, "ec2 diagnostics unavailable")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("instance_id"))
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	// GetConsoleOutput takes an instance id and the EC2 client is region-scoped, so any
	// pool's launcher in this FM process can serve any managed instance's console.
	// We use the first one; the inner DescribeInstances tag check still ensures the
	// instance is one of ours (TagKeyManaged), regardless of which pool owns it.
	out, err := s.EC2Launchers[0].ManagedInstanceConsoleOutput(ctx, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}
