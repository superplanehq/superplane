package main

import (
	"context"
	"fmt"
	"os"

	eebilling "github.com/superplanehq/superplane/ee/billing"
	"github.com/superplanehq/superplane/ee/billing/development"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/registryimports"
	"gorm.io/gorm"
)

var _ = registryimports.Loaded

const polarSandboxNote = "Cancel the Polar sandbox subscription separately."

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	billing.Bind(eebilling.New(nil))
	if dbName := os.Getenv("DB_NAME"); dbName != "superplane_dev" {
		return fmt.Errorf("db.reset.after.onboarding only runs against superplane_dev")
	}

	encryptor, err := encryptorFromEnv()
	if err != nil {
		return err
	}

	reg, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	if err != nil {
		return fmt.Errorf("build integration registry: %w", err)
	}

	db := database.Conn()
	err = db.Transaction(func(tx *gorm.DB) error {
		return development.ResetLocalAfterOnboarding(tx)
	})
	if err != nil {
		return fmt.Errorf("reset runtime: %w", err)
	}

	deps := factories.IntakeDependencies{
		Registry:  reg,
		Encryptor: encryptor,
	}
	if err := reseedIntakes(context.Background(), deps, db); err != nil {
		return err
	}

	summary, err := loadResetSummary(db)
	if err != nil {
		return err
	}

	fmt.Printf("Reset local environment after onboarding.\n")
	fmt.Printf("Workspaces: %d\n", summary.workspaces)
	fmt.Printf("Work orders: %d\n", summary.workOrders)
	fmt.Printf("Usage events: %d\n", summary.usageEvents)
	fmt.Printf("Organizations on trial: %d\n", summary.trialOrgs)
	fmt.Println(polarSandboxNote)
	return nil
}

func reseedIntakes(ctx context.Context, deps factories.IntakeDependencies, db *gorm.DB) error {
	var factoryList []models.Factory
	if err := db.Find(&factoryList).Error; err != nil {
		return fmt.Errorf("list workspaces for intake seed: %w", err)
	}

	for i := range factoryList {
		if err := factories.SeedFactoryIntakes(ctx, deps, db, &factoryList[i]); err != nil {
			return fmt.Errorf("reseed intakes for workspace %s: %w", factoryList[i].ID, err)
		}
	}

	return nil
}

type resetSummary struct {
	workspaces  int64
	workOrders  int64
	usageEvents int64
	trialOrgs   int64
}

func loadResetSummary(db *gorm.DB) (resetSummary, error) {
	var summary resetSummary
	if err := db.Model(&models.Factory{}).Count(&summary.workspaces).Error; err != nil {
		return summary, err
	}
	if err := db.Model(&models.FactoryWorkOrder{}).Count(&summary.workOrders).Error; err != nil {
		return summary, err
	}
	if err := db.Model(&models.WorkspaceUsageEvent{}).Count(&summary.usageEvents).Error; err != nil {
		return summary, err
	}
	if err := db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("plan = ? AND plan_source = ?", billingmodels.BillingPlanTrial, billingmodels.BillingPlanSourceSystem).
		Count(&summary.trialOrgs).Error; err != nil {
		return summary, err
	}
	return summary, nil
}

func encryptorFromEnv() (crypto.Encryptor, error) {
	ctx := context.Background()
	masters, primary, err := crypto.MasterKeysFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	if masters == nil {
		return nil, fmt.Errorf("ENCRYPTION_MASTER_KEYS is required")
	}
	store := encryption.NewStore(masters, primary)
	if err := store.Initialize(ctx); err != nil {
		return nil, err
	}
	return crypto.NewVersionedEncryptor(store), nil
}
