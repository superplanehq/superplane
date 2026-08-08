// Package buildinfo holds build-time metadata shared by the server and CLI.
package buildinfo

// Version is the release version of this build.
// It is set at build time via -ldflags:
//
//	go build -ldflags "-X github.com/superplanehq/superplane/pkg/buildinfo.Version=v1.2.3"
//
// Defaults to "dev" for development builds.
var Version = "dev"

// VersionHeader is the HTTP response header the server stamps with Version and
// the CLI reads back to detect version skew. Kept here so the server and CLI
// share one definition and cannot drift on the header name.
const VersionHeader = "X-Superplane-Version"
