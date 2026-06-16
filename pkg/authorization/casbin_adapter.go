package authorization

import (
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

const casbinRuleTable = "casbin_rule"

// newCasbinAdapter creates the primary Casbin adapter without AutoMigrate.
// The casbin_rule schema is managed by SQL migrations.
func newCasbinAdapter(db *gorm.DB) (*gormadapter.Adapter, error) {
	gormadapter.TurnOffAutoMigrate(db)
	return gormadapter.NewTransactionalAdapterByDB(db)
}

// newCasbinFilteredAdapter creates a filtered Casbin adapter for read paths without AutoMigrate.
func newCasbinFilteredAdapter(db *gorm.DB) (*gormadapter.Adapter, error) {
	return gormadapter.NewFilteredAdapterByDB(db, "", casbinRuleTable)
}
