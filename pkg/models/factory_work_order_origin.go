package models

import (
	"errors"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const factoryWorkOrderOriginUniqueConstraint = "idx_factory_work_orders_factory_origin_url"

var ErrFactoryWorkOrderOriginTaken = errors.New("a work order already exists for this origin")

// WorkOrderOrigin is the external ticket a work order was created from.
type WorkOrderOrigin struct {
	URL   string
	Label string
}

func OriginFromIntakePayload(source string, payload map[string]any) *WorkOrderOrigin {
	originURL := intakeOriginURL(source, payload)
	if originURL == "" {
		return nil
	}

	return &WorkOrderOrigin{
		URL:   originURL,
		Label: OriginLabelFromURL(originURL),
	}
}

func OriginFromIntakeRootEvent(source string, event *CanvasEvent) *WorkOrderOrigin {
	if event == nil {
		return nil
	}

	payload, ok := RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return nil
	}

	return OriginFromIntakePayload(source, payload)
}

func OriginLabelFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return strings.TrimSpace(rawURL)
	}

	if label := githubOriginLabel(parsed); label != "" {
		return label
	}

	if label := hostedOriginLabel(parsed); label != "" {
		return label
	}

	return strings.TrimSpace(rawURL)
}

func applyWorkOrderOrigin(order *FactoryWorkOrder, origin *WorkOrderOrigin) {
	if order == nil || origin == nil {
		return
	}

	url := strings.TrimSpace(origin.URL)
	if url == "" {
		return
	}

	order.OriginURL = &url
	label := strings.TrimSpace(origin.Label)
	if label == "" {
		label = OriginLabelFromURL(url)
	}
	if label != "" {
		order.OriginLabel = &label
	}
}

func MapFactoryWorkOrderOriginUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		(pgErr.ConstraintName == factoryWorkOrderOriginUniqueConstraint ||
			strings.Contains(pgErr.Message, factoryWorkOrderOriginUniqueConstraint) ||
			strings.Contains(pgErr.Detail, factoryWorkOrderOriginUniqueConstraint)) {
		return ErrFactoryWorkOrderOriginTaken
	}

	if strings.Contains(err.Error(), factoryWorkOrderOriginUniqueConstraint) {
		return ErrFactoryWorkOrderOriginTaken
	}

	return err
}

func (f *Factory) FindWorkOrderByOriginURL(tx *gorm.DB, originURL string) (*FactoryWorkOrder, error) {
	trimmed := strings.TrimSpace(originURL)
	if trimmed == "" {
		return nil, ErrFactoryWorkOrderNotFound
	}

	var order FactoryWorkOrder
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND origin_url = ? AND state IN ?", f.OrganizationID, f.ID, trimmed, []string{
			FactoryWorkOrderStateDraft,
			FactoryWorkOrderStateOpen,
		}).
		First(&order).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderNotFound
		}
		return nil, err
	}

	return f.FindWorkOrder(tx, order.ID)
}

func intakeOriginURL(source string, payload map[string]any) string {
	switch source {
	case FactoryIntakeSourceGitHubIssues:
		return nestedMapString(payload, "issue", "html_url")
	case FactoryIntakeSourceSentryExceptions:
		return nestedMapString(payload, "data", "issue", "permalink")
	case FactoryIntakeSourcePagerDutyIncidents:
		return nestedMapString(payload, "incident", "html_url")
	default:
		return ""
	}
}

func nestedMapString(payload map[string]any, path ...string) string {
	current := any(payload)
	for _, key := range path {
		record, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = record[key]
	}

	value, ok := current.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

func githubOriginLabel(parsed *url.URL) string {
	if parsed.Hostname() != "github.com" {
		return ""
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 {
		return ""
	}

	owner, repo, kind, number := parts[0], parts[1], parts[2], parts[3]
	if owner == "" || repo == "" || number == "" {
		return ""
	}
	if kind != "issues" && kind != "pull" {
		return ""
	}

	return owner + "/" + repo + "#" + number
}

func hostedOriginLabel(parsed *url.URL) string {
	host := parsed.Hostname()
	parts := strings.FieldsFunc(parsed.Path, func(r rune) bool { return r == '/' })
	var id string
	switch {
	case strings.HasSuffix(host, "sentry.io"):
		id = nextPathSegment(parts, "issues")
	case strings.HasSuffix(host, "pagerduty.com"):
		id = nextPathSegment(parts, "incidents")
	default:
		return ""
	}
	if id == "" {
		return ""
	}

	org := strings.Split(host, ".")[0]
	if org == "" {
		return id
	}
	return org + "#" + id
}

func nextPathSegment(parts []string, marker string) string {
	for i, part := range parts {
		if part == marker && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
