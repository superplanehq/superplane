package grpc

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/ai"
	"github.com/superplanehq/superplane/pkg/authorization"
	scripts "github.com/superplanehq/superplane/pkg/grpc/actions/scripts"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/scripts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PluginActivator interface {
	ActivateScript(name string, source string, manifestJSON []byte) error
	DeactivateScript(name string) error
}

type ScriptsService struct {
	pb.UnimplementedScriptsServer
	aiClient       *ai.Client
	pluginManager  PluginActivator
}

func NewScriptsService(aiClient *ai.Client, pluginManager PluginActivator) *ScriptsService {
	return &ScriptsService{aiClient: aiClient, pluginManager: pluginManager}
}

func (s *ScriptsService) ListScripts(ctx context.Context, req *pb.ListScriptsRequest) (*pb.ListScriptsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return scripts.ListScripts(ctx, organizationID)
}

func (s *ScriptsService) DescribeScript(ctx context.Context, req *pb.DescribeScriptRequest) (*pb.DescribeScriptResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return scripts.DescribeScript(ctx, organizationID, req.Id)
}

func (s *ScriptsService) CreateScript(ctx context.Context, req *pb.CreateScriptRequest) (*pb.CreateScriptResponse, error) {
	if req.Script == nil {
		return nil, status.Error(codes.InvalidArgument, "script is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return scripts.CreateScript(ctx, organizationID, req.Script)
}

func (s *ScriptsService) UpdateScript(ctx context.Context, req *pb.UpdateScriptRequest) (*pb.UpdateScriptResponse, error) {
	if req.Script == nil {
		return nil, status.Error(codes.InvalidArgument, "script is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	resp, err := scripts.UpdateScript(ctx, organizationID, req.Id, req.Script)
	if err != nil {
		return nil, err
	}

	if s.pluginManager != nil && req.Script.Status != "" {
		s.handleScriptStatusChange(resp.Script)
	}

	return resp, nil
}

func (s *ScriptsService) handleScriptStatusChange(script *pb.Script) {
	if script == nil {
		log.Warn("handleScriptStatusChange: script is nil")
		return
	}

	log.Infof("handleScriptStatusChange: name=%s, status=%s, sourceLen=%d, pluginManager=%v",
		script.Name, script.Status, len(script.Source), s.pluginManager != nil)

	switch script.Status {
	case models.ScriptStatusActive:
		manifest := []byte(script.ManifestJson)
		if len(manifest) == 0 {
			manifest = []byte("{}")
		}
		if err := s.pluginManager.ActivateScript(script.Name, script.Source, manifest); err != nil {
			log.WithError(err).Errorf("Failed to activate script %s", script.Name)
		} else {
			log.Infof("Script %s activated via API", script.Name)
		}
	case models.ScriptStatusDraft:
		if err := s.pluginManager.DeactivateScript(script.Name); err != nil {
			log.WithError(err).Errorf("Failed to deactivate script %s", script.Name)
		}
	default:
		log.Infof("handleScriptStatusChange: unhandled status %s for script %s", script.Status, script.Name)
	}
}

func (s *ScriptsService) DeleteScript(ctx context.Context, req *pb.DeleteScriptRequest) (*pb.DeleteScriptResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)

	if s.pluginManager != nil {
		existing, err := models.FindScript(organizationID, req.Id)
		if err == nil && existing.Status == models.ScriptStatusActive {
			if err := s.pluginManager.DeactivateScript(existing.Name); err != nil {
				log.WithError(err).Errorf("Failed to deactivate script %s before deletion", existing.Name)
			}
		}
	}

	return scripts.DeleteScript(ctx, organizationID, req.Id)
}

func (s *ScriptsService) GenerateScript(ctx context.Context, req *pb.GenerateScriptRequest) (*pb.GenerateScriptResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return scripts.GenerateScript(ctx, s.aiClient, organizationID, req.ScriptId, req.Message)
}
