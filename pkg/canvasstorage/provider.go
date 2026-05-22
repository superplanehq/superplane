package canvasstorage

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/config"
)

func NewProvider(cfg config.CanvasStorageConfig) (Provider, error) {
	switch cfg.Driver {
	case config.CanvasStorageDriverCodeStorage:
		return NewCodeStorageProvider(cfg)
	case config.CanvasStorageDriverLocalGit:
		return NewLocalGitProvider(cfg)
	case config.CanvasStorageDriverDisabled:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported canvas storage driver %q", cfg.Driver)
	}
}
