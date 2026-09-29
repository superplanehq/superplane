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
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      --radius: 0.5rem;
      --workspace-page-title-size: 22px;
      --workspace-page-title-line-height: 33px;
      --workspace-page-title-tracking: -0.02em;
      --workspace-body-size: 13px;
      --workspace-body-line-height: 19.5px;
      --workspace-label-size: 11px;
      --workspace-label-line-height: 16.5px;
      --workspace-label-tracking: 0.04em;
      --background: #ffffff;
      --foreground: #26251e;
      --card: #ffffff;
      --primary: #26251e;
      --primary-foreground: #ffffff;
      --muted: #f7f7f7;
      --muted-foreground: #737373;
      --border: #e5e5e5;
      --field: #ffffff;
      --ring: #a3a3a3;
      --destructive: #e5484d;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --background: #14120b;
        --foreground: #edecec;
        --card: #1b1913;
        --primary: #f7f7f4;
        --primary-foreground: #26251e;
        --muted: #26251e;
        --muted-foreground: #a3a3a3;
        --border: #33312b;
        --field: #221f18;
        --ring: #8a8578;
        --destructive: #fca5a5;
      }
    }
    * { box-sizing: border-box; }
    html, body { margin: 0; min-height: 100%; }
    body {
      font-family: "Inter", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      font-feature-settings: "cv02", "cv03", "cv04", "cv11";
      background: var(--muted);
      color: var(--foreground);
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 2rem 1rem;
    }
    .card {
      width: 100%;
      max-width: 26.5rem;
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 2rem;
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 0.625rem;
      color: var(--foreground);
      margin-bottom: 1.5rem;
    }
    .brand svg { display: block; }
    .brand-name {
      font-size: 13px;
      font-weight: 600;
      letter-spacing: -0.01em;
    }
    h1 {
      margin: 0;
      font-size: var(--workspace-page-title-size);
      font-weight: 600;
      letter-spacing: var(--workspace-page-title-tracking);
      line-height: var(--workspace-page-title-line-height);
    }
    .lead {
      margin: 0.5rem 0 0;
      color: var(--muted-foreground);
      font-size: var(--workspace-body-size);
      line-height: var(--workspace-body-line-height);
    }
    .error {
      margin: 1rem 0 0;
      color: var(--destructive);
      font-size: var(--workspace-body-size);
      line-height: var(--workspace-body-line-height);
    }
    form { margin-top: 1.5rem; }
    label {
      display: block;
      color: var(--muted-foreground);
      font-size: var(--workspace-label-size);
      font-weight: 500;
      letter-spacing: var(--workspace-label-tracking);
      line-height: var(--workspace-label-line-height);
      text-transform: uppercase;
    }
    select {
      display: block;
      width: 100%;
      margin-top: 0.375rem;
      padding: 0.5rem 0.75rem;
      background: var(--field);
      color: var(--foreground);
      border: 1px solid var(--border);
      border-radius: calc(var(--radius) - 2px);
      font: inherit;
      font-size: var(--workspace-body-size);
      line-height: var(--workspace-body-line-height);
    }
    select:focus {
      outline: none;
      border-color: var(--ring);
    }
    button {
      display: block;
      width: 100%;
      margin-top: 1.25rem;
      padding: 0.625rem 0.875rem;
      background: var(--primary);
      color: var(--primary-foreground);
      border: 0;
      border-radius: calc(var(--radius) - 2px);
      font: inherit;
      font-size: var(--workspace-body-size);
      font-weight: 500;
      line-height: var(--workspace-body-line-height);
      cursor: pointer;
    }
    button:hover { opacity: 0.92; }
    .empty {
      margin: 1.5rem 0 0;
      color: var(--muted-foreground);
      font-size: var(--workspace-body-size);
      line-height: var(--workspace-body-line-height);
    }
  </style>
</head>
<body>
  <main class="card">
    <div class="brand">
      <svg width="29" height="21" viewBox="0 0 29 21" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path d="M29 14.32C28.5655 14.7542 28.1107 15.1686 27.6379 15.5626L17.112 5.16914L22.7409 18.5877C22.1782 18.8335 21.6021 19.0552 21.0139 19.2508L15.4286 5.93611V20.2791C15.1216 20.2926 14.8129 20.3 14.5025 20.3L14.2391 20.2982C14.0172 20.2955 13.7961 20.2888 13.5759 20.2791V5.88893L7.97322 19.2446C7.38516 19.0484 6.80914 18.8264 6.24652 18.58L11.8656 5.18536L1.36087 15.5578C0.888333 15.1638 0.434276 14.7489 0 14.3147L14.4975 0L29 14.32Z" fill="currentColor"/>
      </svg>
      <span class="brand-name">SuperPlane</span>
    </div>
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
    <p class="empty">This account has no workspace that this application can use.</p>
    {{end}}
  </main>
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

func OrganizationAllowsPublicMCP(orgID uuid.UUID) bool {
	enabled, err := models.HasExperimentalFeature(orgID, features.FeatureSuperPlaneMCPServer)
	return err == nil && enabled
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
		if !OrganizationAllowsPublicMCP(organization.ID) {
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
	if !OrganizationAllowsPublicMCP(factoryModel.OrganizationID) {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("workspace is required")
	}
	user, err := models.FindActiveHumanUserByAccountAndOrganization(database.DB(ctx), factoryModel.OrganizationID, account.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	return user.ID, factoryModel.OrganizationID, factoryModel.ID, nil
}
