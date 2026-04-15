package core

import "github.com/google/uuid"

type AuthReader interface {
	AuthenticatedUser() *User
	GetUser(id uuid.UUID) (*User, error)
	GetRole(name string) (*RoleRef, error)
	GetGroup(name string) (*GroupRef, error)
	HasRole(role string) (bool, error)
	InGroup(group string) (bool, error)
}
