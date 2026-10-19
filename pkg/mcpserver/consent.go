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
	"gorm.io/gorm"
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
      <svg width="18.39" height="21" viewBox="0 0 712 813" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill-rule="evenodd" clip-rule="evenodd" d="M359.243 20.3306C442.865 -24.9703 544.526 35.5787 544.526 130.682V259.217C544.526 265.434 544.056 271.582 543.159 277.621C621.622 252.358 706.558 310.405 706.558 397.337V525.873C706.557 571.926 681.336 614.286 640.84 636.224L352.314 792.528C268.693 837.826 167.032 777.279 167.031 682.176V553.641C167.031 547.422 167.497 541.271 168.394 535.228C89.9332 560.487 5.00156 502.452 5 415.521V286.986C5.0004 240.93 30.2287 198.572 70.7227 176.634L359.243 20.3306ZM218.662 538.045C217.591 543.115 217.031 548.337 217.031 553.641V682.176C217.032 739.389 278.19 775.813 328.496 748.563L470.967 671.375C463.157 669.994 455.349 667.324 447.793 663.197L218.662 538.045ZM656.558 397.337C656.558 340.122 595.399 303.698 545.093 330.951L409.321 404.496L493.774 450.624C534.058 472.627 559.116 514.87 559.116 560.77V597.157C559.116 607.896 556.948 617.896 553.105 626.878L617.021 592.259C641.381 579.062 656.557 553.581 656.558 525.873V397.337ZM245.698 495.843L471.758 619.315C488.586 628.506 509.116 625.32 509.116 606.148V560.77C509.116 533.156 494.043 507.742 469.81 494.505L359.438 434.222L245.698 495.843ZM94.541 220.599C70.1788 233.797 55.0004 259.28 55 286.986V415.521C55.0016 472.733 116.158 509.155 166.465 481.903L305.674 406.483L232.744 366.971C192.25 345.034 167.022 302.676 167.021 256.62V220.135C167.022 201.272 173.695 184.674 184.438 171.893L94.541 220.599ZM254.302 197.933C237.478 188.82 217.023 192.204 217.021 211.338V256.62C217.022 284.326 232.2 309.808 256.562 323.006L355.669 376.693L469.971 314.774L254.302 197.933ZM494.526 130.682C494.526 73.4675 433.368 37.0432 383.062 64.2954L233.501 145.316C248.116 143.556 263.574 146.088 278.12 153.968L493.633 270.721C494.216 266.949 494.526 263.105 494.526 259.217V130.682Z" fill="currentColor"/>
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

func organizationAllowsFactories(orgID uuid.UUID) bool {
	enabled, err := models.HasExperimentalFeature(orgID, features.FeatureFactories)
	return err == nil && enabled
}

func UserAccountBlocked(tx *gorm.DB, userID uuid.UUID) bool {
	user, err := models.FindActiveUserByIDAnyOrg(tx, userID)
	if err != nil || user.AccountID == nil {
		return true
	}
	var account models.Account
	if err := tx.Where("id = ?", *user.AccountID).First(&account).Error; err != nil {
		return true
	}
	return account.IsBlocked()
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
		if !organizationAllowsFactories(organization.ID) {
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
	if !OrganizationAllowsPublicMCP(factoryModel.OrganizationID) || !organizationAllowsFactories(factoryModel.OrganizationID) {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("workspace is required")
	}
	user, err := models.FindActiveHumanUserByAccountAndOrganization(database.DB(ctx), factoryModel.OrganizationID, account.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	return user.ID, factoryModel.OrganizationID, factoryModel.ID, nil
}
