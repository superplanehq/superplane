package seatquantity

import log "github.com/sirupsen/logrus"

// LogAlert writes a removal recovery line. Call it only after the transaction commits.
func LogAlert(alert Alert) {
	log.WithFields(log.Fields{
		"organization_id":       alert.OrganizationID.String(),
		"operation_id":          alert.OperationID.String(),
		"polar_subscription_id": alert.SubscriptionID,
		"target_seats":          alert.Target,
		"member_count":          alert.Members,
		"boundary":              alert.Boundary,
		"recovery_status":       alert.RecoveryStatus,
	}).Error(AlertRecoveryKey)
}
