package runner

import _ "embed"

//go:embed turn_telemetry.js
var TurnTelemetryScript string

func TurnTelemetryTaskFile() TaskFile {
	return TaskFile{Path: "turn_telemetry.js", Content: TurnTelemetryScript, Mode: "0644"}
}
