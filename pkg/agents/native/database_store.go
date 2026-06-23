package native

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/agents/native/llm"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DatabaseStore struct {
	db *gorm.DB
}

type nativeAgentSessionRow struct {
	ProviderSessionID  string `gorm:"primaryKey"`
	History            datatypes.JSONSlice[llm.Message]
	Awaiting           bool
	Interrupted        bool
	Steps              int
	LastToolSignature  string
	RepeatedToolCalls  int
	PendingToolNames   datatypes.JSONMap
	CompactionFailures int
	Summary            string
}

func (nativeAgentSessionRow) TableName() string {
	return "native_agent_sessions"
}

func NewDatabaseStore() *DatabaseStore {
	return &DatabaseStore{db: database.Conn()}
}

func NewDatabaseStoreWithDB(db *gorm.DB) *DatabaseStore {
	return &DatabaseStore{db: db}
}

func (s *DatabaseStore) Save(ctx context.Context, snapshot SessionSnapshot) error {
	row, err := rowFromSnapshot(snapshot)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider_session_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"history":             row.History,
			"awaiting":            row.Awaiting,
			"interrupted":         row.Interrupted,
			"steps":               row.Steps,
			"last_tool_signature": row.LastToolSignature,
			"repeated_tool_calls": row.RepeatedToolCalls,
			"pending_tool_names":  row.PendingToolNames,
			"compaction_failures": row.CompactionFailures,
			"summary":             row.Summary,
			"updated_at":          gorm.Expr("NOW()"),
		}),
	}).Create(row).Error
}

func (s *DatabaseStore) Load(ctx context.Context, providerSessionID string) (*SessionSnapshot, error) {
	var row nativeAgentSessionRow
	err := s.db.WithContext(ctx).
		Where("provider_session_id = ?", providerSessionID).
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errNativeSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return snapshotFromRow(row)
}

func (s *DatabaseStore) Delete(ctx context.Context, providerSessionID string) error {
	return s.db.WithContext(ctx).
		Where("provider_session_id = ?", providerSessionID).
		Delete(&nativeAgentSessionRow{}).
		Error
}

func rowFromSnapshot(snapshot SessionSnapshot) (*nativeAgentSessionRow, error) {
	pending, err := pendingToolNamesMap(snapshot.PendingToolNames)
	if err != nil {
		return nil, err
	}
	return &nativeAgentSessionRow{
		ProviderSessionID:  snapshot.ID,
		History:            datatypes.JSONSlice[llm.Message](cloneMessages(snapshot.History)),
		Awaiting:           snapshot.Awaiting,
		Interrupted:        snapshot.Interrupted,
		Steps:              snapshot.Steps,
		LastToolSignature:  snapshot.LastToolSignature,
		RepeatedToolCalls:  snapshot.RepeatedToolCalls,
		PendingToolNames:   pending,
		CompactionFailures: snapshot.CompactionFailures,
		Summary:            snapshot.Summary,
	}, nil
}

func snapshotFromRow(row nativeAgentSessionRow) (*SessionSnapshot, error) {
	pending := map[string]string{}
	for key, value := range row.PendingToolNames {
		value, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("native agent session %q pending tool %q has non-string name", row.ProviderSessionID, key)
		}
		pending[key] = value
	}
	return &SessionSnapshot{
		ID:                 row.ProviderSessionID,
		History:            cloneMessages([]llm.Message(row.History)),
		Awaiting:           row.Awaiting,
		Interrupted:        row.Interrupted,
		Steps:              row.Steps,
		LastToolSignature:  row.LastToolSignature,
		RepeatedToolCalls:  row.RepeatedToolCalls,
		PendingToolNames:   pending,
		CompactionFailures: row.CompactionFailures,
		Summary:            row.Summary,
	}, nil
}

func pendingToolNamesMap(names map[string]string) (datatypes.JSONMap, error) {
	data, err := json.Marshal(names)
	if err != nil {
		return nil, fmt.Errorf("marshal pending tool names: %w", err)
	}
	var result datatypes.JSONMap
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode pending tool names: %w", err)
	}
	return result, nil
}
