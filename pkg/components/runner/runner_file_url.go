package runner

import (
	"net/url"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/storedfiles"
)

// RewriteLoopbackTaskFileURLs points localhost file URLs at the origin a
// local runner container can reach. A remote broker leaves every URL as
// minted. A public host, including object storage, stays unchanged.
func RewriteLoopbackTaskFileURLs(text string, files []storedfiles.DispatchFile) (string, []storedfiles.DispatchFile) {
	if !isLocalTaskBrokerURL(os.Getenv("TASK_BROKER_BASE_URL")) || len(files) == 0 {
		return text, files
	}
	rewritten := make([]storedfiles.DispatchFile, len(files))
	copy(rewritten, files)
	changed := false
	for i := range rewritten {
		next := rewriteLoopbackTaskFileURL(rewritten[i].URL)
		if next == rewritten[i].URL {
			continue
		}
		text = strings.ReplaceAll(text, rewritten[i].URL, next)
		rewritten[i].URL = next
		changed = true
	}
	if !changed {
		return text, files
	}
	return text, rewritten
}

func rewriteLoopbackTaskFileURL(raw string) string {
	if !isLoopbackBaseURL(raw) {
		return raw
	}
	base := localComposeSuperplaneBaseURL("")
	if base == "" || isLoopbackBaseURL(base) {
		return raw
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	baseURL, err := url.Parse(base)
	if err != nil || baseURL.Host == "" {
		return raw
	}
	parsed.Scheme = baseURL.Scheme
	parsed.Host = baseURL.Host
	return parsed.String()
}
