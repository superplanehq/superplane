package secrets

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseDomainID(domainID string) (uuid.UUID, error) {
	id, err := uuid.Parse(domainID)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid domain id")
	}

	return id, nil
}

func findSecretInDomain(domainType, domainID, idOrName string) (*models.Secret, error) {
	id, err := parseDomainID(domainID)
	if err != nil {
		return nil, err
	}

	var (
		secret    *models.Secret
		lookupErr error
	)
	if err := actions.ValidateUUIDs(idOrName); err != nil {
		secret, lookupErr = models.FindSecretByName(domainType, id, idOrName)
	} else {
		secret, lookupErr = models.FindSecretByID(domainType, id, idOrName)
	}

	if lookupErr != nil {
		return nil, status.Error(codes.InvalidArgument, "secret not found")
	}

	return secret, nil
}
