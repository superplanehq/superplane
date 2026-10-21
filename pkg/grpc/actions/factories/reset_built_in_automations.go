package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

// ErrUnknownBuiltInAutomation means the caller asked to reset an automation
// that has no current template.
var ErrUnknownBuiltInAutomation = errors.New("unknown built-in automation")

const builtInAutomationResetMessage = "Reset built-in automation"

// BuiltInAutomationReset is how many built-in automations were saved.
type BuiltInAutomationReset struct {
	Reset int `json:"reset"`
}

// BuiltInAutomationResetFailure is one organization that was left unchanged.
type BuiltInAutomationResetFailure struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	Error            string `json:"error"`
}

// BuiltInAutomationInstallationReset is the result of resetting one automation
// in every organization.
type BuiltInAutomationInstallationReset struct {
	Reset         int                             `json:"reset"`
	Organizations int                             `json:"organizations"`
	Failed        []BuiltInAutomationResetFailure `json:"failed"`
}

// ResetOrganizationBuiltInAutomations replaces every built-in automation in
// the organization with its current template and saves that canvas. Custom
// automations are left unchanged. Builders carry supported user settings into
// the current templates; prompts and workflow structure come from those templates.
func ResetOrganizationBuiltInAutomations(
	ctx context.Context,
	organizationID uuid.UUID,
	deps IntakeDependencies,
) (BuiltInAutomationReset, error) {
	return resetOrganizationBuiltInAutomations(ctx, organizationID, "", deps)
}

// ResetBuiltInAutomationInAllOrganizations replaces one built-in automation in
// every organization and workspace. Each organization is saved on its own, so
// a settings failure leaves that organization unchanged and the reset continues.
func ResetBuiltInAutomationInAllOrganizations(
	ctx context.Context,
	templateID string,
	deps IntakeDependencies,
) (BuiltInAutomationInstallationReset, error) {
	if _, ok := lookupBuiltInAutomation(templateID); !ok {
		return BuiltInAutomationInstallationReset{}, ErrUnknownBuiltInAutomation
	}
	organizations, err := models.ListOrganizations(database.DB(ctx))
	if err != nil {
		return BuiltInAutomationInstallationReset{}, err
	}
	return resetBuiltInAutomationForOrganizations(ctx, organizations, templateID, deps)
}

func resetBuiltInAutomationForOrganizations(
	ctx context.Context,
	organizations []models.Organization,
	templateID string,
	deps IntakeDependencies,
) (BuiltInAutomationInstallationReset, error) {
	if _, ok := lookupBuiltInAutomation(templateID); !ok {
		return BuiltInAutomationInstallationReset{}, ErrUnknownBuiltInAutomation
	}

	result := BuiltInAutomationInstallationReset{Failed: []BuiltInAutomationResetFailure{}}
	for _, organization := range organizations {
		reset, err := resetOrganizationBuiltInAutomations(ctx, organization.ID, templateID, deps)
		if err != nil {
			log.Errorf("reset built-in automation %s for organization %s: %v", templateID, organization.ID, err)
			result.Failed = append(result.Failed, BuiltInAutomationResetFailure{
				OrganizationID:   organization.ID.String(),
				OrganizationName: organization.Name,
				Error:            resetFailureMessage(err),
			})
			continue
		}
		if reset.Reset == 0 {
			continue
		}
		result.Organizations++
		result.Reset += reset.Reset
	}
	return result, nil
}

func resetFailureMessage(err error) string {
	var settingsError *BuiltInAutomationSettingsError
	if errors.As(err, &settingsError) {
		return fmt.Sprintf("Reset stopped: %v. This organization was not changed.", err)
	}
	return "Could not reset this organization."
}

func resetOrganizationBuiltInAutomations(
	ctx context.Context,
	organizationID uuid.UUID,
	templateID string,
	deps IntakeDependencies,
) (BuiltInAutomationReset, error) {
	db := database.DB(ctx)
	factoryList, err := models.ListFactories(db, organizationID)
	if err != nil {
		return BuiltInAutomationReset{}, err
	}

	var result BuiltInAutomationReset
	err = db.Transaction(func(tx *gorm.DB) error {
		for i := range factoryList {
			canvasesForFactory, listErr := factoryList[i].ListCanvases(tx)
			if listErr != nil {
				return listErr
			}
			for j := range canvasesForFactory {
				saved, resetErr := resetBuiltInAutomation(ctx, tx, &factoryList[i], &canvasesForFactory[j], templateID, deps)
				if resetErr != nil {
					return fmt.Errorf("reset %q in factory %q: %w", canvasesForFactory[j].Name, factoryList[i].Name, resetErr)
				}
				if saved {
					result.Reset++
				}
			}
		}
		return nil
	})
	if err != nil {
		return BuiltInAutomationReset{}, err
	}
	return result, nil
}

func resetBuiltInAutomation(
	ctx context.Context,
	tx *gorm.DB,
	factory *models.Factory,
	canvas *models.Canvas,
	templateID string,
	deps IntakeDependencies,
) (bool, error) {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return false, err
	}
	if templateID != "" {
		liveTemplateID, ok, classifyErr := liveBuiltInAutomationTemplateID(tx, factory, canvas, version)
		if classifyErr != nil {
			return false, classifyErr
		}
		if !ok || liveTemplateID != templateID {
			return false, nil
		}
	}
	materialized, err := materializeLiveAutomationDefaults(tx, factory, canvas, version, factoryAutomationSettingsFromVersion(version))
	if notBuiltInAutomation(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if templateID != "" && materialized.templateID != templateID {
		return false, nil
	}

	doc, err := yaml.CanvasFromYAML([]byte(materialized.canvasYAML))
	if err != nil {
		return false, err
	}
	if err := validateBuiltInResetSettings(doc, deps.Registry); err != nil {
		return false, err
	}
	nodes, edges, err := doc.Parse(deps.Registry, canvas.OrganizationID.String())
	if err != nil {
		return false, err
	}
	err = canvases.PublishGeneratedCanvasNodesWithOwner(
		ctx,
		tx,
		canvas,
		nil,
		builtInAutomationResetMessage,
		nodes,
		edges,
		changesets.CanvasPublisherOptions{
			Registry:       deps.Registry,
			OrgID:          canvas.OrganizationID,
			Encryptor:      deps.Encryptor,
			AuthService:    deps.AuthService,
			WebhookBaseURL: deps.WebhookBaseURL,
		},
	)
	if err != nil {
		return false, err
	}
	return true, nil
}

func notBuiltInAutomation(err error) bool {
	return errors.Is(err, errInvalidArgument) && strings.Contains(err.Error(), "no bundled defaults")
}

// liveBuiltInAutomationTemplateID reports the template a reset would publish.
// Intake and pull request feedback follow materializeLiveAutomationDefaults.
// Task analysis matches the backlog app identity, so a custom automation that
// starts on a work order stays unchanged.
func liveBuiltInAutomationTemplateID(
	tx *gorm.DB,
	factory *models.Factory,
	canvas *models.Canvas,
	version *models.CanvasVersion,
) (string, bool, error) {
	intake, intakeErr := models.FindFactoryIntakeByCanvasID(tx, canvas.ID)
	switch {
	case intakeErr == nil && intake.FactoryID == factory.ID:
		return intakeTemplateID(intake.Source), true, nil
	case intakeErr != nil && !errors.Is(intakeErr, models.ErrFactoryIntakeNotFound):
		return "", false, intakeErr
	}

	handler, err := models.FindPRFeedbackHandlerByCanvasID(tx, canvas.ID)
	if err != nil {
		return "", false, err
	}
	if handler != nil && handler.FactoryID == factory.ID {
		if handler.Source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
			return prFeedbackChecksTemplateID, true, nil
		}
		return prFeedbackDiscussionTemplateID, true, nil
	}

	spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
	if models.IsBacklogFactoryApp(spec.Nodes, spec.Edges) {
		return models.FactoryAppTemplateBacklogID, true, nil
	}
	resolved, ok := resolveFactoryTemplate(version.Nodes)
	if !ok {
		return "", false, nil
	}
	return resolved.id, true, nil
}
