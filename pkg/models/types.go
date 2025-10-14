package models

import (
	"database/sql/driver"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UUIDArray is a custom type for PostgreSQL UUID arrays
type UUIDArray []uuid.UUID

// Value implements the driver.Valuer interface for database storage
func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}

	stringArray := make([]string, len(a))
	for i, u := range a {
		stringArray[i] = u.String()
	}

	return pq.StringArray(stringArray).Value()
}

// Scan implements the sql.Scanner interface for reading from database
func (a *UUIDArray) Scan(value interface{}) error {
	var stringArray pq.StringArray
	if err := stringArray.Scan(value); err != nil {
		return err
	}

	*a = make(UUIDArray, len(stringArray))
	for i, s := range stringArray {
		u, err := uuid.Parse(s)
		if err != nil {
			return err
		}
		(*a)[i] = u
	}

	return nil
}
