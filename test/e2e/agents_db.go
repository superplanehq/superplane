package e2e

import (
	"fmt"
	"os"
	"sync"

	postgres "gorm.io/driver/postgres"
	gorm "gorm.io/gorm"
)

var (
	agentsDBMu sync.Mutex
	agentsDB   *gorm.DB
)

// agentsDBConn returns a lazily opened gorm connection to the agents_test DB.
// The app process already uses DB_NAME=superplane_test, so we open a separate
// connection here rather than reusing database.Conn().
//
// The handle is only cached on success: if the first attempt fails (e.g. the
// agent container is still starting), subsequent calls will retry rather than
// returning a cached error indefinitely.
func agentsDBConn() (*gorm.DB, error) {
	agentsDBMu.Lock()
	defer agentsDBMu.Unlock()

	if agentsDB != nil {
		return agentsDB, nil
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USERNAME")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("AGENTS_DB_NAME")
	if name == "" {
		name = "agents_test"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable application_name=e2e-agents",
		host, port, user, pass, name,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	agentsDB = db
	return agentsDB, nil
}

// closeAgentsDB releases the cached connection to agents_test. Safe to call
// multiple times; intended for suite-level teardown.
func closeAgentsDB() {
	agentsDBMu.Lock()
	defer agentsDBMu.Unlock()

	if agentsDB == nil {
		return
	}
	if sqlDB, err := agentsDB.DB(); err == nil {
		_ = sqlDB.Close()
	}
	agentsDB = nil
}

// truncateAgentChatTables wipes all chat state from the agents_test DB so tests
// start from a clean slate. It does not touch schema_migrations.
//
// The child tables are listed explicitly alongside the CASCADE so that any new
// child table added to the agent schema in the future shows up as a review
// signal here; CASCADE alone would silently continue to work but might leave
// reviewers guessing whether the test intentionally covers a new table.
//
// NOTE: this truncates every org's chats, not just the current subtest's.
// Agent chat subtests therefore MUST run serially (no t.Parallel()).
func truncateAgentChatTables() error {
	db, err := agentsDBConn()
	if err != nil {
		return err
	}
	return db.Exec(`
		truncate table
			agent_chat_runs,
			agent_chat_messages,
			agent_chats,
			agent_canvas_markdown_memory
		restart identity cascade;
	`).Error
}

// countAgentChatsForCanvas returns the number of agent_chats rows for the given
// (org_id, canvas_id) in the agents_test DB.
func countAgentChatsForCanvas(orgID, canvasID string) (int64, error) {
	db, err := agentsDBConn()
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Raw(
		`SELECT count(*) FROM agent_chats WHERE org_id = ? AND canvas_id = ?`,
		orgID, canvasID,
	).Scan(&count).Error
	return count, err
}

// countAgentChatMessagesForCanvas returns the number of agent_chat_messages
// rows across all chats for the given (org_id, canvas_id) in agents_test.
//
// agent_chat_messages is written by PersistedRunRecorder.save_authoritative_messages
// near the end of _stream_agent_run, so this is the right table to poll for
// "did the run actually persist user + assistant messages?". agent_chats on its
// own is written synchronously in CreateAgentChat before streaming begins, so
// counting it is a near-tautology once the stream has completed.
func countAgentChatMessagesForCanvas(orgID, canvasID string) (int64, error) {
	db, err := agentsDBConn()
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Raw(
		`SELECT count(*) FROM agent_chat_messages m
		 JOIN agent_chats c ON c.id = m.chat_id
		 WHERE c.org_id = ? AND c.canvas_id = ?`,
		orgID, canvasID,
	).Scan(&count).Error
	return count, err
}

// lastRunModelForCanvas returns the model recorded on the most recently created
// agent_chat_runs row for any chat under (org_id, canvas_id). Used as a
// race-free replacement for sniffing the client-side TEST_MODE_HINT in the
// DOM: the hint is written by applyStreamOutcome but then immediately
// overwritten when setCurrentChatId triggers useLoadChatConversation to reload
// messages from the DB, whereas agent_chat_runs.model is persisted by
// PersistedRunRecorder and doesn't move.
//
// When no matching run row exists yet this returns ("", nil) — gorm's Scan
// does not raise ErrRecordNotFound on an empty result set, it just leaves the
// destination at its zero value. Callers should therefore check both err and
// the returned model string; a safe "poll again" predicate is
// `err == nil && model == ""`.
func lastRunModelForCanvas(orgID, canvasID string) (string, error) {
	db, err := agentsDBConn()
	if err != nil {
		return "", err
	}
	var model string
	err = db.Raw(
		`SELECT r.model FROM agent_chat_runs r
		 JOIN agent_chats c ON c.id = r.chat_id
		 WHERE c.org_id = ? AND c.canvas_id = ?
		 ORDER BY r.created_at DESC
		 LIMIT 1`,
		orgID, canvasID,
	).Scan(&model).Error
	return model, err
}
