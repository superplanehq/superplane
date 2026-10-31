package runners

import (
	lifecycle "github.com/superplanehq/superplane/pkg/grpc/actions/runners"
	"github.com/superplanehq/superplane/pkg/jwt"
)

// Service retains the installation transport contract.
type Service = lifecycle.Service

func NewService(signer *jwt.Signer, baseURL string) *Service {
	return lifecycle.NewService(signer, baseURL)
}
