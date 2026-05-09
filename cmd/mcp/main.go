package main

import (
	"log"

	"github.com/superplanehq/superplane/pkg/mcp"
)

const version = "dev"

func main() {
	if err := mcp.StartServer(version); err != nil {
		log.Fatal(err)
	}
}
