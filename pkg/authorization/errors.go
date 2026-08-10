package authorization

import "errors"

var (
	// ErrRoleNotFound indicates that a role does not exist in the requested domain.
	ErrRoleNotFound = errors.New("role not found")

	// ErrRoleNotAssignable indicates that a role cannot be assigned to the requested subject.
	ErrRoleNotAssignable = errors.New("role not assignable")
)
