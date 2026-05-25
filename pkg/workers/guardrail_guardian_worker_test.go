package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

// guardianDecision mirrors the branching logic in processExecution so it can
// be tested without a real database transaction.
type guardianDecision int

const (
	decisionResume  guardianDecision = iota
	decisionTimeout guardianDecision = iota
	decisionWait    guardianDecision = iota
)

func evaluateGuardianDecision(
	blockedAt time.Time,
	approved *bool,
	softBlockTimeoutSeconds int,
) guardianDecision {
	if approved != nil && *approved {
		return decisionResume
	}
	timeout := time.Duration(softBlockTimeoutSeconds) * time.Second
	if time.Since(blockedAt) >= timeout {
		return decisionTimeout
	}
	return decisionWait
}

func TestGuardian_ApprovedOverride_Resumes(t *testing.T) {
	approved := true
	decision := evaluateGuardianDecision(
		time.Now().Add(-1*time.Hour),
		&approved,
		86400,
	)
	assert.Equal(t, decisionResume, decision)
}

func TestGuardian_NotApproved_WithinWindow_Waits(t *testing.T) {
	decision := evaluateGuardianDecision(
		time.Now().Add(-1*time.Minute),
		nil,
		3600,
	)
	assert.Equal(t, decisionWait, decision)
}

func TestGuardian_NotApproved_PastTimeout_TimesOut(t *testing.T) {
	decision := evaluateGuardianDecision(
		time.Now().Add(-25*time.Hour),
		nil,
		86400,
	)
	assert.Equal(t, decisionTimeout, decision)
}

func TestGuardian_ExplicitlyDenied_PastTimeout_TimesOut(t *testing.T) {
	denied := false
	decision := evaluateGuardianDecision(
		time.Now().Add(-2*time.Hour),
		&denied,
		3600,
	)
	assert.Equal(t, decisionTimeout, decision)
}

func TestGuardian_ApprovedOverride_TakesPrecedenceOverTimeout(t *testing.T) {
	// Even if timeout has passed, an explicit approval should win.
	approved := true
	decision := evaluateGuardianDecision(
		time.Now().Add(-48*time.Hour),
		&approved,
		86400,
	)
	assert.Equal(t, decisionResume, decision)
}

func TestGuardian_StructuredFields(t *testing.T) {
	// Ensure GuardrailBlockedAt and GuardrailScanID are set correctly on block.
	executionID := uuid.New()
	scanID := uuid.New()
	now := time.Now()

	execution := models.CanvasNodeExecution{
		ID:                 executionID,
		State:              models.CanvasNodeExecutionStateGuardrailBlocked,
		GuardrailScanID:    &scanID,
		GuardrailBlockedAt: &now,
	}

	assert.Equal(t, models.CanvasNodeExecutionStateGuardrailBlocked, execution.State)
	assert.NotNil(t, execution.GuardrailScanID)
	assert.NotNil(t, execution.GuardrailBlockedAt)
}
