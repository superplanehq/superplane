package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryAppTemplateHumanReviewID = "human-review"
	HumanReviewTriggerNodeID        = "on-human-review"
	HumanReviewAgentNodeID          = "human-review-agent"
)

var (
	humanReviewTeamPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9][A-Za-z0-9_-]*$`)
	humanReviewReviewerPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

type HumanReviewSettings struct {
	Enabled         bool
	DefaultReviewer string
	FallbackTeam    string
	AgentNodeID     string
}

// HumanReviewSettingsFromNodes only recognizes the GitHub built-in template.
// A user-created onRun trigger cannot enable enforcement for a factory.
func HumanReviewSettingsFromNodes(nodes []Node) (HumanReviewSettings, bool) {
	if FactoryAppTemplateID(nodes) != FactoryAppTemplateHumanReviewID {
		return HumanReviewSettings{}, false
	}
	for _, node := range nodes {
		if node.ID != HumanReviewTriggerNodeID || node.Type != NodeTypeTrigger {
			continue
		}
		enabled, _ := node.Configuration["enabled"].(bool)
		reviewer, _ := node.Configuration["defaultReviewer"].(string)
		team, _ := node.Configuration["fallbackTeam"].(string)
		return HumanReviewSettings{Enabled: enabled, DefaultReviewer: strings.TrimSpace(reviewer), FallbackTeam: strings.TrimSpace(team), AgentNodeID: HumanReviewAgentNodeID}, true
	}
	return HumanReviewSettings{}, false
}

func (s HumanReviewSettings) Validate() error {
	if !s.Enabled {
		return nil
	}
	if s.DefaultReviewer != "" {
		if !humanReviewReviewerPattern.MatchString(s.DefaultReviewer) {
			return fmt.Errorf("enter a GitHub username without @, spaces, or a URL")
		}
		return nil
	}
	if !humanReviewTeamPattern.MatchString(s.FallbackTeam) {
		return fmt.Errorf("select a default GitHub reviewer or a valid fallback team before enabling Human review")
	}
	return nil
}

// LockHumanReviewPolicy serializes policy saves without blocking foreign-key
// checks from decision inserts that hold a publication lock.
func (f *Factory) LockHumanReviewPolicy(tx *gorm.DB) error {
	return tx.Clauses(clause.Locking{Strength: lockingForUpdateNoKey}).
		Where("id = ? AND organization_id = ?", f.ID, f.OrganizationID).
		First(f).Error
}

// LockHumanReviewPublications waits for in-flight provider mutations before a
// new policy becomes visible. The worker takes the same lock for one PR.
func (f *Factory) LockHumanReviewPublications(tx *gorm.DB) error {
	var ids []uuid.UUID
	if err := tx.Model(&FactoryPullRequest{}).
		Where("factory_id = ? AND provider = ? AND state IN ?", f.ID, FactoryPullRequestProviderGitHub, []string{FactoryPullRequestStateOpen, FactoryPullRequestStateDraft}).
		Order("id ASC").Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "human-review:"+id.String()).Error; err != nil {
			return err
		}
	}
	return nil
}

// HasOtherHumanReviewAutomation runs while the caller holds the Factory lock.
// One automation owns the policy, independent of editable canvas names.
func (f *Factory) HasOtherHumanReviewAutomation(tx *gorm.DB, canvasID uuid.UUID) (bool, error) {
	canvases, err := f.ListCanvases(tx)
	if err != nil {
		return false, err
	}
	for _, canvas := range canvases {
		if canvas.ID == canvasID || canvas.LiveVersionID == nil {
			continue
		}
		version, err := FindLiveCanvasVersionByCanvasInTransaction(tx, &canvas)
		if err != nil {
			return false, err
		}
		if FactoryAppTemplateID(version.Nodes) == FactoryAppTemplateHumanReviewID {
			return true, nil
		}
	}
	return false, nil
}
