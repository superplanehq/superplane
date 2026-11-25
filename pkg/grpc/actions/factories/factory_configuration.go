package factories

import (
	"bytes"
	"context"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func ExportFactoryConfiguration(ctx context.Context, organizationID string, req *pb.ExportFactoryConfigurationRequest) (*pb.ExportFactoryConfigurationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "Failed to export stage configuration.")
	}
	var response *pb.ExportFactoryConfigurationResponse
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := loadFactoryConfigurationTarget(tx, orgID, req.GetFactoryId(), req.GetLineId())
		if err != nil {
			return err
		}
		manifest := exportFactoryConfigurationManifest(target)
		var content bytes.Buffer
		encoder := yaml.NewEncoder(&content)
		encoder.SetIndent(2)
		if err := encoder.Encode(manifest); err != nil {
			return err
		}
		if err := encoder.Close(); err != nil {
			return err
		}
		files := []*pb.FactoryConfigurationFile{{Path: factoryConfigurationManifestPath, Content: content.String()}}
		paths := make([]string, 0, len(manifest.Sources))
		for name := range manifest.Sources {
			paths = append(paths, name)
		}
		slices.Sort(paths)
		for _, name := range paths {
			files = append(files, &pb.FactoryConfigurationFile{Path: name, Content: manifest.Sources[name]})
		}
		response = &pb.ExportFactoryConfigurationResponse{Coverage: factoryConfigurationCoverage, Bundle: &pb.FactoryConfigurationBundle{ManifestPath: factoryConfigurationManifestPath, Files: files}}
		return nil
	}, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, factoryErrorToStatus(err, "Failed to export stage configuration.")
	}
	return response, nil
}

func PlanFactoryConfiguration(ctx context.Context, deps IntakeDependencies, organizationID string, req *pb.PlanFactoryConfigurationRequest) (*pb.PlanFactoryConfigurationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "Failed to plan stage configuration.")
	}
	response := &pb.PlanFactoryConfigurationResponse{Coverage: factoryConfigurationCoverage}
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := loadFactoryConfigurationTarget(tx, orgID, req.GetFactoryId(), req.GetLineId())
		if err != nil {
			return err
		}
		manifest, diagnostic := parseFactoryConfigurationBundle(req.GetBundle())
		if diagnostic == nil {
			diagnostic, err = validateFactoryConfigurationManifest(ctx, tx, deps, manifest, target)
			if err != nil {
				return err
			}
		}
		if diagnostic != nil {
			response.Diagnostics = []*pb.FactoryConfigurationDiagnostic{diagnostic}
			return nil
		}
		response.Changes = compareFactoryConfiguration(target, manifest)
		response.BaseStateToken, err = factoryConfigurationToken(target, manifest)
		return err
	}, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, factoryErrorToStatus(err, "Failed to plan stage configuration.")
	}
	return response, nil
}

func ApplyFactoryConfiguration(ctx context.Context, deps IntakeDependencies, organizationID string, req *pb.ApplyFactoryConfigurationRequest) (*pb.ApplyFactoryConfigurationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "Failed to apply stage configuration.")
	}
	response := &pb.ApplyFactoryConfigurationResponse{Coverage: factoryConfigurationCoverage}
	var notifications []func()
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := requireFactoryAsCode(tx, orgID); err != nil {
			return err
		}
		target, err := lockFactoryConfigurationTarget(tx, orgID, req.GetFactoryId(), req.GetLineId())
		if err != nil {
			return err
		}
		actor := versionOwnerID(ctx)
		if actor == uuid.Nil || deps.AuthService == nil {
			return grpcerrors.PermissionDenied(nil, "Factory update permission is required.")
		}
		allowed, err := deps.AuthService.CheckOrganizationPermission(ctx, actor.String(), orgID.String(), "factories", "update")
		if err != nil {
			return err
		}
		if !allowed {
			return grpcerrors.PermissionDenied(nil, "Factory update permission is required.")
		}
		manifest, diagnostic := parseFactoryConfigurationBundle(req.GetBundle())
		if diagnostic == nil {
			diagnostic, err = validateFactoryConfigurationManifest(ctx, tx, deps, manifest, target)
			if err != nil {
				return err
			}
		}
		if diagnostic != nil {
			return grpcerrors.InvalidArgument(nil, fmt.Sprintf("%s: %s: %s", diagnostic.Path, diagnostic.Field, diagnostic.Message))
		}
		token, err := factoryConfigurationToken(target, manifest)
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(req.GetBaseStateToken()), []byte(token)) != 1 {
			return grpcerrors.FailedPrecondition(nil, "The configuration or requested input changed. Run plan again before apply.")
		}
		response.Changes = compareFactoryConfiguration(target, manifest)
		if len(response.Changes) == 0 {
			return nil
		}
		if err := applyFactoryConfigurationStages(tx, target, manifest); err != nil {
			return err
		}
		seen := map[uuid.UUID]bool{}
		byKey := configurationStagesByKey(target)
		for _, stage := range manifest.Stages {
			live := byKey[stage.Key]
			for _, current := range live.automations {
				retained := slices.ContainsFunc(stage.Automations, func(automation factoryConfigurationAutomation) bool {
					return configurationAutomationByIdentity(live, automation) == current
				})
				if retained || seen[current.canvas.ID] {
					continue
				}
				notify, err := removeFactoryConfigurationAutomation(tx, target, current)
				if err != nil {
					return err
				}
				seen[current.canvas.ID] = true
				notifications = append(notifications, notify)
			}
		}
		for _, stage := range manifest.Stages {
			live := byKey[stage.Key]
			for _, automation := range stage.Automations {
				current := configurationAutomationByIdentity(live, automation)
				if current == nil {
					source := manifest.Sources[configurationAutomationPath(automation.Uses)]
					notify, err := createFactoryConfigurationCustom(ctx, tx, deps, target.factory, configurationStagePlacement(live), automation.ID, source, actor)
					if err != nil {
						return err
					}
					notifications = append(notifications, notify)
					continue
				}
				if seen[current.canvas.ID] || !factoryConfigurationAutomationChanged(current, automation, manifest) {
					continue
				}
				seen[current.canvas.ID] = true
				if !factoryConfigurationAutomationContentChanged(current, automation, manifest) {
					if err := renameFactoryConfigurationAutomation(tx, target.factory, current, automation, manifest); err != nil {
						return err
					}
					notifications = append(notifications, func() { canvases.PublishCanvasUpdated(current.canvas) })
					continue
				}
				if current.templateID != "" {
					nodes, edges, err := copyFactoryConfigurationBuiltIn(current.version, current.defaults, current.templateID, automation.With)
					if err != nil {
						return err
					}
					if err := canvases.PublishGeneratedCanvasNodes(ctx, tx, current.canvas, actor, "Update stage automation settings", nodes, edges, intakePublisherOptions(deps, orgID)); err != nil {
						return err
					}
					if err := canvases.UpdateCanvasInTransaction(tx, current.canvas, &automation.Name, nil); err != nil {
						return err
					}
					notifications = append(notifications, func() { canvases.PublishCanvasUpdated(current.canvas) })
					continue
				}
				source := manifest.Sources[configurationAutomationPath(automation.Uses)]
				notify, err := applyFactoryConfigurationCustom(ctx, tx, deps, target.factory, current.canvas, current.version, current.placement, source, actor)
				if err != nil {
					return err
				}
				notifications = append(notifications, notify)
			}
		}
		response.Changed = true
		return nil
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "Failed to apply stage configuration.")
	}
	for _, notify := range notifications {
		if notify != nil {
			notify()
		}
	}
	return response, nil
}

func applyFactoryConfigurationStages(tx *gorm.DB, target *factoryConfigurationTarget, manifest *factoryConfigurationManifest) error {
	stages := make([]models.FactoryLineStage, 0, len(manifest.Stages))
	steps := append([]models.FactoryLineStep{}, target.line.Steps...)
	byKey := configurationStagesByKey(target)
	for _, stage := range manifest.Stages {
		current := byKey[stage.Key].stage
		current.Name = stage.Name
		stages = append(stages, current)
		if current.StepIndex != nil {
			steps[*current.StepIndex].MaxParallelism = stage.MaxParallelism
		}
	}
	if !factoryConfigurationStagesChanged(target, manifest) && equalCanvasJSON(steps, target.line.Steps) {
		return nil
	}
	return target.line.UpdateConfiguration(tx, nil, steps, nil, stages)
}
