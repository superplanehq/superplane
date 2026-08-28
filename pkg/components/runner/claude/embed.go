package claude

import _ "embed"

//go:embed run.js
var runScript string

//go:embed ask_work_order_mcp.js
var askWorkOrderMCPScript string

//go:embed mcp.json
var askWorkOrderMCPConfig string
