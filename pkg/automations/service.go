package automations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type Summary struct {
	ID        uuid.UUID `json:"id"`
	FactoryID uuid.UUID `json:"factory_id"`
	Stage     string    `json:"stage"`
	Name      string    `json:"name"`
}

type Automation struct {
	Summary
	YAML     string `json:"yaml"`
	Revision int64  `json:"revision"`
}

type Service struct {
	Backend Backend
}

func (s Service) List(ctx context.Context, placement Placement) ([]Summary, error) {
	result := []Summary{}
	err := readSnapshot(ctx, func(tx *gorm.DB) error {
		if err := requireFeatures(tx, placement.OrganizationID); err != nil {
			return err
		}
		if _, err := models.FindFactory(tx, placement.OrganizationID, placement.FactoryID); err != nil {
			return factoryError(err)
		}

		sources, err := models.ListAutomationSources(tx, placement.OrganizationID, placement.FactoryID)
		if err != nil {
			return err
		}
		artifactIDs := make([]string, 0, len(sources))
		adopted := make(map[string]bool, len(sources))
		for _, source := range sources {
			artifactIDs = append(artifactIDs, source.ArtifactID)
			adopted[source.ArtifactID] = true
		}
		stages, err := s.Backend.ArtifactStages(tx, placement, artifactIDs)
		if err != nil {
			return err
		}
		for _, source := range sources {
			if stage, ok := stages[source.ArtifactID]; ok {
				result = append(result, summarize(&source, stage))
			}
		}

		if importer, ok := s.Backend.(LegacyImporter); ok {
			legacy, err := importer.ListLegacy(ctx, tx, placement)
			if err != nil {
				return err
			}
			for _, item := range legacy {
				if !adopted[item.Artifact.ID] {
					result = append(result, Summary{item.ID, placement.FactoryID, item.Placement.Stage, item.Name})
				}
			}
		}

		slices.SortFunc(result, func(a, b Summary) int {
			if order := strings.Compare(a.Name, b.Name); order != 0 {
				return order
			}
			return strings.Compare(a.ID.String(), b.ID.String())
		})
		return nil
	})
	return result, err
}

func (s Service) Read(ctx context.Context, organizationID, id uuid.UUID) (Automation, error) {
	var result Automation
	err := readSnapshot(ctx, func(tx *gorm.DB) error {
		if err := requireFeatures(tx, organizationID); err != nil {
			return err
		}
		record, legacy, err := s.find(ctx, tx, organizationID, id)
		if err != nil {
			return err
		}
		if _, err := models.FindFactory(tx, organizationID, record.FactoryID); err != nil {
			return factoryError(err)
		}
		placement, err := s.artifactPlacement(tx, record, legacy)
		if err != nil {
			return err
		}
		if legacy != nil {
			record.Source, err = s.Backend.(LegacyImporter).ImportLegacy(ctx, tx, *legacy)
			if err != nil {
				return err
			}
		}
		result = fromRecord(record, placement.Stage)
		return nil
	})
	return result, err
}

func (s Service) Validate(ctx context.Context, placement Placement, source string) (*automationspec.Document, error) {
	var doc *automationspec.Document
	err := readSnapshot(ctx, func(tx *gorm.DB) error {
		if err := requireFeatures(tx, placement.OrganizationID); err != nil {
			return err
		}
		var err error
		doc, err = automationspec.Parse(source)
		if err != nil {
			return err
		}
		if _, err := models.FindFactory(tx, placement.OrganizationID, placement.FactoryID); err != nil {
			return factoryError(err)
		}
		if err := checkAutomationReferences(tx, placement, doc); err != nil {
			return err
		}
		return s.Backend.Check(ctx, tx, placement, doc)
	})
	return doc, err
}

func (s Service) Create(ctx context.Context, placement Placement, actor uuid.UUID, source string) (Automation, error) {
	return s.save(ctx, actor, uuid.New(), placement, source, false)
}

func (s Service) Update(ctx context.Context, organizationID, id, actor uuid.UUID, source string) (Automation, error) {
	return s.save(ctx, actor, id, Placement{OrganizationID: organizationID}, source, true)
}

func (s Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	return write(ctx, func(tx *gorm.DB) (func(), error) {
		if err := requireFeatures(tx, organizationID); err != nil {
			return nil, err
		}
		record, legacy, err := s.lockExisting(ctx, tx, organizationID, id)
		if err != nil {
			return nil, err
		}
		placement, err := s.artifactPlacement(tx, record, legacy)
		if err != nil {
			return nil, err
		}
		notify, err := s.Backend.Remove(ctx, tx, placement, artifactOf(record))
		if err != nil {
			return nil, err
		}
		if record.Revision == 0 {
			return notify, nil
		}
		return notify, record.Delete(tx)
	})
}

func (s Service) save(ctx context.Context, actor, id uuid.UUID, placement Placement, source string, updating bool) (Automation, error) {
	record := &models.AutomationSource{ID: id, OrganizationID: placement.OrganizationID, FactoryID: placement.FactoryID}
	err := write(ctx, func(tx *gorm.DB) (func(), error) {
		if err := requireFeatures(tx, record.OrganizationID); err != nil {
			return nil, err
		}
		doc, err := automationspec.Parse(source)
		if err != nil {
			return nil, err
		}
		if updating {
			var legacy *LegacyAutomation
			record, legacy, err = s.lockExisting(ctx, tx, record.OrganizationID, record.ID)
			if err != nil {
				return nil, err
			}
			placement, err = s.artifactPlacement(tx, record, legacy)
			if err != nil {
				return nil, err
			}
			if legacy != nil {
				if _, err := s.Backend.(LegacyImporter).ImportLegacy(ctx, tx, *legacy); err != nil {
					return nil, err
				}
			}
		} else if _, err := models.LockAutomationFactory(tx, record.OrganizationID, record.FactoryID); err != nil {
			return nil, factoryError(err)
		}

		if err := checkAutomationReferences(tx, placement, doc); err != nil {
			return nil, err
		}
		artifact, notify, err := s.Backend.Apply(ctx, tx, placement, actor, record.ID, artifactOf(record), doc)
		if err != nil {
			return nil, err
		}

		record.Name = strings.TrimSpace(doc.Name)
		record.Source = source
		record.Revision++
		record.ArtifactID = artifact.ID
		record.ArtifactRevision = artifact.Revision
		return notify, record.Save(tx)
	})
	if err != nil {
		return Automation{}, err
	}
	return fromRecord(record, placement.Stage), nil
}

func checkAutomationReferences(tx *gorm.DB, placement Placement, doc *automationspec.Document) error {
	if strings.HasPrefix(placement.Stage, "custom-") {
		factory, err := models.FindFactory(tx, placement.OrganizationID, placement.FactoryID)
		if err != nil {
			return factoryError(err)
		}
		lines, err := factory.ListLines(tx)
		if err != nil {
			return err
		}
		found := slices.ContainsFunc(lines, func(line models.FactoryLine) bool {
			return slices.ContainsFunc(line.StagesValue(), func(stage models.FactoryLineStage) bool {
				return stage.Key == placement.Stage && stage.Kind == models.FactoryLineStagePhase
			})
		})
		if !found {
			return doc.Diagnostic("$", "Select a custom stage in this factory.")
		}
	}
	if err := automationspec.ValidateStage(doc, automationspec.Stage(placement.Stage)); err != nil {
		return err
	}

	var names []string
	secrets := make(map[string]bool)
	for _, step := range doc.Steps {
		for _, name := range step.Secrets {
			if _, seen := secrets[name]; !seen {
				names = append(names, name)
				secrets[name] = false
			}
		}
	}
	existing, err := models.FindExistingSecretNames(tx, models.DomainTypeOrganization, placement.OrganizationID, names)
	if err != nil {
		return err
	}
	for _, name := range existing {
		secrets[name] = true
	}
	for i, step := range doc.Steps {
		for j, name := range step.Secrets {
			if !secrets[name] {
				return doc.Diagnostic(fmt.Sprintf("steps[%d].secrets[%d]", i, j), fmt.Sprintf("Create the organization secret %q or select an existing secret.", name))
			}
		}
	}

	return nil
}

func (s Service) lockExisting(ctx context.Context, tx *gorm.DB, organizationID, id uuid.UUID) (*models.AutomationSource, *LegacyAutomation, error) {
	record, _, err := s.find(ctx, tx, organizationID, id)
	if err != nil {
		return nil, nil, err
	}
	if _, err := models.LockAutomationFactory(tx, organizationID, record.FactoryID); err != nil {
		return nil, nil, factoryError(err)
	}
	// Re-read after obtaining the lock. A competing save can adopt a legacy
	// automation or publish a new revision while this transaction waits.
	return s.find(ctx, tx, organizationID, id)
}

func (s Service) find(ctx context.Context, tx *gorm.DB, organizationID, id uuid.UUID) (*models.AutomationSource, *LegacyAutomation, error) {
	record, err := models.FindAutomationSource(tx, organizationID, id)
	if err == nil {
		return record, nil, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}
	importer, ok := s.Backend.(LegacyImporter)
	if !ok {
		return nil, nil, grpcerrors.NotFound(gorm.ErrRecordNotFound, "Automation not found.")
	}
	legacy, err := importer.FindLegacy(ctx, tx, organizationID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, grpcerrors.NotFound(gorm.ErrRecordNotFound, "Automation not found.")
	}
	if err != nil {
		return nil, nil, err
	}
	return &models.AutomationSource{
		ID: legacy.ID, OrganizationID: organizationID, FactoryID: legacy.Placement.FactoryID,
		Name: legacy.Name, ArtifactID: legacy.Artifact.ID, ArtifactRevision: legacy.Artifact.Revision,
	}, &legacy, nil
}

func (s Service) artifactPlacement(tx *gorm.DB, record *models.AutomationSource, legacy *LegacyAutomation) (Placement, error) {
	if legacy != nil {
		return legacy.Placement, nil
	}
	placement := Placement{OrganizationID: record.OrganizationID, FactoryID: record.FactoryID}
	stages, err := s.Backend.ArtifactStages(tx, placement, []string{record.ArtifactID})
	if err != nil {
		return Placement{}, err
	}
	stage, ok := stages[record.ArtifactID]
	if !ok {
		return Placement{}, grpcerrors.NotFound(gorm.ErrRecordNotFound, "Automation not found.")
	}
	placement.Stage = stage
	return placement, nil
}

func requireFeatures(tx *gorm.DB, organizationID uuid.UUID) error {
	organization, err := models.FindOrganizationByIDInTransaction(tx, organizationID.String())
	if err != nil {
		return err
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryAsCode) {
		return grpcerrors.NotFound(gorm.ErrRecordNotFound, "Enable Factory as code for this organization.")
	}
	return nil
}

func readSnapshot(ctx context.Context, read func(*gorm.DB) error) error {
	return database.DB(ctx).Transaction(read, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
}

func write(ctx context.Context, operation func(*gorm.DB) (func(), error)) error {
	var notify func()
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		notify, err = operation(tx)
		return err
	})
	if err == nil && notify != nil {
		notify()
	}
	return err
}

func factoryError(err error) error {
	if errors.Is(err, models.ErrFactoryNotFound) {
		return grpcerrors.NotFound(err, "Factory not found.")
	}
	return err
}

func artifactOf(record *models.AutomationSource) Artifact {
	return Artifact{record.ArtifactID, record.ArtifactRevision}
}

func summarize(record *models.AutomationSource, stage string) Summary {
	return Summary{record.ID, record.FactoryID, stage, record.Name}
}

func fromRecord(record *models.AutomationSource, stage string) Automation {
	return Automation{summarize(record, stage), record.Source, record.Revision}
}
