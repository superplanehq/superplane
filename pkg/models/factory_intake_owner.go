package models

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindSubscriptionIntakeOwnerForRun reads ownership from the run's persisted
// version. Event input and agent-supplied owner IDs cannot authorize intake.
// An unrelated automation returns gorm.ErrRecordNotFound.
func FindSubscriptionIntakeOwnerForRun(tx *gorm.DB, organizationID, factoryID, runID uuid.UUID) (uuid.UUID, error) {
	run, err := FindUnscopedCanvasRun(tx, runID)
	if err != nil {
		return uuid.Nil, err
	}
	canvas, err := FindCanvasInTransaction(tx, organizationID, run.WorkflowID)
	if err != nil {
		return uuid.Nil, err
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != factoryID {
		return uuid.Nil, ErrSubscriptionTaskAssociation
	}
	version, err := FindCanvasVersionInTransaction(tx, canvas.ID, run.VersionID)
	if err != nil {
		return uuid.Nil, err
	}
	intake, err := FindFactoryIntakeByCanvasID(tx, canvas.ID)
	if err != nil && !errors.Is(err, ErrFactoryIntakeNotFound) {
		return uuid.Nil, err
	}
	if intake != nil {
		if intake.OrganizationID != organizationID || intake.FactoryID != factoryID || !ValidFactoryIntakeSource(intake.Source) {
			return uuid.Nil, ErrSubscriptionTaskAssociation
		}
	} else {
		if FactoryAppTemplateID(version.Nodes) != "scheduled-intake" {
			return uuid.Nil, gorm.ErrRecordNotFound
		}
		if canvas.ColumnKey == nil || *canvas.ColumnKey != CanvasColumnKeyBacklog || run.NodeID != "schedule" {
			return uuid.Nil, ErrSubscriptionTaskAssociation
		}
		var schedules, agents, markers int
		for _, node := range version.Nodes {
			if metadata, ok := node.Metadata[FactoryAppTemplateMetadataKey].(map[string]any); ok {
				markers++
				if node.ID != "schedule" || metadata["id"] != "scheduled-intake" {
					return uuid.Nil, ErrSubscriptionTaskAssociation
				}
			}
			if node.ID == "schedule" && node.ComponentName() == "schedule" {
				schedules++
			}
			if node.ID == "scheduled-agent" && node.ComponentName() == "runnerCodex" {
				agents++
			}
		}
		if schedules != 1 || agents != 1 || markers != 1 {
			return uuid.Nil, ErrSubscriptionTaskAssociation
		}
	}
	ownerID, err := subscriptionIntakeDefaultOwner(version.Nodes)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := ActiveCodexMember(tx, organizationID, ownerID); err != nil {
		if errors.Is(err, ErrCodexConnectionNotEligible) {
			return uuid.Nil, ErrSubscriptionOwnerInvalid
		}
		return uuid.Nil, err
	}
	return ownerID, nil
}

// RunUsesSupersededVersion reports whether a newer live version replaced the
// version that the run started on. A model source switch publishes such a
// version, so a run that started before the switch reports true.
func RunUsesSupersededVersion(tx *gorm.DB, runID uuid.UUID) (bool, error) {
	run, err := FindUnscopedCanvasRun(tx, runID)
	if err != nil {
		return false, err
	}
	var canvas Canvas
	if err := tx.Select("live_version_id").Where("id = ?", run.WorkflowID).First(&canvas).Error; err != nil {
		return false, err
	}
	return canvas.LiveVersionID != nil && *canvas.LiveVersionID != run.VersionID, nil
}

// CreateIntakeWorkOrder preserves a configured active owner after their Codex
// disconnects. Interactive creation continues to require a connected owner.
func (f *Factory) CreateIntakeWorkOrder(tx *gorm.DB, title, description string, sourceRunID uuid.UUID, origin *WorkOrderOrigin) (*FactoryWorkOrder, error) {
	ownerID, err := FindSubscriptionIntakeOwnerForRun(tx, f.OrganizationID, f.ID, sourceRunID)
	if err != nil {
		return nil, err
	}
	return f.insertWorkOrder(tx, title, description, nil, []uuid.UUID{ownerID}, &sourceRunID, origin, nil)
}

// ValidateSubscriptionIntakeOwners protects settings saved through the canvas
// editor as well as intake APIs. Existing active defaults can reconnect later.
func ValidateSubscriptionIntakeOwners(tx *gorm.DB, canvas *Canvas, draft, live *CanvasVersion) error {
	if canvas.FactoryID == nil || draft == nil {
		return nil
	}
	f, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil || !f.OnboardingConfigValue().CodexSubscription {
		return err
	}
	intake, err := FindFactoryIntakeByCanvasID(tx, canvas.ID)
	if err != nil && !errors.Is(err, ErrFactoryIntakeNotFound) {
		return err
	}
	scheduled := FactoryAppTemplateID(draft.Nodes) == "scheduled-intake" || (live != nil && FactoryAppTemplateID(live.Nodes) == "scheduled-intake")
	if intake == nil && !scheduled {
		return nil
	}
	ownerID, err := subscriptionIntakeDefaultOwner(draft.Nodes)
	if err != nil {
		return err
	}
	if _, err := ActiveCodexMember(tx, canvas.OrganizationID, ownerID); err != nil {
		if errors.Is(err, ErrCodexConnectionNotEligible) {
			return ErrSubscriptionOwnerInvalid
		}
		return err
	}
	if live != nil {
		previous, err := subscriptionIntakeDefaultOwner(live.Nodes)
		if err == nil && previous == ownerID {
			return nil
		}
	}
	eligible, err := SubscriptionOwnerEligible(tx, canvas.OrganizationID, *canvas.FactoryID, ownerID)
	if err != nil {
		return err
	}
	if !eligible {
		return ErrSubscriptionOwnerInvalid
	}
	return nil
}

func subscriptionIntakeDefaultOwner(nodes []Node) (uuid.UUID, error) {
	var ownerID uuid.UUID
	var creators int
	for _, node := range nodes {
		if node.ComponentName() != "createWorkOrder" {
			continue
		}
		creators++
		value, _ := node.Configuration["defaultOwnerId"].(string)
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil || id == uuid.Nil {
			return uuid.Nil, ErrSubscriptionOwnerRequired
		}
		ownerID = id
	}
	if creators != 1 {
		return uuid.Nil, ErrSubscriptionTaskAssociation
	}
	return ownerID, nil
}
