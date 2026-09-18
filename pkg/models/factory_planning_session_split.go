package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanningSplitTask is one task the agent splits off the draft it refines.
type PlanningSplitTask struct {
	Title       string
	Description string
}

// PlanningTaskMessage is the JSON body of a task-role chat message. It
// snapshots the key and title at creation so the transcript and the rewind
// prompt can name the task even when the order is later renamed.
type PlanningTaskMessage struct {
	WorkOrderID string `json:"work_order_id"`
	Key         string `json:"key"`
	Title       string `json:"title"`
}

// ParsePlanningTaskMessage decodes a task-role message body.
func ParsePlanningTaskMessage(text string) (PlanningTaskMessage, bool) {
	var body PlanningTaskMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &body); err != nil {
		return PlanningTaskMessage{}, false
	}
	if body.WorkOrderID == "" {
		return PlanningTaskMessage{}, false
	}
	return body, true
}

// CreateSplitTask creates a new draft in the factory for part of the work
// this session refines, links it to the session, and records a task-role
// chat message so the transcript shows the new task in order. The parent's
// owners carry over, plus the person whose last chat message confirmed the
// split. That person is the creator, so the new draft reads as theirs.
func (s *FactoryPlanningSession) CreateSplitTask(
	tx *gorm.DB,
	factoryModel *Factory,
	task PlanningSplitTask,
	activityID uuid.UUID,
) (*FactoryWorkOrder, error) {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: task title is required", ErrFactoryPlanningSessionInvalid)
	}
	description := strings.TrimSpace(task.Description)
	if description == "" {
		return nil, fmt.Errorf("%w: task description is required", ErrFactoryPlanningSessionInvalid)
	}
	var created *FactoryWorkOrder
	err := s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		parent, err := s.analysisWorkOrder(inner)
		if err != nil {
			return err
		}
		existing, err := s.SplitTaskOrders(inner)
		if err != nil {
			return err
		}
		if len(existing) >= maxPlanningSplitTasks {
			return fmt.Errorf("%w: this session already created %d tasks", ErrFactoryPlanningSessionInvalid, maxPlanningSplitTasks)
		}
		creator, owners, err := s.splitTaskPeople(inner, parent)
		if err != nil {
			return err
		}
		order, err := factoryModel.CreateWorkOrder(inner, title, description, creator, owners, nil)
		if err != nil {
			return err
		}
		if err := s.attachCreatedWorkOrder(inner, order.ID); err != nil {
			return err
		}
		if err := s.recordTaskMessage(inner, factoryModel, order, activityID); err != nil {
			return err
		}
		created = order
		return nil
	})
	return created, err
}

// SplitTaskOrders lists the tasks this session split off its draft. The
// draft itself is also linked to the session, so it is left out.
func (s *FactoryPlanningSession) SplitTaskOrders(tx *gorm.DB) ([]FactoryWorkOrder, error) {
	orders, err := s.CreatedOrders(tx)
	if err != nil {
		return nil, err
	}
	if s.DraftWorkOrderID == nil {
		return orders, nil
	}
	return slices.DeleteFunc(orders, func(order FactoryWorkOrder) bool {
		return order.ID == *s.DraftWorkOrderID
	}), nil
}

// splitTaskPeople picks the creator and owners for a split task: the last
// person who wrote in the chat (they confirmed the split), falling back to
// the parent's creator, plus every current owner of the parent.
func (s *FactoryPlanningSession) splitTaskPeople(tx *gorm.DB, parent *FactoryWorkOrder) (*uuid.UUID, []uuid.UUID, error) {
	assignees, err := parent.ListAssignees(tx)
	if err != nil {
		return nil, nil, err
	}
	owners := make([]uuid.UUID, 0, len(assignees)+1)
	for _, assignee := range assignees {
		owners = append(owners, assignee.UserID)
	}
	creator := s.lastChatUser()
	if creator == nil {
		creator = parent.CreatedByID
	}
	if creator != nil && !slices.Contains(owners, *creator) {
		owners = append(owners, *creator)
	}
	return creator, owners, nil
}

func (s *FactoryPlanningSession) lastChatUser() *uuid.UUID {
	for i := len(s.Messages) - 1; i >= 0; i-- {
		message := s.Messages[i]
		if message.Role == PlanningSessionMessageRoleUser && message.UserID != nil {
			id := *message.UserID
			return &id
		}
	}
	return nil
}

func (s *FactoryPlanningSession) recordTaskMessage(tx *gorm.DB, factoryModel *Factory, order *FactoryWorkOrder, activityID uuid.UUID) error {
	body, err := json.Marshal(PlanningTaskMessage{
		WorkOrderID: order.ID.String(),
		Key:         factoryModel.WorkOrderKey(order.Number),
		Title:       order.Title,
	})
	if err != nil {
		return err
	}
	message := PlanningSessionMessage{
		ID:        uuid.New(),
		SessionID: s.ID,
		Role:      PlanningSessionMessageRoleTask,
		Text:      string(body),
		Delivered: true,
		CreatedAt: time.Now(),
	}
	linked, err := s.ownsActivity(tx, activityID)
	if err != nil {
		return err
	}
	if linked {
		message.ActivityID = &activityID
	}
	if err := tx.Create(&message).Error; err != nil {
		return err
	}
	return s.reloadMessages(tx)
}

// ownsActivity reports whether the activity exists and belongs to this
// session. An unknown activity is not an error: the message is stored
// without a link, like agent messages.
func (s *FactoryPlanningSession) ownsActivity(tx *gorm.DB, activityID uuid.UUID) (bool, error) {
	if activityID == uuid.Nil {
		return false, nil
	}
	var activity PlanningSessionActivity
	err := tx.Select("id", "session_id").Where("id = ?", activityID).First(&activity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if activity.SessionID != s.ID {
		return false, fmt.Errorf("%w: activity belongs to a different session", ErrFactoryPlanningSessionInvalid)
	}
	return true, nil
}
