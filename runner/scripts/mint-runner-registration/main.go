// Command mint-runner-registration signs a single-use runner registration JWT.
//
// Usage:
//
//	go run ./scripts/mint-runner-registration -fleet local -secret "$AUTH_TOKEN"
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/superplane/runner/shared/runnerregistrationtoken"
)

func main() {
	fleetID := flag.String("fleet", "", "fleet id claim")
	secret := flag.String("secret", "", "broker AUTH_TOKEN HMAC secret")
	ttl := flag.Duration("ttl", 10*time.Minute, "token lifetime")
	flag.Parse()
	if *fleetID == "" || *secret == "" {
		fmt.Fprintln(os.Stderr, "usage: mint-runner-registration -fleet <id> -secret <auth-token>")
		os.Exit(2)
	}
	token, err := runnerregistrationtoken.Mint(*fleetID, *secret, time.Now().UTC().Add(*ttl))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(token)
}
