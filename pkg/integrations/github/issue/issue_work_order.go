package issue

import (
	"crypto/sha256"
	"encoding/binary"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// PayloadType is the event type that the On Issue trigger emits.
const PayloadType = "github.issue"

// Ref identifies one GitHub issue by its page URL.
type Ref struct {
	Host       string
	Repository string
	Number     int
}

func (r Ref) valid() bool {
	return r.Host != "" && r.Repository != "" && r.Number > 0
}

// RefFromEventData reads the issue from a canvas root event envelope:
//
//	{ "type": "github.issue", "data": { "issue": { "html_url": "https://github.com/acme/app/issues/12" } } }
func RefFromEventData(eventData any) (Ref, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return Ref{}, false
	}
	if typeName, _ := envelope["type"].(string); typeName != PayloadType {
		return Ref{}, false
	}

	body, _ := models.RootEventSourcePayload(eventData).(map[string]any)
	issue, _ := body["issue"].(map[string]any)
	page, _ := issue["html_url"].(string)
	return RefFromURL(page)
}

// RefFromURL reads the issue from a page URL with the path
// /{owner}/{repo}/issues/{number}.
func RefFromURL(rawURL string) (Ref, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return Ref{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] != "issues" {
		return Ref{}, false
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil {
		return Ref{}, false
	}

	ref := Ref{
		Host:       strings.ToLower(parsed.Hostname()),
		Repository: strings.ToLower(parts[0] + "/" + parts[1]),
		Number:     number,
	}
	return ref, ref.valid()
}

// LockWorkOrder serializes work-order creation for one issue in this
// factory. GitHub sends one webhook for each added label, so two runs for
// the same issue can arrive together. The lock is held until tx commits.
func LockWorkOrder(tx *gorm.DB, factory *models.Factory, ref Ref) error {
	if tx == nil || factory == nil || !ref.valid() {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", lockKey(factory.ID, ref)).Error
}

func lockKey(factoryID uuid.UUID, ref Ref) int64 {
	sum := sha256.New()
	sum.Write([]byte("github-issue-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(ref.Host))
	sum.Write([]byte{0})
	sum.Write([]byte(ref.Repository))
	sum.Write([]byte{0})
	sum.Write([]byte(strconv.Itoa(ref.Number)))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

// HasActiveWorkOrder reports whether this factory has a task for the issue
// that is not closed. A closed task does not count, so a reopened issue can
// create a new task.
func HasActiveWorkOrder(tx *gorm.DB, factory *models.Factory, ref Ref) (bool, error) {
	if factory == nil || !ref.valid() {
		return false, nil
	}

	orders, err := factory.ListWorkOrdersByOriginURLFragment(tx, ref.Repository+"/issues/"+strconv.Itoa(ref.Number))
	if err != nil {
		return false, err
	}

	for i := range orders {
		if orders[i].IsClosed() || orders[i].OriginURL == nil {
			continue
		}
		if found, ok := RefFromURL(*orders[i].OriginURL); ok && found == ref {
			return true, nil
		}
	}
	return false, nil
}
