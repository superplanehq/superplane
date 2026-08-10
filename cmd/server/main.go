package main

import (
	// Embed the IANA timezone database into the binary so time.LoadLocation works
	// even in containers that don't ship the system tzdata package. This is required
	// for schedules and time gates to resolve DST-aware IANA identifiers.
	_ "time/tzdata"

	"github.com/superplanehq/superplane/pkg/server"
)

func main() {
	server.Start()
}
