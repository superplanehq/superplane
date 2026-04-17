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
