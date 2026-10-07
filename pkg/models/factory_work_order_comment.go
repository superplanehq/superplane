package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FactoryWorkOrderComment is a first-class work order comment. Timeline
// events still record `order.comment.added`; this row is the source of
// truth for the thread, mentions, and `order().comments`.
type FactoryWorkOrderComment struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	AuthorUserID   *uuid.UUID
	AuthorKind     string
	Automation     datatypes.JSON
	SourceRunID    *uuid.UUID
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// MentionedUserIDs is filled after RecordCommentAdded persists
	// mention rows. It is not a database column.
	MentionedUserIDs []uuid.UUID `gorm:"-"`
}

func (FactoryWorkOrderComment) TableName() string {
	return "factory_work_order_comments"
}

// FactoryWorkOrderCommentParams carries the fields RecordCommentAdded
// needs to persist a comment, its mentions, and the timeline event.
type FactoryWorkOrderCommentParams struct {
	Body             string
	Author           factory.WorkOrderCommentAuthor
	Run              *factory.RunRef
	MentionedUserIDs []uuid.UUID
}

type FactoryWorkOrderCommentMention struct {
	CommentID uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
}

func (FactoryWorkOrderCommentMention) TableName() string {
	return "factory_work_order_comment_mentions"
}

// Author reconstructs the timeline author payload from the comment row.
func (c *FactoryWorkOrderComment) Author() factory.WorkOrderCommentAuthor {
	author := factory.WorkOrderCommentAuthor{Kind: c.AuthorKind}
	if c.AuthorUserID != nil {
		userID := c.AuthorUserID.String()
		author.UserID = &userID
	}
	if automation := decodeCommentAutomation(c.Automation); automation != nil {
		author.Automation = automation
	}
	return author
}

func (c *FactoryWorkOrderComment) RunRef() *factory.RunRef {
	if c.SourceRunID == nil {
		return nil
	}
	return &factory.RunRef{ID: *c.SourceRunID}
}

func (o *FactoryWorkOrder) RecordCommentAdded(
	tx *gorm.DB,
	params FactoryWorkOrderCommentParams,
) (*FactoryWorkOrderComment, error) {
	comment, err := o.insertComment(tx, params)
	if err != nil {
		return nil, err
	}

	mentioned, err := o.insertCommentMentions(tx, comment.ID, params.MentionedUserIDs)
	if err != nil {
		return nil, err
	}
	comment.MentionedUserIDs = mentioned

	if err := o.recordCommentAddedEvent(tx, comment, params.Author); err != nil {
		return nil, err
	}

	return comment, nil
}

// ListComments returns the work order's comment thread, oldest first —
// unlike ListEvents (which is DESC for activity feeds), a comment thread
// reads chronologically oldest→newest.
func (o *FactoryWorkOrder) ListComments(tx *gorm.DB) ([]FactoryWorkOrderComment, error) {
	comments := []FactoryWorkOrderComment{}
	err := tx.
		Where("work_order_id = ?", o.ID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&comments).
		Error
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (o *FactoryWorkOrder) insertComment(
	tx *gorm.DB,
	params FactoryWorkOrderCommentParams,
) (*FactoryWorkOrderComment, error) {
	now := time.Now()
	comment := &FactoryWorkOrderComment{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		AuthorKind:     params.Author.Kind,
		Automation:     encodeCommentAutomation(params.Author.Automation),
		Body:           params.Body,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if params.Author.UserID != nil {
		if userID, err := uuid.Parse(*params.Author.UserID); err == nil {
			comment.AuthorUserID = &userID
		}
	}
	if params.Run != nil {
		runID := params.Run.ID
		comment.SourceRunID = &runID
	}

	if err := tx.Create(comment).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

func (o *FactoryWorkOrder) insertCommentMentions(
	tx *gorm.DB,
	commentID uuid.UUID,
	mentionedUserIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	memberIDs, err := filterOrgMemberIDs(tx, o.OrganizationID, mentionedUserIDs)
	if err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	now := time.Now()
	mentions := make([]FactoryWorkOrderCommentMention, 0, len(memberIDs))
	for _, userID := range memberIDs {
		mentions = append(mentions, FactoryWorkOrderCommentMention{
			CommentID: commentID,
			UserID:    userID,
			CreatedAt: now,
		})
	}

	if err := tx.Create(&mentions).Error; err != nil {
		return nil, err
	}

	return memberIDs, nil
}

func (o *FactoryWorkOrder) recordCommentAddedEvent(
	tx *gorm.DB,
	comment *FactoryWorkOrderComment,
	author factory.WorkOrderCommentAuthor,
) error {
	data := factory.WorkOrderCommentAdded{
		Order:          o.Ref(),
		CommentID:      comment.ID,
		Body:           comment.Body,
		Author:         &author,
		Run:            comment.RunRef(),
		MentionedUsers: userRefs(comment.MentionedUserIDs),
	}

	return o.recordEvent(tx, factory.EventTypeOrderCommentAdded, data)
}

func encodeCommentAutomation(automation *factory.AutomationRef) datatypes.JSON {
	if automation == nil {
		return nil
	}

	encoded, err := json.Marshal(automation)
	if err != nil {
		return nil
	}

	return datatypes.JSON(encoded)
}

func decodeCommentAutomation(raw datatypes.JSON) *factory.AutomationRef {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var automation factory.AutomationRef
	if err := json.Unmarshal(raw, &automation); err != nil {
		return nil
	}

	return &automation
}

func filterOrgMemberIDs(tx *gorm.DB, organizationID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	uniqueIDs := uniqueUUIDs(ids)
	if len(uniqueIDs) == 0 {
		return []uuid.UUID{}, nil
	}

	var found []uuid.UUID
	err := tx.Model(&User{}).
		Where("organization_id = ?", organizationID).
		Where("id IN ?", uniqueIDs).
		Pluck("id", &found).
		Error
	if err != nil {
		return nil, err
	}

	allowed := make(map[uuid.UUID]struct{}, len(found))
	for _, id := range found {
		allowed[id] = struct{}{}
	}

	members := make([]uuid.UUID, 0, len(found))
	for _, id := range uniqueIDs {
		if _, ok := allowed[id]; ok {
			members = append(members, id)
		}
	}

	return members, nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	unique := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func uuidStrings(ids []uuid.UUID) []string {
	if len(ids) == 0 {
		return nil
	}

	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, id.String())
	}
	return result
}

func userRefs(ids []uuid.UUID) []factory.UserRef {
	refs := make([]factory.UserRef, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, factory.UserRef{ID: id})
	}
	return refs
}
