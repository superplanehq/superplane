package public

import (
	"net/http"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
)

// The flow tools of the global runner MCP server reuse the legacy flow
// handlers. The runner token replaces the flow token, and its scope must
// allow the tool.

func (s *Server) handleRunnerGlobalPlanUpdate(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopePlanWrite)
	if !ok {
		return
	}
	planning, ok := scope.PlanningSessionScope()
	if !ok {
		writeRunnerFlowMismatch(w)
		return
	}
	s.runnerPlanningUpdate(w, r, planning)
}

func (s *Server) handleRunnerGlobalMergeCheck(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeMergeChecksWrite)
	if !ok {
		return
	}
	merge, ok := scope.MergeConfidenceScope()
	if !ok {
		writeRunnerFlowMismatch(w)
		return
	}
	s.reportRunnerMergeConfidenceCheck(w, r, merge)
}

func (s *Server) handleRunnerGlobalArtifactUpload(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunnerAny(w, r, runneraction.ScopeEvidenceWrite, runneraction.ScopeLinksWrite)
	if !ok {
		return
	}
	artifact, ok := scope.ArtifactUploadScope()
	if !ok {
		writeRunnerFlowMismatch(w)
		return
	}
	s.uploadRunnerArtifact(w, r, artifact)
}

func (s *Server) handleRunnerGlobalArtifactLink(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeLinksWrite)
	if !ok {
		return
	}
	artifact, ok := scope.ArtifactUploadScope()
	if !ok {
		writeRunnerFlowMismatch(w)
		return
	}
	s.attachRunnerArtifactLink(w, r, artifact)
}

func (s *Server) handleRunnerGlobalArtifactMarkdown(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, runneraction.ScopeLinksWrite)
	if !ok {
		return
	}
	artifact, ok := scope.ArtifactUploadScope()
	if !ok {
		writeRunnerFlowMismatch(w)
		return
	}
	s.attachRunnerArtifactMarkdown(w, r, artifact)
}

func writeRunnerFlowMismatch(w http.ResponseWriter) {
	http.Error(w, "This run cannot use this tool", http.StatusForbidden)
}
