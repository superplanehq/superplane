package models

import "errors"

// SubscriptionAdmissionError requires a correction before the attempt can run.
// Lock contention and database errors do not use this type.
type SubscriptionAdmissionError struct {
	message string
}

func (e *SubscriptionAdmissionError) Error() string {
	return e.message
}

func IsPermanentSubscriptionAdmissionError(err error) bool {
	var admissionError *SubscriptionAdmissionError
	return errors.As(err, &admissionError)
}

func newSubscriptionAdmissionError(message string) error {
	return &SubscriptionAdmissionError{message: message}
}
