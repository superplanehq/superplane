package public

import (
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	remotelogs "github.com/superplanehq/superplane/pkg/runners/logs/remote"
)

func (s *Server) activeRunnerLogStore() runnerlogs.Store {
	if store := runnerlogs.Current(); store != nil {
		return store
	}
	if s.jwt == nil {
		return nil
	}
	return remotelogs.FromEnvironment(s.jwt.Secret)
}
