package cloudsmith

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListPackages struct{}

type ListPackagesSpec struct {
	Repository          string `json:"repository" mapstructure:"repository"`
	SyncStatus          string `json:"syncStatus" mapstructure:"syncStatus"`
	QuarantineStatus    string `json:"quarantineStatus" mapstructure:"quarantineStatus"`
	VulnerabilityStatus string `json:"vulnerabilityStatus" mapstructure:"vulnerabilityStatus"`
}

// TrimmedPackage holds the subset of package fields emitted in the list result.
type TrimmedPackage struct {
	Description        string         `json:"description"`
	DisplayName        string         `json:"display_name"`
	Format             string         `json:"format"`
	IsQuarantined      bool           `json:"is_quarantined"`
	License            string         `json:"license"`
	PolicyViolated     bool           `json:"policy_violated"`
	Repository         string         `json:"repository"`
	SecurityScanStatus string         `json:"security_scan_status"`
	SlugPerm           string         `json:"slug_perm"`
	StageStr           string         `json:"stage_str"`
	StatusStr          string         `json:"status_str"`
	Tags               map[string]any `json:"tags"`
}

// ListPackagesResult is the single payload emitted by ListPackages.Execute.
type ListPackagesResult struct {
	Packages []TrimmedPackage `json:"packages"`
}

func (l *ListPackages) Name() string {
	return "cloudsmith.listPackages"
}

func (l *ListPackages) Label() string {
	return "List Packages"
}

func (l *ListPackages) Description() string {
	return "List packages in a Cloudsmith repository with optional filtering by sync status, quarantine, and vulnerability"
}

func (l *ListPackages) Documentation() string {
	return `The List Packages component fetches all packages in a Cloudsmith repository and optionally filters them by sync status, quarantine state, or vulnerability scan result.

## Use Cases

- **Release auditing**: List all fully synchronized packages before a release gate
- **Quarantine monitoring**: Enumerate quarantined packages for a security review workflow
- **Vulnerability triage**: Retrieve packages with detected vulnerabilities and route them to a remediation step
- **Inventory**: Collect a complete snapshot of packages in a repository for reporting

## Configuration

- **Repository** (required): The repository to list packages from, in the form ` + "`owner/repository`" + `.
- **Sync Status** (optional): Filter by package synchronization state (` + "`Any`" + `, ` + "`Fully Synchronised`" + `, ` + "`Awaiting Sync`" + `, ` + "`Sync Failed`" + `).
- **Quarantine Status** (optional): Filter by quarantine state (` + "`Any`" + `, ` + "`Quarantined`" + `, ` + "`Not Quarantined`" + `).
- **Vulnerability Status** (optional): Filter by security scan result (` + "`Any`" + `, ` + "`No Vulnerabilities`" + `, ` + "`Vulnerabilities Found`" + `).

## Output

Emits a single payload containing a ` + "`packages`" + ` array. Each entry includes:
- **display_name** / **format**: Package display name and format
- **status_str** / **stage_str**: Human-readable status and sync stage
- **is_quarantined** / **policy_violated**: Quarantine and policy state
- **description** / **license**: Package description and license
- **slug_perm** / **repository**: Permanent identifier and repository slug
- **tags**: Package tags`
}

func (l *ListPackages) Icon() string {
	return "list"
}

func (l *ListPackages) Color() string {
	return "gray"
}

func (l *ListPackages) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListPackages) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The repository to list packages from",
			Placeholder: "Select repository",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "syncStatus",
			Label:       "Sync Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Filter packages by their synchronization state",
			Default:     "any",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Any", Value: "any"},
						{Label: "Fully Synchronised", Value: "fully_synchronised"},
						{Label: "Awaiting Sync", Value: "awaiting"},
						{Label: "Sync Failed", Value: "failed"},
					},
				},
			},
		},
		{
			Name:        "quarantineStatus",
			Label:       "Quarantine Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Filter packages by their quarantine state",
			Default:     "any",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Any", Value: "any"},
						{Label: "Quarantined", Value: "quarantined"},
						{Label: "Not Quarantined", Value: "not_quarantined"},
					},
				},
			},
		},
		{
			Name:        "vulnerabilityStatus",
			Label:       "Vulnerability Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Filter packages by their security scan result",
			Default:     "any",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Any", Value: "any"},
						{Label: "No Vulnerabilities", Value: "no_vulnerabilities"},
						{Label: "Vulnerabilities Found", Value: "vulnerabilities_found"},
					},
				},
			},
		},
	}
}

func (l *ListPackages) Setup(ctx core.SetupContext) error {
	spec := ListPackagesSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Repository == "" {
		return errors.New("repository is required")
	}

	return resolveRepositoryMetadata(ctx, spec.Repository)
}

func (l *ListPackages) Execute(ctx core.ExecutionContext) error {
	spec := ListPackagesSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	owner, repo, err := parseRepositoryID(spec.Repository)
	if err != nil {
		return fmt.Errorf("invalid repository %q: %w", spec.Repository, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	query := buildPackageQuery(spec)
	packages, err := client.ListPackagesWithFilters(owner, repo, query)
	if err != nil {
		return fmt.Errorf("failed to list packages: %v", err)
	}

	trimmed := make([]TrimmedPackage, len(packages))
	for i, pkg := range packages {
		trimmed[i] = TrimmedPackage{
			Description:        pkg.Description,
			DisplayName:        pkg.DisplayName,
			Format:             pkg.Format,
			IsQuarantined:      pkg.IsQuarantined,
			License:            pkg.License,
			PolicyViolated:     pkg.PolicyViolated,
			Repository:         pkg.Repository,
			SecurityScanStatus: pkg.SecurityScanStatus,
			SlugPerm:           pkg.SlugPerm,
			StageStr:           pkg.StageStr,
			StatusStr:          pkg.StatusStr,
			Tags:               pkg.Tags,
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudsmith.packages.listed",
		[]any{ListPackagesResult{Packages: trimmed}},
	)
}

// buildPackageQuery constructs a Cloudsmith Lucene-style query string from the spec filters.
func buildPackageQuery(spec ListPackagesSpec) string {
	var parts []string

	switch spec.SyncStatus {
	case "fully_synchronised":
		parts = append(parts, "is_sync_completed:true")
	case "awaiting":
		parts = append(parts, "is_sync_awaiting:true")
	case "failed":
		parts = append(parts, "is_sync_failed:true")
	}

	switch spec.QuarantineStatus {
	case "quarantined":
		parts = append(parts, "is_quarantined:true")
	case "not_quarantined":
		parts = append(parts, "is_quarantined:false")
	}

	switch spec.VulnerabilityStatus {
	case "no_vulnerabilities":
		parts = append(parts, "security_scan_status:\"No Vulnerabilities Found\"")
	case "vulnerabilities_found":
		parts = append(parts, "security_scan_status:\"Scan Detected Vulnerabilities\"")
	}

	return strings.Join(parts, " AND ")
}

func (l *ListPackages) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListPackages) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListPackages) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (l *ListPackages) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (l *ListPackages) Hooks() []core.Hook {
	return []core.Hook{}
}

func (l *ListPackages) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
