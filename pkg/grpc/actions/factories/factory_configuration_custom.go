package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func exportFactoryConfigurationCustom(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, placement string) (string, error) {
	placed := *canvas
	placed.ColumnKey = &placement
	source, err := importAutomationCanvas(tx, factory, &placed, version)
	if err != nil {
		return "", fmt.Errorf("automation %q: %w", canvas.Name, err)
	}
	authored, err := findLegacyAutomationSource(tx, canvas)
	if err != nil {
		return "", err
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

func validateFactoryConfigurationCustom(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, placement, source string) (*automationspec.Document, error) {
	document, err := automationspec.Parse(source)
	if err != nil {
		return nil, err
	}
	if err := validateAutomationCanvasPlacement(document, placement); err != nil {
		return nil, err
	}
	if canvas == nil {
		_, err := models.FindCanvasByName(tx, factory.OrganizationID, &factory.ID, document.Name)
		if err == nil {
			return nil, document.Diagnostic("name", "Use a unique automation name in this workspace.")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if err := requireFactoryConfigurationCustomFeature(tx, factory, canvas, version, placement, document); err != nil {
		return nil, err
	}
	if err := validateFactoryConfigurationCustomSecrets(tx, factory.OrganizationID, document); err != nil {
		return nil, err
	}
	_, _, err = (CanvasAutomationBackend{Deps: deps}).compile(tx, automations.Placement{
		OrganizationID: factory.OrganizationID, FactoryID: factory.ID, Stage: placement,
	}, document, version, versionOwnerID(ctx))
	if err != nil {
		return nil, err
	}
	return document, nil
}

func requireFactoryConfigurationCustomFeature(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, placement string, desired *automationspec.Document) error {
	if canvas != nil {
		source, err := exportFactoryConfigurationCustom(tx, factory, canvas, version, placement)
		if err != nil {
			return err
		}
		current, err := automationspec.Parse(source)
		if err != nil {
			return err
		}
		current.Definition.Name = desired.Name
		if equalCanvasJSON(current.Definition, desired.Definition) {
			return nil
		}
	}
	feature := features.FeatureFactoryCustomAutomations
	if strings.HasPrefix(placement, "custom-") {
		feature = features.FeatureFactoryCustomStages
	}
	enabled, err := models.OrganizationHasExperimentalFeatures(tx, factory.OrganizationID, feature)
	if err != nil {
		return err
	}
	if !enabled {
		return desired.Diagnostic("$", fmt.Sprintf("Enable %s before changing this automation.", feature))
	}
	return nil
}

func configurationAutomationSourceID(factoryID uuid.UUID, placement, id string) uuid.UUID {
	return uuid.NewSHA1(factoryID, []byte("factory-configuration\x00"+placement+"\x00"+id))
}

func createFactoryConfigurationCustom(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, factory *models.Factory, placement, manifestID, source string, actor uuid.UUID) (func(), error) {
	document, err := validateFactoryConfigurationCustom(ctx, tx, deps, factory, nil, nil, placement, source)
	if err != nil {
		return nil, err
	}
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
	for i, step := range document.Steps {
		for j, name := range step.Secrets {
			if !found[name] {
				return document.Diagnostic(fmt.Sprintf("steps[%d].secrets[%d]", i, j), fmt.Sprintf("Create the organization secret %q or select an existing secret.", name))
			}
		}
	}
	return nil
}

func applyFactoryConfigurationCustom(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, placement, source string, actor uuid.UUID) (func(), error) {
	current, err := models.LockCanvasForUpdate(tx, factory.OrganizationID, canvas.ID)
	if err != nil {
		return nil, err
	}
	if current.FactoryID == nil || *current.FactoryID != factory.ID || current.LiveVersionID == nil || *current.LiveVersionID != version.ID {
		return nil, grpcerrors.FailedPrecondition(nil, "The automation changed. Run plan again before apply.")
	}
	document, err := validateFactoryConfigurationCustom(ctx, tx, deps, factory, current, version, placement, source)
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
