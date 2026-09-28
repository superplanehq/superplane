package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
)

type WorkspaceOption struct {
	FactoryID string
	Label     string
}

type consentPageData struct {
	ClientName string
	Consent    string
	Workspaces []WorkspaceOption
	Error      string
}

var consentPage = template.Must(template.New("mcp-consent").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Allow access to SuperPlane</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem auto; max-width: 32rem; color: #111; }
    label, select, button { display: block; width: 100%; }
    select, button { margin-top: 0.5rem; padding: 0.5rem; font: inherit; }
    button { margin-top: 1rem; }
    .lead { margin-bottom: 1.5rem; }
    .error { color: #8b0000; margin-bottom: 1rem; }
  </style>
</head>
<body>
  <h1>Allow access to SuperPlane</h1>
  <p class="lead">Choose the workspace {{.ClientName}} can use. The token applies to that workspace only.</p>
  {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
  {{if .Workspaces}}
  <form method="post" action="/oauth/authorize">
    <input type="hidden" name="consent" value="{{.Consent}}">
    <label for="factory_id">Workspace</label>
    <select id="factory_id" name="factory_id" required>
      {{range .Workspaces}}
      <option value="{{.FactoryID}}">{{.Label}}</option>
      {{end}}
    </select>
    <button type="submit">Allow access</button>
  </form>
  {{else}}
  <p>This account has no workspace.</p>
  {{end}}
</body>
</html>`))

func RenderConsentPage(clientName, consentToken, errMessage string, workspaces []WorkspaceOption) ([]byte, error) {
	name := strings.TrimSpace(clientName)
	if name == "" {
		name = "this application"
	}
	var buf bytes.Buffer
	err := consentPage.Execute(&buf, consentPageData{
		ClientName: name,
		Consent:    consentToken,
		Workspaces: workspaces,
		Error:      errMessage,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ListConsentWorkspaces(ctx context.Context, account *models.Account) ([]WorkspaceOption, error) {
	organizations, err := models.FindOrganizationsForAccount(account.Email)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	options := make([]WorkspaceOption, 0)
	for _, organization := range organizations {
		_, err := models.FindActiveHumanUserByAccountAndOrganization(db, organization.ID, account.ID)
		if err != nil {
			continue
		}
		enabled, err := models.HasExperimentalFeature(organization.ID, features.FeatureFactories)
		if err != nil || !enabled {
			continue
		}
		listed, err := models.ListFactories(db, organization.ID)
		if err != nil {
			return nil, err
		}
		for _, factory := range listed {
			options = append(options, WorkspaceOption{
				FactoryID: factory.ID.String(),
				Label:     strings.TrimSpace(organization.Name) + " / " + strings.TrimSpace(factory.Name),
			})
		}
	}
	return options, nil
}

func ResolveConsentWorkspace(ctx context.Context, account *models.Account, factoryID string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(factoryID))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("workspace is required")
	}
	factoryModel, err := models.FindFactoryByID(database.DB(ctx), id)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	user, err := models.FindActiveHumanUserByAccountAndOrganization(database.DB(ctx), factoryModel.OrganizationID, account.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	return user.ID, factoryModel.OrganizationID, factoryModel.ID, nil
}
