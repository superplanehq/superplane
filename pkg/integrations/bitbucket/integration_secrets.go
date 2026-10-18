package bitbucket

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretBitbucketToken       = "BITBUCKET_TOKEN"
	integrationSecretBitbucketGitUsername = "BITBUCKET_GIT_USERNAME"
	integrationSecretBitbucketEmail       = "BITBUCKET_EMAIL"
	bitbucketSetupName                    = "Set up Bitbucket"

	gitUsernameWorkspaceAccessToken = "x-token-auth"
	gitUsernameAPIToken             = "x-bitbucket-api-token-auth"
)

const bitbucketSecretUsage = `A Bitbucket token is available in the BITBUCKET_TOKEN environment variable.
Git is already configured to use it for https://bitbucket.org. Use normal HTTPS repository URLs such as https://bitbucket.org/<workspace>/<repo>.git.
For the Bitbucket REST API (https://api.bitbucket.org/2.0), send "Authorization: Bearer $BITBUCKET_TOKEN".
When BITBUCKET_EMAIL is set, use HTTP basic authentication with BITBUCKET_EMAIL and BITBUCKET_TOKEN instead.
Do not put the token in URLs, commands, Git configuration, or output.`

const bitbucketSetupScript = `set -euo pipefail
: "${BITBUCKET_TOKEN:?BITBUCKET_TOKEN is required}"
: "${BITBUCKET_GIT_USERNAME:?BITBUCKET_GIT_USERNAME is required}"
git config --global credential.https://bitbucket.org.helper ''
git config --global --add credential.https://bitbucket.org.helper '!f() { test "$1" = get || exit 0; echo "username=${BITBUCKET_GIT_USERNAME}"; echo "password=${BITBUCKET_TOKEN}"; }; f'
`

func (b *Bitbucket) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return core.IntegrationSecrets{}, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(metadata.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return core.IntegrationSecrets{}, err
	}

	token := strings.TrimSpace(client.Token)
	if token == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("token is required")
	}

	values := map[string][]byte{
		integrationSecretBitbucketToken:       []byte(token),
		integrationSecretBitbucketGitUsername: []byte(gitUsernameWorkspaceAccessToken),
	}

	if client.AuthType == AuthTypeAPIToken {
		values[integrationSecretBitbucketGitUsername] = []byte(gitUsernameAPIToken)
		values[integrationSecretBitbucketEmail] = []byte(strings.TrimSpace(client.Email))
	}

	return core.IntegrationSecrets{
		Values:    values,
		Usage:     bitbucketSecretUsage,
		Setup:     bitbucketSetupScript,
		SetupName: bitbucketSetupName,
	}, nil
}
