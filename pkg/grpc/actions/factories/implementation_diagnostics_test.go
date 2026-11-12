package factories

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func TestImplementationPullRequestCheck(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			result, err := materializeFactoryTemplate("line-implementation", provider, factoryTemplateInput{appID: "app", appName: "Implement", multiRepo: true})
			require.NoError(t, err)
			canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
			require.NoError(t, err)
			agent := findYAMLNode(t, canvas, implementationAgentNodeID)
			script := implementationStep(t, agent, "Check pull requests")["command"].(string)
			bin := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(bin, "superplane"), []byte("#!/bin/bash\nprintf '%s' \"$PR_RESPONSE\"\nprintf '%s' \"$PR_ERROR\" >&2\nexit \"$PR_EXIT\"\n"), 0700))
			for _, tc := range []struct {
				name, response, cliError, message string
				cliExit, exit                     int
			}{
				{name: "has PR", response: `{"pull_requests":[{"url":"https://example.com/pr/1"}]}`},
				{name: "no PR", response: `{"pull_requests":[]}`, message: "did not open a pull request", exit: 1},
				{name: "authentication", cliError: "Authentication failed", cliExit: 23, message: "Authentication failed", exit: 23},
				{name: "network", cliError: "connection reset", cliExit: 7, message: "connection reset", exit: 7},
				{name: "invalid JSON", response: `not JSON`, message: "Invalid pull request response", exit: 1},
				{name: "missing array", response: `{}`, message: "Invalid pull request response", exit: 1},
			} {
				t.Run(tc.name, func(t *testing.T) {
					cmd := exec.Command("bash", "-c", script)
					cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "PR_RESPONSE="+tc.response, "PR_ERROR="+tc.cliError, "PR_EXIT="+fmt.Sprint(tc.cliExit))
					output, err := cmd.CombinedOutput()
					if tc.exit == 0 {
						require.NoError(t, err, string(output))
						return
					}
					var exit *exec.ExitError
					require.ErrorAs(t, err, &exit, string(output))
					assert.Equal(t, tc.exit, exit.ExitCode())
					assert.Contains(t, string(output), tc.message)
					if tc.cliExit != 0 {
						assert.NotContains(t, string(output), "did not open a pull request")
					}
				})
			}
		})
	}
}
