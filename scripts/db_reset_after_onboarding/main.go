package main

import (
	"context"
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
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
		return models.ResetLocalAfterOnboarding(tx)
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
	if err := db.Model(&models.OrganizationBillingPlan{}).
		Where("plan = ? AND plan_source = ?", models.BillingPlanTrial, models.BillingPlanSourceSystem).
		Count(&summary.trialOrgs).Error; err != nil {
		return summary, err
	}
	return summary, nil
}

func encryptorFromEnv() (crypto.Encryptor, error) {
	if os.Getenv("NO_ENCRYPTION") == "yes" {
		return crypto.NewNoOpEncryptor(), nil
	}

	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be set when NO_ENCRYPTION is not yes")
	}

	return crypto.NewAESGCMEncryptor([]byte(key)), nil
}
