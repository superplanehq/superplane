package github

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type NodeMetadata struct {
	Repository *Repository `json:"repository"`
}

type BaseRepositoryConfig struct {
	Repository string `mapstructure:"repository"`
}

func ensureRepoInMetadata(ctx core.MetadataContext, app core.AppInstallationContext, configuration any) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	//
	// If metadata is already set, skip setup
	//
	if nodeMetadata.Repository != nil {
		return nil
	}

	var config BaseRepositoryConfig
	if err := mapstructure.Decode(configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	//
	// Validate that the app has access to this repository
	//
	var appMetadata Metadata
	if err := mapstructure.Decode(app.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	repoIndex := slices.IndexFunc(appMetadata.Repositories, func(r Repository) bool {
		return r.Name == config.Repository
	})

	if repoIndex == -1 {
		return fmt.Errorf("repository %s is not accessible to app installation", config.Repository)
	}

	return ctx.Set(&NodeMetadata{
		Repository: &appMetadata.Repositories[repoIndex],
	})
}

func verifySignature(ctx core.WebhookRequestContext, expectedType string) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing X-GitHub-Event header")
	}

	//
	// If event is not of the expect type, we ignore it.
	//
	if eventType != expectedType {
		return http.StatusOK, nil
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	err = crypto.VerifySignature(secret, ctx.Body, signature)
	if err != nil {
		return http.StatusForbidden, err
	}

	return http.StatusOK, nil
}
