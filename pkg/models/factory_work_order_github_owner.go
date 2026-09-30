package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const githubIssueOwnerSavepoint = "github_issue_owner"

func (o *FactoryWorkOrder) assignOwnerFromGitHubIssue(tx *gorm.DB) error {
	ownerID, ok := o.githubIssueOwner(tx)
	if !ok {
		return nil
	}
	if err := o.UpdateAssignees(tx, []uuid.UUID{ownerID}, ownerID); err != nil {
		return err
	}
	return tx.Preload("User").Where("work_order_id = ?", o.ID).Find(&o.Assignees).Error
}

func (o *FactoryWorkOrder) githubIssueOwner(tx *gorm.DB) (uuid.UUID, bool) {
	if o == nil || o.SourceRunID == nil {
		return uuid.Nil, false
	}
	if err := tx.SavePoint(githubIssueOwnerSavepoint).Error; err != nil {
		return uuid.Nil, false
	}

	ownerID, ok, err := o.findGitHubIssueOwner(tx)
	if err != nil || !ok {
		_ = tx.RollbackTo(githubIssueOwnerSavepoint)
		return uuid.Nil, false
	}
	return ownerID, true
}

func (o *FactoryWorkOrder) findGitHubIssueOwner(tx *gorm.DB) (uuid.UUID, bool, error) {
	assignees, err := o.ListAssignees(tx)
	if err != nil {
		return uuid.Nil, false, err
	}
	if len(assignees) > 0 {
		return uuid.Nil, false, nil
	}

	event, err := FindRootEventForRun(tx, *o.SourceRunID)
	if err != nil {
		return uuid.Nil, false, err
	}
	if event == nil {
		return uuid.Nil, false, nil
	}

	logins := gitHubIssueAssigneeLogins(RootEventSourcePayload(event.Data.Data()))
	if len(logins) == 0 {
		return uuid.Nil, false, nil
	}

	members, err := ListFactoryVelocityMembers(tx, o.OrganizationID)
	if err != nil {
		return uuid.Nil, false, err
	}
	ownerID, ok := firstMatchedGitHubMember(members, logins)
	return ownerID, ok, nil
}

func gitHubIssueAssigneeLogins(payload any) []string {
	source, ok := objectMap(payload)
	if !ok || !isGitHubIssuePayload(source) {
		return nil
	}
	issue, ok := objectMap(source["issue"])
	if !ok {
		return nil
	}
	rawAssignees, ok := issue["assignees"].([]any)
	if !ok {
		return nil
	}

	logins := make([]string, 0, len(rawAssignees))
	for _, rawAssignee := range rawAssignees {
		assignee, ok := objectMap(rawAssignee)
		if !ok {
			continue
		}
		login, _ := assignee["login"].(string)
		if login == "" {
			continue
		}
		logins = append(logins, login)
	}
	return logins
}

func isGitHubIssuePayload(payload map[string]any) bool {
	repository, ok := objectMap(payload["repository"])
	if !ok {
		return false
	}
	fullName, _ := repository["full_name"].(string)
	if strings.TrimSpace(fullName) == "" {
		return false
	}
	_, ok = objectMap(payload["issue"])
	return ok
}

func firstMatchedGitHubMember(members []FactoryVelocityMember, logins []string) (uuid.UUID, bool) {
	byLogin := make(map[string]uuid.UUID, len(members))
	for _, member := range members {
		login := normalizeGitHubLogin(member.GitHubLogin)
		if login == "" {
			continue
		}
		byLogin[login] = member.UserID
	}
	for _, login := range logins {
		userID, ok := byLogin[normalizeGitHubLogin(login)]
		if ok {
			return userID, true
		}
	}
	return uuid.Nil, false
}

func normalizeGitHubLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}
