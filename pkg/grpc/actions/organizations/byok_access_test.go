package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

// Exercise effective permission through the APIs and the selectable catalog.
// In particular, a rollout alone must never bypass a gated product feature.
func TestBYOKAccess(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	var original models.PlanFeatureDefinition
	require.NoError(t, db.First(&original, "key = ?", "byo_key").Error)
	t.Cleanup(func() { require.NoError(t, models.UpdatePlanFeatureDefinition(db, &original)) })
	plan := models.BillingPlan{Key: "byok_" + uuid.NewString()[:8], Name: "BYOK", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	for _, provider := range []string{models.UsageProviderOpenAI, models.UsageProviderCustom} {
		_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, provider, datatypes.JSONSlice[string]{"test-model"})
		require.NoError(t, err)
	}
	for _, tc := range []struct {
		name                                                      string
		rollout, custom, gated, included, grant, billing, allowed bool
		provider                                                  string
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.billing {
				t.Setenv(models.BillingEnabledEnv, "yes")
			} else {
				t.Setenv(models.BillingEnabledEnv, "no")
			}
			require.NoError(t, models.UpdatePlanFeatureDefinition(db, &models.PlanFeatureDefinition{Key: "byo_key", Label: original.Label, Gated: tc.gated}))
			plan.Features = []string{}
			if tc.included {
				plan.Features = []string{"byo_key"}
			}
			require.NoError(t, models.SaveBillingPlan(db, &plan))
			require.NoError(t, models.SetOrganizationFeatureGrant(db, models.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: tc.grant}))
			flags := datatypes.JSONSlice[string]{}
			if tc.rollout {
				flags = append(flags, features.FeatureOrganizationBYOK)
			}
			if tc.custom {
				flags = append(flags, features.FeatureOrganizationBYOKCustomProvider)
			}
			require.NoError(t, db.Model(r.Organization).Update("enabled_experimental_features", flags).Error)
			billing, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
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
}
