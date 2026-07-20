package claude

import _ "embed"

//go:embed stream_format.js
var streamFormatJS string

//go:embed prompt_step.sh
var promptStepScript string
