package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type OrganizationOption struct {
	OrganizationID string
	Name           string
}

type WorkspaceOption struct {
	FactoryID string
	Label     string
}

// ConsentView is the data for one step of the consent page. The page shows the
// workspace step when Organization is set. Otherwise it shows the organization
// step.
type ConsentView struct {
	ClientName    string
	ConsentToken  string
	Error         string
	Organizations []OrganizationOption
	Organization  *OrganizationOption
	Workspaces    []WorkspaceOption
}

func (v ConsentView) CanChangeOrganization() bool {
	return len(v.Organizations) > 1
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
    fieldset {
      margin: 0;
      padding: 0;
      border: 0;
      min-width: 0;
    }
    legend {
      padding: 0;
      color: var(--muted-foreground);
      font-size: var(--workspace-label-size);
      font-weight: 500;
      letter-spacing: var(--workspace-label-tracking);
      line-height: var(--workspace-label-line-height);
      text-transform: uppercase;
    }
    label.option {
      display: flex;
      align-items: center;
      gap: 0.625rem;
      margin-top: 0.375rem;
      padding: 0.5rem 0.75rem;
      background: var(--field);
      color: var(--foreground);
      border: 1px solid var(--border);
      border-radius: calc(var(--radius) - 2px);
      font-size: var(--workspace-body-size);
      font-weight: 400;
      letter-spacing: normal;
      line-height: var(--workspace-body-line-height);
      text-transform: none;
      cursor: pointer;
    }
    label.option input { margin: 0; }
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
    button.secondary {
      margin-top: 0.5rem;
      background: transparent;
      color: var(--foreground);
      border: 1px solid var(--border);
    }
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
    {{if .Organization}}
    <p class="lead">Select the workspaces that {{.ClientName}} can use in {{.Organization.Name}}. {{.ClientName}} can do only what you can do in these workspaces.</p>
    {{else}}
    <p class="lead">Choose the organization that {{.ClientName}} can use. In the next step, you select the workspaces.</p>
    {{end}}
    {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
    {{if .Organization}}
    <form method="post" action="/oauth/authorize">
      <input type="hidden" name="consent" value="{{.ConsentToken}}">
      <input type="hidden" name="organization_id" value="{{.Organization.OrganizationID}}">
      {{if .Workspaces}}
      <fieldset>
        <legend>Workspaces in {{.Organization.Name}}</legend>
        {{range .Workspaces}}
        <label class="option"><input type="checkbox" name="factory_id" value="{{.FactoryID}}"> {{.Label}}</label>
        {{end}}
      </fieldset>
      <button type="submit">Allow access</button>
      {{else}}
      <p class="empty">This organization has no workspace that this application can use.</p>
      {{end}}
      {{if .CanChangeOrganization}}
      <button type="submit" name="step" value="organizations" formnovalidate class="secondary">Change organization</button>
      {{end}}
    </form>
    {{else if .Organizations}}
    <form method="post" action="/oauth/authorize">
      <input type="hidden" name="consent" value="{{.ConsentToken}}">
      <input type="hidden" name="step" value="organization">
      <label for="organization_id">Organization</label>
      <select id="organization_id" name="organization_id" required>
        {{range .Organizations}}
        <option value="{{.OrganizationID}}">{{.Name}}</option>
        {{end}}
      </select>
      <button type="submit">Choose workspaces</button>
    </form>
    {{else}}
    <p class="empty">This account has no organization that this application can use.</p>
    {{end}}
  </main>
</body>
</html>`))

func RenderConsentPage(view ConsentView) ([]byte, error) {
	if strings.TrimSpace(view.ClientName) == "" {
		view.ClientName = "this application"
	}
	var buf bytes.Buffer
	if err := consentPage.Execute(&buf, view); err != nil {
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

// NewConsentView builds the consent step for an account. It shows the
// workspace step when selectedOrganizationID names a qualifying organization.
// It also shows that step when only one organization qualifies, because there
// is nothing to choose. Otherwise it shows the organization step.
func NewConsentView(ctx context.Context, account *models.Account, selectedOrganizationID string) (ConsentView, error) {
	organizations, err := ListConsentOrganizations(ctx, account)
	if err != nil {
		return ConsentView{}, err
	}
	view := ConsentView{Organizations: organizations}

	selected := selectOrganization(organizations, selectedOrganizationID)
	if selected == nil {
		return view, nil
	}
	workspaces, err := ListConsentWorkspaces(ctx, account, selected.OrganizationID)
	if err != nil {
		return ConsentView{}, err
	}
	view.Organization = selected
	view.Workspaces = workspaces
	return view, nil
}

func selectOrganization(organizations []OrganizationOption, organizationID string) *OrganizationOption {
	organizationID = strings.TrimSpace(organizationID)
	for i := range organizations {
		if organizations[i].OrganizationID == organizationID {
			return &organizations[i]
		}
	}
	if len(organizations) == 1 {
		return &organizations[0]
	}
	return nil
}

// ListConsentOrganizations lists the organizations where the account is an
// active human member and both MCP and workspaces are enabled.
func ListConsentOrganizations(ctx context.Context, account *models.Account) ([]OrganizationOption, error) {
	organizations, err := models.FindOrganizationsForAccount(account.Email)
	if err != nil {
		return nil, err
	}
	db := database.DB(ctx)
	options := make([]OrganizationOption, 0, len(organizations))
	for _, organization := range organizations {
		if _, err := findConsentUser(db, account, organization.ID); err != nil {
			continue
		}
		options = append(options, OrganizationOption{
			OrganizationID: organization.ID.String(),
			Name:           strings.TrimSpace(organization.Name),
		})
	}
	return options, nil
}

// ListConsentWorkspaces lists the workspaces of one organization that the
// account can grant. It returns none when the organization does not qualify.
func ListConsentWorkspaces(ctx context.Context, account *models.Account, organizationID string) ([]WorkspaceOption, error) {
	orgID, err := uuid.Parse(strings.TrimSpace(organizationID))
	if err != nil {
		return nil, fmt.Errorf("organization is required")
	}
	db := database.DB(ctx)
	if _, err := findConsentUser(db, account, orgID); err != nil {
		return []WorkspaceOption{}, nil
	}
	listed, err := models.ListFactories(db, orgID)
	if err != nil {
		return nil, err
	}
	options := make([]WorkspaceOption, 0, len(listed))
	for _, factory := range listed {
		options = append(options, WorkspaceOption{
			FactoryID: factory.ID.String(),
			Label:     strings.TrimSpace(factory.Name),
		})
	}
	return options, nil
}

// ResolveConsentGrant validates a consent choice. It returns the account's user
// in the organization, the organization, and the unique workspaces chosen.
func ResolveConsentGrant(
	ctx context.Context,
	account *models.Account,
	organizationID string,
	factoryIDs []string,
) (uuid.UUID, uuid.UUID, []uuid.UUID, error) {
	orgID, err := uuid.Parse(strings.TrimSpace(organizationID))
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, fmt.Errorf("organization is required")
	}
	if len(factoryIDs) == 0 {
		return uuid.Nil, uuid.Nil, nil, fmt.Errorf("workspace is required")
	}
	db := database.DB(ctx)
	user, err := findConsentUser(db, account, orgID)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, err
	}

	granted := make([]uuid.UUID, 0, len(factoryIDs))
	for _, raw := range factoryIDs {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return uuid.Nil, uuid.Nil, nil, fmt.Errorf("workspace is required")
		}
		if slices.Contains(granted, id) {
			continue
		}
		if _, err := models.FindFactory(db, orgID, id); err != nil {
			return uuid.Nil, uuid.Nil, nil, err
		}
		granted = append(granted, id)
	}
	return user.ID, orgID, granted, nil
}

func findConsentUser(db *gorm.DB, account *models.Account, orgID uuid.UUID) (*models.User, error) {
	if !OrganizationAllowsPublicMCP(orgID) || !organizationAllowsFactories(orgID) {
		return nil, fmt.Errorf("organization is not available")
	}
	return models.FindActiveHumanUserByAccountAndOrganization(db, orgID, account.ID)
}
