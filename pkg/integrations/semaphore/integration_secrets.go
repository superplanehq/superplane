package semaphore

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	integrationSecretSemaphoreAPIToken        = "SEMAPHORE_API_TOKEN"
	integrationSecretSemaphoreOrganizationURL = "SEMAPHORE_ORGANIZATION_URL"
	semaphoreSetupName                        = "Set up Semaphore"
	semaphoreSecretUsage                      = `A Semaphore API token is available in the SEMAPHORE_API_TOKEN environment variable.
The organization URL is available in the SEMAPHORE_ORGANIZATION_URL environment variable.
The sem-ai CLI is on PATH and is connected to this organization.
Use sem-ai for projects, pipelines, diagnose, and YAML validate.
Do not print the token.`
	semaphoreSecretUsageWithoutOrganizationURL = `A Semaphore API token is available in the SEMAPHORE_API_TOKEN environment variable.
Use this token as a Bearer token for the Semaphore API.
Do not print the token.`
	semaphoreSetupScript = `set -euo pipefail
: "${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}"
: "${SEMAPHORE_API_TOKEN:?SEMAPHORE_API_TOKEN is required}"
: "${SEMAPHORE_ORGANIZATION_URL:?SEMAPHORE_ORGANIZATION_URL is required}"

bin="$SUPERPLANE_TASK_DIR/bin"
lib="$SUPERPLANE_TASK_DIR/lib"
home="$SUPERPLANE_TASK_DIR/home"
mkdir -p "$bin" "$lib" "$home"

if [ ! -x "$lib/sem-ai" ]; then
  if command -v sem-ai >/dev/null 2>&1; then
    cp "$(command -v sem-ai)" "$lib/sem-ai"
  else
    curl -fsSL https://raw.githubusercontent.com/semaphoreio/sem-ai/main/install.sh | sh
    if [ -x "${HOME}/.local/bin/sem-ai" ]; then
      cp "${HOME}/.local/bin/sem-ai" "$lib/sem-ai"
    elif [ -x "${HOME}/.semaphore-ai/bin/sem-ai" ]; then
      cp "${HOME}/.semaphore-ai/bin/sem-ai" "$lib/sem-ai"
    elif command -v sem-ai >/dev/null 2>&1; then
      cp "$(command -v sem-ai)" "$lib/sem-ai"
    else
      echo "sem-ai install did not produce a binary" >&2
      exit 1
    fi
  fi
  chmod +x "$lib/sem-ai"
fi

cat > "$bin/sem-ai" <<EOF
#!/bin/sh
export HOME='$home'
exec '$lib/sem-ai' "\$@"
EOF
chmod +x "$bin/sem-ai"

org="${SEMAPHORE_ORGANIZATION_URL#https://}"
org="${org#http://}"
org="${org%/}"
HOME="$home" "$lib/sem-ai" connect "$org" "$SEMAPHORE_API_TOKEN"
`
)

func (s *Semaphore) ResolveSecrets(ctx core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	token, err := resolveAPIToken(ctx.Integration)
	if err != nil {
		return core.IntegrationSecrets{}, err
	}

	values := map[string][]byte{
		integrationSecretSemaphoreAPIToken: []byte(token),
	}

	orgURL := organizationURL(ctx.Integration)
	if orgURL == "" {
		return core.IntegrationSecrets{
			Values: values,
			Usage:  semaphoreSecretUsageWithoutOrganizationURL,
		}, nil
	}

	values[integrationSecretSemaphoreOrganizationURL] = []byte(orgURL)
	return core.IntegrationSecrets{
		Values:    values,
		Usage:     semaphoreSecretUsage,
		Setup:     semaphoreSetupScript,
		SetupName: semaphoreSetupName,
	}, nil
}

func resolveAPIToken(integrationCtx core.IntegrationContext) (string, error) {
	if integrationCtx.LegacySetup() {
		apiToken, err := integrationCtx.GetConfig("apiToken")
		if err != nil {
			return "", fmt.Errorf("failed to get API token: %w", err)
		}

		token := strings.TrimSpace(string(apiToken))
		if token == "" {
			return "", fmt.Errorf("API token is required")
		}

		return token, nil
	}

	token, err := integrationCtx.Secrets().Get(SecretAPIToken)
	if err != nil {
		return "", fmt.Errorf("failed to get API token: %w", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("API token is required")
	}

	return token, nil
}

func organizationURL(integrationCtx core.IntegrationContext) string {
	if !integrationCtx.LegacySetup() {
		url, err := integrationCtx.Properties().GetString(PropertyOrganizationURL)
		if err != nil {
			return ""
		}
		return strings.TrimRight(strings.TrimSpace(url), "/")
	}

	raw, err := integrationCtx.GetConfig("organizationUrl")
	if err != nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(string(raw)), "/")
}
