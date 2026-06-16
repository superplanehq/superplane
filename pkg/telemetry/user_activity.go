package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/metadata"
)

const userActivityUpdateInterval = time.Minute

var userActivityLastUpdated sync.Map

func resetUserActivityThrottleForTests() {
	userActivityLastUpdated = sync.Map{}
}

func recordUserDatabaseActivity(ctx context.Context) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return
	}

	now := time.Now()
	if lastUpdated, loaded := userActivityLastUpdated.Load(userID); loaded {
		if now.Sub(lastUpdated.(time.Time)) < userActivityUpdateInterval {
			return
		}
	}

	userActivityLastUpdated.Store(userID, now)

	go func() {
		_ = models.TouchUserLastActiveAt(userID, now)
	}()
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, false
	}

	userMeta, ok := md["x-user-id"]
	if !ok || len(userMeta) == 0 {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(userMeta[0])
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}
