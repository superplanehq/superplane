package authorization2

import (
	"fmt"
	"os"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/superplanehq/superplane/pkg/database"
)

func enforcer() (*casbin.TransactionalEnforcer, error) {
	modelPath := os.Getenv("RBAC_MODEL_PATH")

	adapter, err := gormadapter.NewTransactionalAdapterByDB(database.Conn())
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewTransactionalEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return enforcer, nil
}
