package factories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func renderFactoryConfigurationCustom(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, placement string, authored *models.AutomationSource) (string, error) {
	placed := *canvas
	placed.ColumnKey = &placement
	source, err := importAutomationCanvas(tx, factory, &placed, version)
	if err != nil {
		return "", fmt.Errorf("automation %q: %w", canvas.Name, err)
	}
	if authored == nil || authored.FactoryID != factory.ID || authored.ArtifactRevision != version.ID.String() {
		return source, nil
	}
	document, err := automationspec.Parse(authored.Source)
	if err != nil || document.Name != canvas.Name || validateAutomationCanvasPlacement(document, placement) != nil {
		return source, nil
	}
	// The matching revision preserves authored inheritance and comments.
	return authored.Source, nil
}

func validateFactoryConfigurationCustom(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, target *factoryConfigurationTarget, current *factoryConfigurationAutomationTarget, placement, source string) (*automationspec.Document, error) {
	document, err := automationspec.Parse(source)
	if err != nil {
		return nil, err
	}
	if err := validateAutomationCanvasPlacement(document, placement); err != nil {
		return nil, err
	}
	factory := target.factory
	if current == nil {
		existing, err := target.findCanvasByName(tx, document.Name)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, document.Diagnostic("name", "Use a unique automation name in this workspace.")
		}
	}
	changed, err := factoryConfigurationCustomChanged(current, document)
	if err != nil {
		return nil, err
	}
	if !changed {
		return document, nil
	}
	// Configuration requests check feature access before loading their read inputs.
	if factoryAutomationReads(tx, factory) == nil {
		feature := features.FeatureFactoryAsCode
		enabled, err := models.OrganizationHasExperimentalFeatures(tx, factory.OrganizationID, feature)
		if err != nil {
			return nil, err
		}
		if !enabled {
			return nil, document.Diagnostic("$", fmt.Sprintf("Enable %s before changing this automation.", feature))
		}
	}

	if err := target.validateCustomSecrets(tx, document); err != nil {
		return nil, err
	}
	var version *models.CanvasVersion
	if current != nil {
		version = current.version
	}
	_, _, err = (CanvasAutomationBackend{Deps: deps}).compile(tx, automations.Placement{
		OrganizationID: factory.OrganizationID, FactoryID: factory.ID, Stage: placement,
	}, document, version, versionOwnerID(ctx))
	if err != nil {
		return nil, err
	}
	return document, nil
}

func factoryConfigurationCustomChanged(current *factoryConfigurationAutomationTarget, desired *automationspec.Document) (bool, error) {
	if current == nil {
		return true, nil
	}
	previous, err := automationspec.Parse(current.source)
	if err != nil {
		return false, err
	}
	previous.Definition.Name = desired.Name
	return !equalCanvasJSON(previous.Definition, desired.Definition), nil
}

func configurationAutomationSourceID(factoryID uuid.UUID, placement, id string) uuid.UUID {
	return uuid.NewSHA1(factoryID, []byte("factory-configuration\x00"+placement+"\x00"+id))
}

func createFactoryConfigurationCustom(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, target *factoryConfigurationTarget, placement, manifestID, source string, actor uuid.UUID) (func(), error) {
	document, err := validateFactoryConfigurationCustom(ctx, tx, deps, target, nil, placement, source)
	if err != nil {
		return nil, err
	}
	factory := target.factory
	id := uuid.New()
	artifact, notify, err := (CanvasAutomationBackend{Deps: deps}).Apply(ctx, tx, automations.Placement{
		OrganizationID: factory.OrganizationID, FactoryID: factory.ID, Stage: placement,
	}, actor, id, automations.Artifact{}, document)
	if err != nil {
		return nil, err
	}
	authored := models.AutomationSource{
		ID:             configurationAutomationSourceID(factory.ID, placement, manifestID),
		OrganizationID: factory.OrganizationID, FactoryID: factory.ID,
		Name: document.Name, Source: source, Revision: 1,
		ArtifactID: artifact.ID, ArtifactRevision: artifact.Revision,
	}
	// Validation allows Save to replace a source whose previous canvas was deleted.
	if err := authored.Save(tx); err != nil {
		return nil, err
	}
	return notify, nil
}

func renameFactoryConfigurationAutomation(tx *gorm.DB, factory *models.Factory, current *factoryConfigurationAutomationTarget, desired factoryConfigurationAutomation, manifest *factoryConfigurationManifest) error {
	if err := canvases.UpdateCanvasInTransaction(tx, current.canvas, &desired.Name, nil); err != nil {
		return err
	}
	if current.templateID != "" {
		return nil
	}
	source := manifest.Sources[configurationAutomationPath(desired.Uses)]
	document, err := automationspec.Parse(source)
	if err != nil {
		return err
	}
	return saveFactoryConfigurationCustomSource(tx, factory, current.canvas, source, document)
}

func validateFactoryConfigurationCustomSecrets(tx *gorm.DB, organizationID uuid.UUID, document *automationspec.Document) error {
	names := make([]string, 0)
	for _, step := range document.Steps {
		names = append(names, step.Secrets...)
	}
	existing, err := models.FindExistingSecretNames(tx, models.DomainTypeOrganization, organizationID, names)
	if err != nil {
		return err
	}
	found := make(map[string]bool, len(existing))
	for _, name := range existing {
		found[name] = true
	}
	return checkFactoryConfigurationCustomSecrets(document, found)
}

func checkFactoryConfigurationCustomSecrets(document *automationspec.Document, found map[string]bool) error {
	for i, step := range document.Steps {
		for j, name := range step.Secrets {
			if !found[name] {
				return document.Diagnostic(fmt.Sprintf("steps[%d].secrets[%d]", i, j), fmt.Sprintf("Create the organization secret %q or select an existing secret.", name))
			}
		}
	}
	return nil
}

func applyFactoryConfigurationCustom(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, target *factoryConfigurationTarget, previous *factoryConfigurationAutomationTarget, source string, actor uuid.UUID) (func(), error) {
	factory := target.factory
	version, placement := previous.version, previous.placement
	current, err := models.LockCanvasForUpdate(tx, factory.OrganizationID, previous.canvas.ID)
	if err != nil {
		return nil, err
	}
	if current.FactoryID == nil || *current.FactoryID != factory.ID || current.LiveVersionID == nil || *current.LiveVersionID != version.ID {
		return nil, grpcerrors.FailedPrecondition(nil, "The automation changed. Run plan again before apply.")
	}
	document, err := validateFactoryConfigurationCustom(ctx, tx, deps, target, previous, placement, source)
	if err != nil {
		return nil, err
	}
	nodes, edges, err := (CanvasAutomationBackend{Deps: deps}).compile(tx, automations.Placement{
		OrganizationID: factory.OrganizationID, FactoryID: factory.ID, Stage: placement,
	}, document, version, actor)
	if err != nil {
		return nil, err
	}
	if err := canvases.PublishGeneratedCanvasNodes(ctx, tx, current, actor, "Update custom automation", nodes, edges, intakePublisherOptions(deps, factory.OrganizationID)); err != nil {
		return nil, err
	}
	if err := canvases.UpdateCanvasInTransaction(tx, current, &document.Name, nil); err != nil {
		return nil, automationCanvasWriteError(document, err)
	}
	if err := saveFactoryConfigurationCustomSource(tx, factory, current, source, document); err != nil {
		return nil, err
	}
	return func() { canvases.PublishCanvasUpdated(current) }, nil
}

func saveFactoryConfigurationCustomSource(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, source string, document *automationspec.Document) error {
	authored, err := findLegacyAutomationSource(tx, canvas)
	if err != nil {
		return err
	}
	if authored == nil {
		authored = &models.AutomationSource{ID: canvas.ID, OrganizationID: factory.OrganizationID, FactoryID: factory.ID, ArtifactID: canvas.ID.String()}
	}
	live, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return err
	}
	if err := checkAutomationCanvasNodes(document, live.Nodes); err != nil {
		return err
	}
	authored.Name = document.Name
	authored.Source = source
	authored.Revision++
	authored.ArtifactRevision = live.ID.String()
	return authored.Save(tx)
}
