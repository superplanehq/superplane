package git

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/git/codestorage"
	"github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/git/supergit"
)

func NewProvider() (provider.Provider, error) {
	gitProvider := strings.TrimSpace(os.Getenv("GIT_STORAGE_PROVIDER"))
	if gitProvider == "" {
		return nil, fmt.Errorf("GIT_STORAGE_PROVIDER is not set")
	}

	switch gitProvider {
	case provider.CodeStorageProvider:
		log.Println("Creating Code Storage Provider")
		return codestorage.NewProvider()
	case provider.SuperGitProvider:
		log.Println("Creating SuperGit Provider")
		return supergit.NewProvider()
	default:
		return nil, fmt.Errorf("unsupported git storage provider %q", gitProvider)
	}
}
