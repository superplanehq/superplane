package models

// RunMode selects what the runner executes. Orthogonal to ExecutionMode (host vs docker).
type RunMode string

const (
	// RunModeCommandList runs Commands as sequential shell directives (today's `commands` field).
	RunModeCommandList RunMode = "command_list"
	// RunModeArgv runs Command as a single argv subprocess (today's `command` field).
	RunModeArgv RunMode = "argv"
	// RunModeJavaScript runs Script with node; global $ is the message chain.
	RunModeJavaScript RunMode = "javascript_script"
	// RunModePython runs Script with python3; main(payload) receives the message chain.
	RunModePython RunMode = "python_script"
	// RunModeBash runs Script with bash; message_chain is written to SUPERPLANE_PAYLOAD_FILE.
	RunModeBash RunMode = "bash_script"
)

// InferRunMode maps legacy task fields when RunMode is unset.
func InferRunMode(commands CommandList, command []string, script string) RunMode {
	if len(commands) > 0 {
		return RunModeCommandList
	}
	if len(command) > 0 {
		return RunModeArgv
	}
	if script != "" {
		return RunModeJavaScript
	}
	return ""
}
