package polar

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func (c *Client) ListAndReconcileOrders(ctx context.Context, tx *gorm.DB, orgID uuid.UUID) ([]Order, error) {
	payloads, err := c.listOrderPayloads(ctx, orgID.String())
	if err != nil {
		return nil, err
	}
	c.reconcileOrderPayloads(ctx, tx, orgID, payloads)

	orders := make([]Order, 0, len(payloads))
	for _, payload := range payloads {
		orders = append(orders, payload.toOrder())
	}
	return orders, nil
}

func (c *Client) reconcileOrderPayloads(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, payloads []orderJSON) {
	for _, payload := range payloads {
		err := c.reconcileOrderPayload(ctx, tx, orgID, payload)
		if err == nil {
			continue
		}
		if IsPermanentApplyError(err) {
			LogPermanentApply(&OrderWebhookEvent{Data: payload.toOrderData()}, err)
			continue
		}
		log.WithError(err).WithFields(log.Fields{
			"organization_id": orgID,
			"polar_order_id":  payload.ID,
		}).Warn("failed to reconcile polar order")
	}
}

func (c *Client) reconcileOrderPayload(ctx context.Context, tx *gorm.DB, orgID uuid.UUID, payload orderJSON) error {
	status := strings.ToLower(strings.TrimSpace(payload.Status))
	if !isReconcileStatus(status) {
		return nil
	}

	grant, err := models.FindLLMCreditGrantByPolarOrderID(tx, payload.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if grant != nil && grant.OrganizationID != orgID {
		return permanentApplyError("polar order already granted to another organization")
	}
	needsRefund := status != "paid"
	if grant != nil && !needsRefund {
		return nil
	}

	data := payload.toOrderData()
	if payload.needsHydration() {
		hydrated, hydrateErr := c.GetOrder(ctx, payload.ID)
		if hydrateErr != nil {
			return hydrateErr
		}
		data = hydrated
	}
	data = withOrganizationID(data, orgID)

	event := &OrderWebhookEvent{Type: orderPaidType, Data: data}
	if err := ApplyOrderPaid(ctx, tx, event, c); err != nil {
		return err
	}
	if !needsRefund {
		return nil
	}
	event.Type = orderRefundedType
	return ApplyOrderRefunded(ctx, tx, event, c)
}

func isReconcileStatus(status string) bool {
	switch status {
	case "paid", "refunded", "partially_refunded":
		return true
	default:
		return false
	}
}
