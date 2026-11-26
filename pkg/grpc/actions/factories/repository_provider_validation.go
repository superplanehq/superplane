package factories

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/superplanehq/superplane/pkg/models"
	"slices"
	"strings"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

// These receipts belong to one API request. They contain provider results only;
// database authorization is repeated by every preparation and mutation attempt.
type repositoryProviderChecks struct {
	results map[[32]byte]repositoryProviderResult
	inputs  map[string][32]byte
	frozen  bool
}
type repositoryProviderResult struct {
	value string
	err   error
}
type repositoryProviderCheckRequired struct {
	key [32]byte
	run func() (string, error)
}

func (*repositoryProviderCheckRequired) Error() string {
	return "repository provider validation is required"
}

type repositoryProviderChecksKey struct{}

func newRepositoryProviderChecks(ctx context.Context) (context.Context, *repositoryProviderChecks) {
	checks := &repositoryProviderChecks{results: map[[32]byte]repositoryProviderResult{}, inputs: map[string][32]byte{}}
	return context.WithValue(ctx, repositoryProviderChecksKey{}, checks), checks
}

func repositoryProviderCheck(ctx context.Context, slot string, input any, run func() (string, error)) (string, error) {
	checks, _ := ctx.Value(repositoryProviderChecksKey{}).(*repositoryProviderChecks)
	if checks == nil {
		return run()
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256(encoded)
	if previous, exists := checks.inputs[slot]; exists && previous != key {
		return "", repositoryInputsChanged()
	}
	checks.inputs[slot] = key
	if result, ok := checks.results[key]; ok {
		return result.value, result.err
	}
	if checks.frozen {
		return "", repositoryInputsChanged()
	}
	return "", &repositoryProviderCheckRequired{key: key, run: run}
}

func repositoryInputsChanged() error {
	return grpcerrors.FailedPrecondition(nil, "The configuration or requested input changed. Run plan again before apply.")
}

// A missing receipt ends the preparation transaction before any provider call.
// Rerunning preparation also reloads and reauthorizes all database inputs.
func (c *repositoryProviderChecks) prepare(db *gorm.DB, prepare func(*gorm.DB) error, options ...*sql.TxOptions) error {
	for {
		err := db.Transaction(prepare, options...)
		var required *repositoryProviderCheckRequired
		if !errors.As(err, &required) {
			return err
		}
		value, checkErr := required.run()
		c.results[required.key] = repositoryProviderResult{value: value, err: checkErr}
	}
}

func deferredRepositoryProviderCheck(err error) bool {
	var required *repositoryProviderCheckRequired
	return errors.As(err, &required) || grpcerrors.Code(err) == codes.FailedPrecondition
}

// Include stored credential revisions so a token change invalidates a receipt.
// The snapshot is hashed in memory and is never returned to the caller.
func repositoryProviderIntegration(tx *gorm.DB, integration *models.Integration) (any, error) {
	var secrets []models.IntegrationSecret
	if !integration.IsLegacy() {
		var err error
		secrets, err = models.ListIntegrationSecretsInTransaction(tx, integration.ID)
		if err != nil {
			return nil, err
		}
		slices.SortFunc(secrets, func(a, b models.IntegrationSecret) int { return strings.Compare(a.Name, b.Name) })
	}
	return struct {
		Integration *models.Integration
		Secrets     []models.IntegrationSecret
	}{integration, secrets}, nil
}
