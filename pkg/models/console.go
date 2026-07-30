package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DefaultConsolePageID is the id used for the implicit first page. Legacy
// single-page consoles that get wrapped into the new pages shape reuse
// this id so the round-trip stays stable.
const DefaultConsolePageID = "main"

// DefaultConsolePageName is the human label paired with DefaultConsolePageID
// when a legacy console is wrapped into a multi-page representation.
const DefaultConsolePageName = "Main"

type ConsolePanel struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Content map[string]any `json:"content"`
}

type ConsoleLayoutItem struct {
	I    string `json:"i"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	MinW *int   `json:"minW,omitempty"`
	MinH *int   `json:"minH,omitempty"`
}

// ConsolePage groups panels and their grid layout under one tab. The
// canvas console stores an ordered list of pages; a fresh canvas has
// zero pages and materializes an implicit one on first save.
type ConsolePage struct {
	ID     string              `json:"id"`
	Name   string              `json:"name,omitempty"`
	Panels []ConsolePanel      `json:"panels"`
	Layout []ConsoleLayoutItem `json:"layout"`
}

func consolePagesData(pages datatypes.JSONType[[]ConsolePage]) []ConsolePage {
	if data := pages.Data(); data != nil {
		return data
	}
	return []ConsolePage{}
}

func copyVersionConsoleFields(source *CanvasVersion, target *CanvasVersion) {
	if source == nil || target == nil {
		return
	}

	target.ConsolePages = datatypes.NewJSONType(consolePagesData(source.ConsolePages))
}

// UpdateCanvasVersionConsoleInTransaction replaces the console pages on the
// given version. `pages` may be empty (fresh / cleared console); callers
// that want a normalized single-page representation should assemble it
// before calling.
func UpdateCanvasVersionConsoleInTransaction(
	tx *gorm.DB,
	version *CanvasVersion,
	pages []ConsolePage,
) (*CanvasVersion, error) {
	if pages == nil {
		pages = []ConsolePage{}
	}

	now := time.Now()
	version.ConsolePages = datatypes.NewJSONType(pages)
	version.UpdatedAt = &now

	if err := tx.Save(version).Error; err != nil {
		return nil, err
	}

	return version, nil
}

func UpsertCanvasVersionConsoleInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	pages []ConsolePage,
) (*CanvasVersion, error) {
	version, err := FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return nil, err
	}

	return UpdateCanvasVersionConsoleInTransaction(tx, version, pages)
}

func UpsertCanvasVersionConsole(canvasID uuid.UUID, pages []ConsolePage) (*CanvasVersion, error) {
	return UpsertCanvasVersionConsoleInTransaction(database.Conn(), canvasID, pages)
}
