package models

import "slices"

func resolveOrderClause(sortBy, sortDirection string, allowedColumns []string, defaultOrder string) string {
	if sortBy == "" || !slices.Contains(allowedColumns, sortBy) {
		return defaultOrder
	}

	direction := "DESC"
	if sortDirection == "asc" {
		direction = "ASC"
	}

	return sortBy + " " + direction
}
