package billinghttp

import (
	"time"

	"github.com/google/uuid"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type billingWorkspaceEvidence struct {
	ID                    uuid.UUID  `json:"id"`
	Name                  string     `json:"name"`
	OnboardingCompletedAt *time.Time `json:"onboarding_completed_at"`
}

type billingCheckoutEvidence struct {
	ID              uuid.UUID  `json:"id"`
	PlanKey         string     `json:"plan_key"`
	State           string     `json:"state"`
	PolarCheckoutID *string    `json:"polar_checkout_id"`
	PolarStatus     *string    `json:"polar_status"`
	LastError       *string    `json:"last_error"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ExpiresAt       *time.Time `json:"expires_at"`
	ResolvedAt      *time.Time `json:"resolved_at"`
}

func loadStoredBillingEvidence(tx *gorm.DB, orgID uuid.UUID, response *organizationBillingPlanResponse) error {
	settings, err := billingmodels.FindOrganizationLLMSettings(tx, orgID)
	if err != nil {
		return err
	}
	if settings != nil && settings.PolarCustomerID != nil {
		response.PolarCustomerID = *settings.PolarCustomerID
		response.PolarCustomerURL = polar.CustomerDashboardURL(response.PolarCustomerID)
		response.PolarCustomerEventsURL = polar.CustomerEventsDashboardURL(response.PolarCustomerID)
	}
	factories, err := models.ListFactories(tx, orgID)
	if err != nil {
		return err
	}
	response.Workspaces = []billingWorkspaceEvidence{}
	for _, factory := range factories {
		response.Workspaces = append(response.Workspaces, billingWorkspaceEvidence{ID: factory.ID, Name: factory.Name, OnboardingCompletedAt: factory.OnboardingCompletedAt})
	}
	attempts, err := billingmodels.ListRecentOrganizationBillingCheckouts(tx, orgID)
	if err != nil {
		return err
	}
	response.CheckoutAttempts = []billingCheckoutEvidence{}
	for _, attempt := range attempts {
		response.CheckoutAttempts = append(response.CheckoutAttempts, billingCheckoutEvidence{
			ID: attempt.ID, PlanKey: attempt.PlanKey, State: attempt.State,
			PolarCheckoutID: attempt.PolarCheckoutID, PolarStatus: attempt.PolarStatus, LastError: attempt.LastError,
			CreatedAt: attempt.CreatedAt, UpdatedAt: attempt.UpdatedAt, ExpiresAt: attempt.ExpiresAt, ResolvedAt: attempt.ResolvedAt,
		})
	}
	return nil
}
