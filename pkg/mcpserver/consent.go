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
      <svg width="24" height="27" viewBox="180 135 720 810" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path fill-rule="evenodd" d="M543.228 153.905C626.85 108.604 728.51 169.153 728.511 264.256V392.792C728.511 399.008 728.04 405.156 727.144 411.195C805.606 385.932 890.542 443.979 890.542 530.912V659.447C890.541 705.5 865.32 747.86 824.824 769.798L536.299 926.102C452.678 971.4 351.017 910.853 351.016 815.75V687.215C351.016 680.996 351.482 674.845 352.378 668.802C273.918 694.061 188.986 636.026 188.984 549.095V420.56C188.985 374.504 214.213 332.146 254.707 310.209L543.228 153.905ZM402.646 671.62C401.575 676.69 401.016 681.911 401.016 687.215V815.75C401.017 872.963 462.174 909.387 512.48 882.137L654.951 804.95C647.141 803.568 639.333 800.898 631.777 796.771L402.646 671.62ZM840.542 530.912C840.542 473.696 779.384 437.272 729.077 464.525L593.306 538.07L677.759 584.198C718.042 606.201 743.101 648.444 743.101 694.344V730.731C743.101 741.47 740.933 751.471 737.09 760.453L801.006 725.834C825.365 712.637 840.541 687.155 840.542 659.447V530.912ZM429.683 629.418L655.742 752.889C672.57 762.08 693.1 758.895 693.101 739.723V694.344C693.101 666.73 678.027 641.317 653.794 628.08L543.423 567.796L429.683 629.418ZM278.525 354.173C254.163 367.371 238.985 392.854 238.984 420.56V549.095C238.986 606.308 300.142 642.729 350.449 615.477L489.658 540.057L416.729 500.545C376.234 478.608 351.006 436.25 351.006 390.194V353.71C351.007 334.846 357.679 318.249 368.423 305.467L278.525 354.173ZM438.286 331.507C421.462 322.394 401.007 325.778 401.006 344.912V390.194C401.006 417.9 416.185 443.382 440.547 456.581L539.653 510.267L653.955 448.348L438.286 331.507ZM678.511 264.256C678.51 207.042 617.352 170.618 567.046 197.87L417.485 278.89C432.1 277.13 447.558 279.662 462.104 287.543L677.617 404.295C678.2 400.523 678.511 396.68 678.511 392.792V264.256Z" fill="currentColor"/>
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
