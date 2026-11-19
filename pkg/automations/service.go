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
		adopted := make(map[string]bool, len(sources))
		for _, source := range sources {
			result = append(result, summarize(&source))
			adopted[source.ArtifactID] = true
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
		if legacy != nil {
			record.Source, err = s.Backend.(LegacyImporter).ImportLegacy(ctx, tx, *legacy)
			if err != nil {
				return err
			}
		}
		result = fromRecord(record)
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
		return s.check(ctx, tx, placement, doc)
	})
	return doc, err
}

func (s Service) Create(ctx context.Context, placement Placement, actor uuid.UUID, source string) (Automation, error) {
	record := &models.AutomationSource{
		ID: uuid.New(), OrganizationID: placement.OrganizationID, FactoryID: placement.FactoryID, Stage: placement.Stage,
	}
	return s.save(ctx, actor, source, record, false)
}

func (s Service) Update(ctx context.Context, organizationID, id, actor uuid.UUID, source string) (Automation, error) {
	return s.save(ctx, actor, source, &models.AutomationSource{ID: id, OrganizationID: organizationID}, true)
}

func (s Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	return write(ctx, func(tx *gorm.DB) (func(), error) {
		if err := requireFeatures(tx, organizationID); err != nil {
			return nil, err
		}
		record, _, err := s.lockExisting(ctx, tx, organizationID, id)
		if err != nil {
			return nil, err
		}
		notify, err := s.Backend.Remove(ctx, tx, placementOf(record), artifactOf(record))
		if err != nil {
			return nil, err
		}
		if record.Revision == 0 {
			return notify, nil
		}
		return notify, record.Delete(tx)
	})
}

func (s Service) save(ctx context.Context, actor uuid.UUID, source string, record *models.AutomationSource, updating bool) (Automation, error) {
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
			if legacy != nil {
				if _, err := s.Backend.(LegacyImporter).ImportLegacy(ctx, tx, *legacy); err != nil {
					return nil, err
				}
			}
		} else if _, err := models.LockAutomationFactory(tx, record.OrganizationID, record.FactoryID); err != nil {
			return nil, factoryError(err)
		}

		placement := placementOf(record)
		if err := s.check(ctx, tx, placement, doc); err != nil {
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
	return fromRecord(record), nil
}

func (s Service) check(ctx context.Context, tx *gorm.DB, placement Placement, doc *automationspec.Document) error {
	if err := automationspec.ValidateStage(doc, automationspec.Stage(placement.Stage)); err != nil {
		return err
	}

	for i, step := range doc.Steps {
		for j, name := range step.Secrets {
			_, err := models.FindSecretByNameInTransaction(tx, models.DomainTypeOrganization, placement.OrganizationID, name)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return doc.Diagnostic(fmt.Sprintf("steps[%d].secrets[%d]", i, j), fmt.Sprintf("Create the organization secret %q or select an existing secret.", name))
			}
			if err != nil {
				return err
			}
		}
	}

	return s.Backend.Check(ctx, tx, placement, doc)
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
		Stage: legacy.Placement.Stage, Name: legacy.Name, ArtifactID: legacy.Artifact.ID, ArtifactRevision: legacy.Artifact.Revision,
	}, &legacy, nil
}

func requireFeatures(tx *gorm.DB, organizationID uuid.UUID) error {
	organization, err := models.FindOrganizationByIDInTransaction(tx, organizationID.String())
	if err != nil {
		return err
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryCustomAutomations) ||
		!organization.HasExperimentalFeature(features.FeatureAutomationsAsCode) {
		return grpcerrors.NotFound(gorm.ErrRecordNotFound, "Enable Custom Automations and Automations as code for this organization.")
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

func placementOf(record *models.AutomationSource) Placement {
	return Placement{record.OrganizationID, record.FactoryID, record.Stage}
}

func artifactOf(record *models.AutomationSource) Artifact {
	return Artifact{record.ArtifactID, record.ArtifactRevision}
}

func summarize(record *models.AutomationSource) Summary {
	return Summary{record.ID, record.FactoryID, record.Stage, record.Name}
}

func fromRecord(record *models.AutomationSource) Automation {
	return Automation{summarize(record), record.Source, record.Revision}
}
