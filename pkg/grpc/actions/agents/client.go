package agents

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func closeAgentConnection(conn *grpc.ClientConn) {
	if conn == nil {
		return
	}

	if err := conn.Close(); err != nil {
		log.WithError(err).Warn("failed to close agent GRPC client")
	}
}
