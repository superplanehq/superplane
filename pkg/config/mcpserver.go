package config

import "os"

// EnvMCPServer turns on the public workspace MCP server and its
// authorization endpoints. The value yes enables the surface. Any other
// value, including empty, leaves the routes unregistered.
const EnvMCPServer = "SUPERPLANE_MCP_SERVER"

func MCPServerEnabled() bool {
	return os.Getenv(EnvMCPServer) == "yes"
}
