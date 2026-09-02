package claude

import _ "embed"

//go:embed run.js
var runScript string

//go:embed planning_session_mcp.js
var planningSessionMCPScript string

//go:embed mcp.json
var planningSessionMCPConfig string

//go:embed follow_up_loop.js
var followUpLoopScript string
