package models

import (
	"time"

	"github.com/google/uuid"
)

// Reaction content values. These mirror GitHub's fixed reaction vocabulary
// (see pkg/integrations/github/components/pulls/add_reaction.go) so the
// same emoji set / mapping is used consistently across the product.
const (
	ReactionThumbsUp   = "+1"
	ReactionThumbsDown = "-1"
	ReactionLaugh      = "laugh"
	ReactionConfused   = "confused"
	ReactionHeart      = "heart"
	ReactionHooray     = "hooray"
	ReactionRocket     = "rocket"
	ReactionEyes       = "eyes"
)

// ValidWorkOrderReactionContents is the fixed set of reactions a user can
// add to a work order, in display order.
var ValidWorkOrderReactionContents = []string{
	ReactionThumbsUp,
	ReactionThumbsDown,
	ReactionLaugh,
	ReactionHooray,
	ReactionConfused,
	ReactionHeart,
	ReactionRocket,
	ReactionEyes,
}

// IsValidWorkOrderReactionContent reports whether content is one of the
// fixed GitHub-style reaction values work orders support.
func IsValidWorkOrderReactionContent(content string) bool {
	for _, valid := range ValidWorkOrderReactionContents {
		if content == valid {
			return true
		}
	}

	return false
}

// FactoryWorkOrderReaction is a single (user, emoji) reaction on a work
// order. A user may hold multiple distinct reactions on the same order
// (e.g. both `+1` and `heart`), but only one row per (user, content) pair.
type FactoryWorkOrderReaction struct {
	WorkOrderID uuid.UUID `gorm:"primaryKey"`
	UserID      uuid.UUID `gorm:"primaryKey"`
	Content     string    `gorm:"primaryKey"`
	CreatedAt   time.Time

	User *User `gorm:"foreignKey:UserID"`
}

func (FactoryWorkOrderReaction) TableName() string {
	return "factory_work_order_reactions"
}
