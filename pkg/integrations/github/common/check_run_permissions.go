package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/superplanehq/superplane/pkg/core"
)

// These errors appear directly in automation settings.
var (
	//revive:disable-next-line:error-strings
	ErrCheckRunPermissions = errors.New("Accept the new GitHub App permissions in GitHub installation settings, then try again. PR checks require Checks write permission.")
	//revive:disable-next-line:error-strings
	ErrCheckRunAppRequired = errors.New("Connect a GitHub App with Checks write permission to publish PR checks.")
)

// VerifyCheckRunPermissions reads fresh installation permissions without publishing a check.
func VerifyCheckRunPermissions(ctx context.Context, integration core.IntegrationContext, http core.HTTPContext) error {
	client, err := NewClient(integration, http)
	if err != nil {
		return fmt.Errorf("create GitHub client: %w", err)
	}
	transport, ok := client.underlying.Client().Transport.(*ghinstallation.Transport)
	if !ok {
		return ErrCheckRunAppRequired
	}
	if _, err := transport.Token(ctx); err != nil {
		return fmt.Errorf("read GitHub installation permissions: %w", err)
	}
	permissions, err := transport.Permissions()
	if err != nil {
		return fmt.Errorf("read GitHub installation permissions: %w", err)
	}
	if permissions.GetChecks() != "write" || (permissions.GetPullRequests() != "read" && permissions.GetPullRequests() != "write") {
		return ErrCheckRunPermissions
	}
	return nil
}
