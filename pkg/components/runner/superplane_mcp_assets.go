package runner

import (
	"embed"
	"io/fs"
	"strings"
)

// GlobalMCPServerName is the MCP name of the global runner server. The legacy
// flow servers keep the name "superplane".
const GlobalMCPServerName = "superplane_global"

//go:embed superplane_mcp
var globalMCPAssets embed.FS

// GlobalMCPFiles returns the global runner MCP server. The runner writes the
// files under SUPERPLANE_TASK_DIR/superplane_mcp.
func GlobalMCPFiles() []TaskFile {
	files := []TaskFile{}
	err := fs.WalkDir(globalMCPAssets, "superplane_mcp", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".js") || strings.HasSuffix(path, "_test.js") {
			return err
		}
		content, err := globalMCPAssets.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, TaskFile{Path: path, Content: string(content), Mode: "0644"})
		return nil
	})
	if err != nil {
		panic("read embedded global runner MCP files: " + err.Error())
	}
	return files
}

// AppendGlobalMCP ships the global runner MCP server to runs that hold a
// runner token.
func AppendGlobalMCP(environment []TaskEnvironmentVariable, files []TaskFile) []TaskFile {
	if !HasRunnerToken(environment) {
		return files
	}
	return appendUniqueTaskFiles(files, GlobalMCPFiles()...)
}
