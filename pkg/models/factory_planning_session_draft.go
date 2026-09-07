package models

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FactoryPlanningSessionWorkOrder struct {
	SessionID   uuid.UUID `gorm:"primaryKey"`
	WorkOrderID uuid.UUID `gorm:"primaryKey"`
	CreatedAt   time.Time
}

func (FactoryPlanningSessionWorkOrder) TableName() string {
	return "factory_planning_session_work_orders"
}

func (s *FactoryPlanningSession) ProposeDraft(tx *gorm.DB, draft PlanningSessionDraft) error {
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		title := strings.TrimSpace(draft.Title)
		if title == "" {
			return fmt.Errorf("%w: draft title is required", ErrFactoryPlanningSessionInvalid)
		}
		s.setDraft(PlanningSessionDraft{
			Title:       title,
			Description: strings.TrimSpace(draft.Description),
			WorkOrderID: s.Draft().WorkOrderID,
		})
		return s.saveDraft(inner)
	})
}

func (s *FactoryPlanningSession) UpdateDraft(tx *gorm.DB, draft PlanningSessionDraft) error {
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		if strings.TrimSpace(s.DraftTitle) == "" {
			return ErrFactoryPlanningSessionNoDraft
		}
		title := strings.TrimSpace(draft.Title)
		if title == "" {
			title = s.DraftTitle
		}
		s.setDraft(PlanningSessionDraft{
			Title:       title,
			Description: strings.TrimSpace(draft.Description),
			WorkOrderID: s.Draft().WorkOrderID,
		})
		return s.saveDraft(inner)
	})
}

func (s *FactoryPlanningSession) SkipDraft(tx *gorm.DB) error {
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		if strings.TrimSpace(s.DraftTitle) == "" {
			return ErrFactoryPlanningSessionNoDraft
		}
		s.clearDraft()
		s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindSkipped})
		return s.saveSessionMutation(inner)
	})
}

func (s *FactoryPlanningSession) CreateDraftWorkOrder(tx *gorm.DB, factoryModel *Factory, createdBy uuid.UUID) (*FactoryWorkOrder, error) {
	var order *FactoryWorkOrder
	err := tx.Transaction(func(inner *gorm.DB) error {
		if err := s.lockAndReload(inner); err != nil {
			return err
		}
		if err := s.guardOpen(); err != nil {
			return err
		}
		draft := s.Draft()
		if strings.TrimSpace(draft.Title) == "" {
			return ErrFactoryPlanningSessionNoDraft
		}
		if strings.TrimSpace(draft.WorkOrderID) != "" {
			created, updateErr := s.updateCreatedWorkOrder(inner, factoryModel, draft)
			order = created
			return updateErr
		}

		created, createErr := factoryModel.CreateWorkOrder(inner, draft.Title, draft.Description, &createdBy, []uuid.UUID{createdBy}, nil)
		if createErr != nil {
			return createErr
		}
		if err := inner.Create(&FactoryPlanningSessionWorkOrder{
			SessionID:   s.ID,
			WorkOrderID: created.ID,
			CreatedAt:   time.Now(),
		}).Error; err != nil {
			return err
		}
		s.clearDraft()
		s.resolveWait(PlanningWaitResult{
			Kind:         PlanningWaitKindCreated,
			WorkOrderID:  created.ID.String(),
			WorkOrderKey: factoryModel.WorkOrderKey(created.Number),
			Text:         created.Title,
		})
		if err := s.saveSessionMutation(inner); err != nil {
			return err
		}
		order = created
		return nil
	})
	return order, err
}

func (s *FactoryPlanningSession) applyRefineNote(tx *gorm.DB, text string) (bool, error) {
	key := planningRefineKey(text)
	if key == "" {
		return false, nil
	}
	factoryModel, err := FindFactory(tx, s.OrganizationID, s.FactoryID)
	if err != nil {
		return false, err
	}
	order, err := s.findRefineWorkOrder(tx, factoryModel, key)
	if err != nil {
		return false, err
	}
	if order == nil {
		return false, nil
	}
	if err := s.attachCreatedWorkOrder(tx, order.ID); err != nil {
		return false, err
	}
	s.setDraft(PlanningSessionDraft{
		Title:       order.Title,
		Description: order.Description,
		WorkOrderID: order.ID.String(),
	})
	return true, nil
}

func (s *FactoryPlanningSession) findRefineWorkOrder(tx *gorm.DB, factoryModel *Factory, key string) (*FactoryWorkOrder, error) {
	orders, err := s.CreatedOrders(tx)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		if factoryModel.WorkOrderKey(orders[i].Number) != key {
			continue
		}
		return &orders[i], nil
	}
	return factoryModel.findDraftWorkOrderByKey(tx, key)
}

func (f *Factory) findDraftWorkOrderByKey(tx *gorm.DB, key string) (*FactoryWorkOrder, error) {
	number, ok := f.workOrderNumberFromKey(key)
	if !ok {
		return nil, nil
	}
	var order FactoryWorkOrder
	err := tx.Where("organization_id = ? AND factory_id = ? AND number = ?", f.OrganizationID, f.ID, number).First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if order.State != FactoryWorkOrderStateDraft {
		return nil, nil
	}
	return &order, nil
}

func (f *Factory) workOrderNumberFromKey(key string) (int64, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(key), f.Key+"-")
	if !ok {
		return 0, false
	}
	number, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || number <= 0 {
		return 0, false
	}
	return number, true
}

func (s *FactoryPlanningSession) attachCreatedWorkOrder(tx *gorm.DB, orderID uuid.UUID) error {
	link := FactoryPlanningSessionWorkOrder{
		SessionID:   s.ID,
		WorkOrderID: orderID,
	}
	return tx.Where(link).Attrs(FactoryPlanningSessionWorkOrder{CreatedAt: time.Now()}).FirstOrCreate(&link).Error
}

func (s *FactoryPlanningSession) updateCreatedWorkOrder(
	tx *gorm.DB,
	factoryModel *Factory,
	draft PlanningSessionDraft,
) (*FactoryWorkOrder, error) {
	ids, err := s.CreatedWorkOrderIDs(tx)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(ids, draft.WorkOrderID) {
		return nil, ErrFactoryPlanningSessionInvalid
	}
	orderID, err := uuid.Parse(draft.WorkOrderID)
	if err != nil {
		return nil, ErrFactoryPlanningSessionInvalid
	}
	order, err := factoryModel.FindWorkOrder(tx, orderID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(draft.Title)
	description := strings.TrimSpace(draft.Description)
	if err := order.UpdateContent(tx, &title, &description); err != nil {
		return nil, err
	}
	s.clearDraft()
	s.resolveWait(PlanningWaitResult{
		Kind:         PlanningWaitKindCreated,
		WorkOrderID:  order.ID.String(),
		WorkOrderKey: factoryModel.WorkOrderKey(order.Number),
		Text:         order.Title,
	})
	if err := s.saveSessionMutation(tx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *FactoryPlanningSession) CreatedOrders(tx *gorm.DB) ([]FactoryWorkOrder, error) {
	var links []FactoryPlanningSessionWorkOrder
	if err := tx.Where("session_id = ?", s.ID).Order("created_at ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.WorkOrderID)
	}
	var orders []FactoryWorkOrder
	if err := tx.Where("factory_id = ? AND id IN ?", s.FactoryID, ids).Find(&orders).Error; err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]FactoryWorkOrder, len(orders))
	for _, order := range orders {
		byID[order.ID] = order
	}
	ordered := make([]FactoryWorkOrder, 0, len(ids))
	for _, id := range ids {
		if order, ok := byID[id]; ok {
			ordered = append(ordered, order)
		}
	}
	return ordered, nil
}

func (s *FactoryPlanningSession) CreatedWorkOrderIDs(tx *gorm.DB) ([]string, error) {
	orders, err := s.CreatedOrders(tx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID.String())
	}
	return ids, nil
}
