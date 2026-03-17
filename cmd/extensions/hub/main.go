package main

import (
	"log"
	"net/http"
	"os"

	"github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/extensions/hub"
	appjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	artifactstorage "github.com/superplanehq/superplane/pkg/storage"
)

func main() {
	addr := os.Getenv("EXTENSION_WORKER_HUB_ADDR")
	if addr == "" {
		addr = ":8092"
	}

	extensionStorage, err := newExtensionStorage()
	if err != nil {
		log.Fatalf("error creating extension storage: %v", err)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: hub.New(extensionStorage, newJWTSigner()).Routes(),
	}

	log.Printf("Starting extension worker hub on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func newExtensionStorage() (*extensions.Storage, error) {
	manifestLoader := func(organizationID string) (*extensions.Manifest, error) {
		return models.LoadManifest(organizationID)
	}

	extensionsDir := os.Getenv("EXTENSIONS_DIR")
	if extensionsDir == "" {
		return extensions.NewStorage(artifactstorage.NewInMemoryStorage(), manifestLoader)
	}

	return extensions.NewStorage(artifactstorage.NewLocalFolderStorage(extensionsDir), manifestLoader)
}

func newJWTSigner() *appjwt.Signer {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	return appjwt.NewSigner(jwtSecret)
}
