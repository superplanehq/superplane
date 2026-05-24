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
type Server struct {
	Log         *slog.Logger
	EC2Launcher *ec2provision.Launcher
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) adminManagedRunners(w http.ResponseWriter, r *http.Request) {
	if s.EC2Launcher == nil {
		writeError(w, http.StatusServiceUnavailable, "ec2 diagnostics unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	rows, err := s.EC2Launcher.ListManagedRunners(ctx)
	if err != nil {
		if s.Log != nil {
			s.Log.Warn("admin managed runners", slog.Any("err", err))
		}
		writeError(w, http.StatusBadGateway, "could not list instances")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) adminEc2Console(w http.ResponseWriter, r *http.Request) {
	if s.EC2Launcher == nil {
		writeError(w, http.StatusServiceUnavailable, "ec2 diagnostics unavailable")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("instance_id"))
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	out, err := s.EC2Launcher.ManagedInstanceConsoleOutput(ctx, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}
