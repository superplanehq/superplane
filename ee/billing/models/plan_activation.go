package models

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Catalog keys that count as plan activations.
const (
	PlanActivationKeyTeam       = "team"
	PlanActivationKeyStarter    = "starter"
	PlanActivationKeyPayAsYouGo = "pay_as_you_go"
)

var planActivationKeys = []string{
	PlanActivationKeyTeam,
	PlanActivationKeyStarter,
	PlanActivationKeyPayAsYouGo,
}

// PlanActivation is one successful transition into a target plan. The
// activation time is the local transaction time of that transition. Later
// plan changes never change or remove the record.
type PlanActivation struct {
	ID             uuid.UUID `gorm:"primary_key"`
	OrganizationID uuid.UUID
	PlanKey        string
	ActivatedAt    time.Time
	CreatedAt      time.Time
}

func (PlanActivation) TableName() string {
	return "organization_plan_activations"
}

// PlanActivationCounts holds the activation count for each target plan.
type PlanActivationCounts struct {
	Team       int64 `json:"team"`
	Starter    int64 `json:"starter"`
	PayAsYouGo int64 `json:"pay_as_you_go"`
}

func NewPlanActivation(orgID uuid.UUID, planKey string, activatedAt time.Time) PlanActivation {
	return PlanActivation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		PlanKey:        planKey,
		ActivatedAt:    activatedAt.UTC(),
	}
}

// activePlanKey returns the catalog key of the plan that grants paid access
// now, or an empty string when no such plan is active.
func (p *OrganizationBillingPlan) activePlanKey() string {
	if p == nil || !p.IsActiveBusiness() {
		return ""
	}
	return p.Catalog.Key
}

// recordPlanActivation writes an activation when next makes a target plan
// active and previous did not have that plan active. A repeated assignment,
// a source-only change, or a renewal keeps the same active key and writes
// nothing.
func recordPlanActivation(tx *gorm.DB, previous, next *OrganizationBillingPlan, at time.Time) error {
	key := next.activePlanKey()
	if !slices.Contains(planActivationKeys, key) || previous.activePlanKey() == key {
		return nil
	}
	activation := NewPlanActivation(next.OrganizationID, key, at)
	return tx.Create(&activation).Error
}

// CountPlanActivations counts activations in [start, end) that occurred on or
// after the tracking start. It does not depend on the current plan or on the
// registration time of an organization.
func CountPlanActivations(tx *gorm.DB, start, end, trackingStart time.Time) (PlanActivationCounts, error) {
	var rows []struct {
		PlanKey string
		Count   int64
	}
	err := tx.Model(&PlanActivation{}).
		Select("plan_key, COUNT(*) AS count").
		Where("activated_at >= ? AND activated_at < ?", start, end).
		Where("activated_at >= ?", trackingStart).
		Group("plan_key").
		Scan(&rows).Error
	if err != nil {
		return PlanActivationCounts{}, err
	}
	var counts PlanActivationCounts
	for _, row := range rows {
		switch row.PlanKey {
		case PlanActivationKeyTeam:
			counts.Team = row.Count
		case PlanActivationKeyStarter:
			counts.Starter = row.Count
		case PlanActivationKeyPayAsYouGo:
			counts.PayAsYouGo = row.Count
		}
	}
	return counts, nil
}

// planActivationTracking is the installation-wide cutoff for activation
// history. The migration seeds it once. Installation metadata is not used
// because the installation singleton can be recreated or reset.
type planActivationTracking struct {
	ID        int `gorm:"primary_key"`
	StartedAt time.Time
}

func (planActivationTracking) TableName() string {
	return "plan_activation_tracking"
}

// PlanActivationTrackingStartedAt returns when the installation started
// recording plan activations. Activations before this time are not tracked.
func PlanActivationTrackingStartedAt(tx *gorm.DB) (time.Time, error) {
	var tracking planActivationTracking
	if err := tx.Where("id = ?", 1).First(&tracking).Error; err != nil {
		return time.Time{}, err
	}
	return tracking.StartedAt.UTC(), nil
}
