package organizations

import (
	"context"
	"errors"
	billingorganizations "github.com/superplanehq/superplane/ee/billing/organizations"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Exercise effective permission through the APIs and the selectable catalog.
// A hosted default keeps access subject to rollout and product permissions.
func TestBYOKAccess(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	settings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
			"default_hosted_provider": settings.DefaultHostedProvider,
			"default_hosted_model":    settings.DefaultHostedModel,
		}).Error)
	})
	var original billingmodels.PlanFeatureDefinition
	require.NoError(t, db.First(&original, "key = ?", "byo_key").Error)
	t.Cleanup(func() { require.NoError(t, billingmodels.UpdatePlanFeatureDefinition(db, &original)) })
	plan := billingmodels.BillingPlan{Key: "byok_" + uuid.NewString()[:8], Name: "BYOK", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	for _, provider := range []string{models.UsageProviderAnthropic, models.UsageProviderOpenAI, models.UsageProviderOpenRouter, models.UsageProviderCustom} {
		_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, provider, datatypes.JSONSlice[string]{"test-model"})
		require.NoError(t, err)
	}
	for _, tc := range []struct {
		name                                                      string
		rollout, custom, gated, included, grant, billing, allowed bool
		provider                                                  string
		noHostedDefault                                           bool
	}{
		{name: "rollout alone", rollout: true, gated: true, billing: true, provider: "openai"},
		{name: "product alone", gated: true, included: true, billing: true, provider: "openai"},
		{name: "plan inclusion", rollout: true, gated: true, included: true, billing: true, allowed: true, provider: "openai"},
		{name: "enabled grant", rollout: true, gated: true, grant: true, billing: true, allowed: true, provider: "openai"},
		{name: "ungated omission", rollout: true, billing: true, allowed: true, provider: "openai"},
		{name: "billing disabled", rollout: true, gated: true, allowed: true, provider: "openai"},
		{name: "billing disabled still needs rollout", gated: true, provider: "openai"},
		{name: "custom needs rollout", rollout: true, gated: true, included: true, billing: true, provider: "custom"},
		{name: "custom needs product", rollout: true, custom: true, gated: true, billing: true, provider: "custom"},
		{name: "custom allowed", rollout: true, custom: true, gated: true, included: true, billing: true, allowed: true, provider: "custom"},
		{name: "no default allows anthropic with billing", gated: true, billing: true, allowed: true, provider: "anthropic", noHostedDefault: true},
		{name: "no default allows openai with billing", gated: true, billing: true, allowed: true, provider: "openai", noHostedDefault: true},
		{name: "no default allows openrouter with billing", gated: true, billing: true, allowed: true, provider: "openrouter", noHostedDefault: true},
		{name: "no default allows anthropic without billing", gated: true, allowed: true, provider: "anthropic", noHostedDefault: true},
		{name: "no default allows openai without billing", gated: true, allowed: true, provider: "openai", noHostedDefault: true},
		{name: "no default allows openrouter without billing", gated: true, allowed: true, provider: "openrouter", noHostedDefault: true},
		{name: "no default custom needs byok rollout", custom: true, gated: true, included: true, billing: true, provider: "custom", noHostedDefault: true},
		{name: "no default custom needs custom rollout", rollout: true, gated: true, included: true, billing: true, provider: "custom", noHostedDefault: true},
		{name: "no default custom needs product", rollout: true, custom: true, gated: true, billing: true, provider: "custom", noHostedDefault: true},
		{name: "no default custom allowed", rollout: true, custom: true, gated: true, included: true, billing: true, allowed: true, provider: "custom", noHostedDefault: true},
		{name: "no default custom names stay gated", gated: true, billing: true, provider: " CUSTOM ", noHostedDefault: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var provider, model any = models.UsageProviderAnthropic, "hosted-model"
			if tc.noHostedDefault {
				provider, model = nil, nil
			}
			require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
				"default_hosted_provider": provider,
				"default_hosted_model":    model,
			}).Error)
			if tc.billing {
				t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			} else {
				t.Setenv(billingmodels.BillingEnabledEnv, "no")
			}
			require.NoError(t, billingmodels.UpdatePlanFeatureDefinition(db, &billingmodels.PlanFeatureDefinition{Key: "byo_key", Label: original.Label, Gated: tc.gated}))
			plan.Features = []string{}
			if tc.included {
				plan.Features = []string{"byo_key"}
			}
			require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
			require.NoError(t, billingmodels.SetOrganizationFeatureGrant(db, billingmodels.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: tc.grant}))
			flags := datatypes.JSONSlice[string]{}
			if tc.rollout {
				flags = append(flags, features.FeatureOrganizationBYOK)
			}
			if tc.custom {
				flags = append(flags, features.FeatureOrganizationBYOKCustomProvider)
			}
			require.NoError(t, db.Model(r.Organization).Update("enabled_experimental_features", flags).Error)
			billing, err := billingorganizations.DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
			require.NoError(t, err)
			require.Equal(t, !tc.billing || !tc.gated || tc.included || tc.grant, billing.ProductFeatures["byo_key"])
			_, err = ListBYOKLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{Provider: tc.provider})
			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
			}
			_, err = UpdateBYOKLLMModels(context.Background(), r.Organization.ID.String(), &pb.UpdateBYOKLLMModelsRequest{Provider: tc.provider, AllowedModels: []string{"test-model"}})
			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
			}
			selectable, err := models.ResolveSelectableLLMModels(db, r.Organization.ID, nil, tc.provider, models.UsageFundingSourceBYOK)
			require.NoError(t, err)
			require.Equal(t, tc.allowed, len(selectable) > 0)
		})
	}
	t.Run("no default allows standard catalog", func(t *testing.T) {
		require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
			"default_hosted_provider": nil,
			"default_hosted_model":    nil,
		}).Error)
		require.NoError(t, db.Model(r.Organization).Update("enabled_experimental_features", datatypes.JSONSlice[string]{}).Error)
		allowed, err := models.OrganizationBYOKAccess(db, r.Organization.ID, "")
		require.NoError(t, err)
		require.True(t, allowed)
	})
	t.Run("hosted default lookup failure", func(t *testing.T) {
		lookupFailure := errors.New("temporary hosted default lookup failure")
		callbackName := "test:byok_hosted_default_lookup_failure"
		require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
			if tx.Statement.Table == "installation_llm_settings" {
				tx.AddError(lookupFailure)
			}
		}))
		t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })
		allowed, err := models.OrganizationBYOKAccess(db, r.Organization.ID, models.UsageProviderOpenAI)
		require.ErrorIs(t, err, lookupFailure)
		require.False(t, allowed)
	})
}
