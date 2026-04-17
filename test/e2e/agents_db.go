package e2e

import (
	"fmt"
	"os"
	"sync"

	postgres "gorm.io/driver/postgres"
	gorm "gorm.io/gorm"
)

var (
	agentsDBOnce sync.Once
	agentsDB     *gorm.DB
	agentsDBErr  error
)

// agentsDBConn returns a lazily opened gorm connection to the agents_test DB.
// The app process already uses DB_NAME=superplane_test, so we open a separate
// connection here rather than reusing database.Conn().
func agentsDBConn() (*gorm.DB, error) {
	agentsDBOnce.Do(func() {
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
		agentsDB, agentsDBErr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	})
	return agentsDB, agentsDBErr
}

// truncateAgentChatTables wipes all chat state from the agents_test DB so tests
// start from a clean slate. It does not touch schema_migrations.
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
