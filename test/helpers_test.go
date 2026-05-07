package e2e_test

import (
	"os"
	"strings"
)

// subprocessEnv builds an environment for child processes spawned in tests.
// It drops AUTH_TOKEN so a developer shell exporting fleet-manager AUTH_TOKEN
// does not make unauthenticated POST /v1/tasks return 401.
func subprocessEnv(extra ...string) []string {
	var out []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "AUTH_TOKEN=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, extra...)
}
