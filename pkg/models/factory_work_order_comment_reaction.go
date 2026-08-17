package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Canonical emoji reaction vocabulary. These are GitHub's own reaction
// content values (see pkg/integrations/github/components/pulls/add_reaction.go)
// reused as-is so the value stored here, the value sent to GitHub by
// automations, and any future cross-referencing all agree on one
// vocabulary. Glyph <-> value mapping is a frontend-only concern.
const (
	CommentReactionThumbsUp   = "+1"
	CommentReactionThumbsDown = "-1"
	CommentReactionLaugh      = "laugh"
	CommentReactionHooray     = "hooray"
	CommentReactionConfused   = "confused"
	CommentReactionHeart      = "heart"
	CommentReactionRocket     = "rocket"
	CommentReactionEyes       = "eyes"
)

// AllowedCommentReactionEmoji is the fixed set of emoji a comment can be
// reacted with. Order matches the emoji picker in the UI.
var AllowedCommentReactionEmoji = []string{
	CommentReactionThumbsUp,
	CommentReactionThumbsDown,
	CommentReactionLaugh,
	CommentReactionHooray,
	CommentReactionConfused,
	CommentReactionHeart,
	CommentReactionRocket,
	CommentReactionEyes,
}

var ErrFactoryWorkOrderCommentNotFound = errors.New("factory work order comment not found")

// IsValidCommentReactionEmoji reports whether emoji is one of the fixed,
// GitHub-style reaction values CreateCommentReaction accepts.
func IsValidCommentReactionEmoji(emoji string) bool {
	for _, allowed := range AllowedCommentReactionEmoji {
		if allowed == emoji {
			return true
		}
	}
	return false
}

// FactoryWorkOrderCommentReaction is a single user's reaction to a work
// order comment. Comments aren't a first-class table — they are
// `factory_work_order_events` rows of type `order.comment.added` — so
// CommentID references that table's primary key directly.
type FactoryWorkOrderCommentReaction struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	CommentID      uuid.UUID
	Emoji          string
	UserID         uuid.UUID
	CreatedAt      time.Time
}

func (FactoryWorkOrderCommentReaction) TableName() string {
	return "factory_work_order_comment_reactions"
}

// CommentReactionSummary is the aggregated view of one emoji's reactions
// on a single comment, scoped to the user making the request.
type CommentReactionSummary struct {
	Emoji       string
	Count       int
	ReactedByMe bool
}

// findCommentEvent resolves commentID to the `order.comment.added` event
// it must reference, scoped to this work order. Returns
// ErrFactoryWorkOrderCommentNotFound if the event doesn't exist, belongs
// to a different work order, or isn't a comment (e.g. someone tries to
// react to a status-update event).
func (o *FactoryWorkOrder) findCommentEvent(tx *gorm.DB, commentID uuid.UUID) (*FactoryWorkOrderEvent, error) {
	var event FactoryWorkOrderEvent
	err := tx.
		Where("id = ? AND work_order_id = ? AND type = ?", commentID, o.ID, factory.EventTypeOrderCommentAdded).
		First(&event).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryWorkOrderCommentNotFound
		}
		return nil, err
	}

	return &event, nil
}

// AddCommentReaction records userID's emoji reaction to commentID.
// Idempotent: reacting twice with the same emoji is a no-op, matching the
// toggle semantics the API exposes (the caller decides whether to add or
// remove based on the current summary).
func (o *FactoryWorkOrder) AddCommentReaction(tx *gorm.DB, commentID uuid.UUID, userID uuid.UUID, emoji string) error {
	if _, err := o.findCommentEvent(tx, commentID); err != nil {
		return err
	}

	reaction := &FactoryWorkOrderCommentReaction{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		CommentID:      commentID,
		Emoji:          emoji,
		UserID:         userID,
		CreatedAt:      time.Now(),
	}

	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(reaction).Error
}

// RemoveCommentReaction deletes userID's emoji reaction from commentID.
// Idempotent: removing a reaction that doesn't exist is not an error.
func (o *FactoryWorkOrder) RemoveCommentReaction(tx *gorm.DB, commentID uuid.UUID, userID uuid.UUID, emoji string) error {
	if _, err := o.findCommentEvent(tx, commentID); err != nil {
		return err
	}

	return tx.
		Where("work_order_id = ? AND comment_id = ? AND user_id = ? AND emoji = ?", o.ID, commentID, userID, emoji).
		Delete(&FactoryWorkOrderCommentReaction{}).
		Error
}

// ListCommentReactionSummaries aggregates reactions for every comment id in
// commentIDs in a single grouped query, so callers rendering a page of
// timeline events don't issue one query per comment. currentUserID marks
// which summaries the caller has personally reacted with.
func ListCommentReactionSummaries(
	tx *gorm.DB,
	orderID uuid.UUID,
	commentIDs []uuid.UUID,
	currentUserID uuid.UUID,
) (map[uuid.UUID][]CommentReactionSummary, error) {
	result := make(map[uuid.UUID][]CommentReactionSummary)
	if len(commentIDs) == 0 {
		return result, nil
	}

	type row struct {
		CommentID   uuid.UUID
		Emoji       string
		Count       int
		ReactedByMe bool
	}

	var rows []row
	err := tx.
		Model(&FactoryWorkOrderCommentReaction{}).
		Select("comment_id, emoji, count(*) as count, bool_or(user_id = ?) as reacted_by_me", currentUserID).
		Where("work_order_id = ? AND comment_id IN ?", orderID, commentIDs).
		Group("comment_id, emoji").
		Order("comment_id, emoji").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		result[r.CommentID] = append(result[r.CommentID], CommentReactionSummary{
			Emoji:       r.Emoji,
			Count:       r.Count,
			ReactedByMe: r.ReactedByMe,
		})
	}

	return result, nil
}
