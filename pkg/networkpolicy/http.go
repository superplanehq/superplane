package networkpolicy

import (
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
)

var DefaultBlockedHTTPHosts = []string{
	"metadata.google.internal",
	"metadata.goog",
	"metadata.azure.com",
	"169.254.169.254",
	"fd00:ec2::254",
	"kubernetes.default",
	"kubernetes.default.svc",
	"kubernetes.default.svc.cluster.local",
	"localhost",
	"127.0.0.1",
	"::1",
	"0.0.0.0",
	"::",
}

var DefaultBlockedPrivateIPRanges = []string{
	"0.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

type HTTPPolicy struct {
	AllowPrivateNetworkAccess bool
	BlockedHosts              []string
	PrivateIPRanges           []string
	BlockedHostsOverridden    bool
	PrivateIPRangesOverridden bool
}

func ResolveHTTPPolicy() (*HTTPPolicy, error) {
	metadata, err := models.GetInstallationMetadata()
	if err != nil {
		return nil, err
	}

	return ResolveHTTPPolicyForSetting(metadata.AllowPrivateNetworkAccess), nil
}

func ResolveHTTPPolicyForSetting(allowPrivateNetworkAccess bool) *HTTPPolicy {
	blockedHosts, blockedHostsOverridden := lookupListEnv("BLOCKED_HTTP_HOSTS")
	privateIPRanges, privateIPRangesOverridden := lookupListEnv("BLOCKED_PRIVATE_IP_RANGES")

	if !blockedHostsOverridden {
		if allowPrivateNetworkAccess {
			blockedHosts = []string{}
		} else {
			blockedHosts = clone(DefaultBlockedHTTPHosts)
		}
	}

	if !privateIPRangesOverridden {
		if allowPrivateNetworkAccess {
			privateIPRanges = []string{}
		} else {
			privateIPRanges = clone(DefaultBlockedPrivateIPRanges)
		}
	}

	return &HTTPPolicy{
		AllowPrivateNetworkAccess: allowPrivateNetworkAccess,
		BlockedHosts:              blockedHosts,
		PrivateIPRanges:           privateIPRanges,
		BlockedHostsOverridden:    blockedHostsOverridden,
		PrivateIPRangesOverridden: privateIPRangesOverridden,
	}
}

func lookupListEnv(key string) ([]string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return nil, false
	}

	if strings.TrimSpace(value) == "" {
		return []string{}, true
	}

	items := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		items = append(items, trimmed)
	}

	return items, true
}

func clone(values []string) []string {
	return append([]string{}, values...)
}
