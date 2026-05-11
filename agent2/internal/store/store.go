package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type ChatSession struct {
	ID                   string
	OrgID                string
	UserID               string
	CanvasID             string
	AnthropicSessionID   string
	InitialMessage       string
	CreatedAt            time.Time
}

func New(dbURL string) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Store{pool: pool}, nil
}

func (s *Store) CreateChat(ctx context.Context, orgID, userID, canvasID, anthropicSessionID string) (*ChatSession, error) {
	chat := &ChatSession{
		OrgID:              orgID,
		UserID:             userID,
		CanvasID:           canvasID,
		AnthropicSessionID: anthropicSessionID,
		CreatedAt:          time.Now(),
	}

	err := s.pool.QueryRow(ctx,
		`INSERT INTO agent2_chat_sessions (org_id, user_id, canvas_id, anthropic_session_id, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		chat.OrgID, chat.UserID, chat.CanvasID, chat.AnthropicSessionID, chat.CreatedAt,
	).Scan(&chat.ID)

	if err != nil {
		return nil, fmt.Errorf("insert chat: %w", err)
	}

	return chat, nil
}

func (s *Store) GetChat(ctx context.Context, orgID, chatID string) (*ChatSession, error) {
	chat := &ChatSession{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, user_id, canvas_id, anthropic_session_id, initial_message, created_at
		 FROM agent2_chat_sessions
		 WHERE id = $1 AND org_id = $2`,
		chatID, orgID,
	).Scan(&chat.ID, &chat.OrgID, &chat.UserID, &chat.CanvasID, &chat.AnthropicSessionID, &chat.InitialMessage, &chat.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("get chat: %w", err)
	}

	return chat, nil
}

func (s *Store) ListChats(ctx context.Context, orgID, userID, canvasID string) ([]*ChatSession, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, user_id, canvas_id, anthropic_session_id, initial_message, created_at
		 FROM agent2_chat_sessions
		 WHERE org_id = $1 AND user_id = $2 AND canvas_id = $3
		 ORDER BY created_at DESC`,
		orgID, userID, canvasID,
	)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	var chats []*ChatSession
	for rows.Next() {
		chat := &ChatSession{}
		if err := rows.Scan(&chat.ID, &chat.OrgID, &chat.UserID, &chat.CanvasID, &chat.AnthropicSessionID, &chat.InitialMessage, &chat.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chat: %w", err)
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

func (s *Store) DeleteChat(ctx context.Context, orgID, chatID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM agent2_chat_sessions WHERE id = $1 AND org_id = $2`,
		chatID, orgID,
	)
	return err
}

func (s *Store) UpdateInitialMessage(ctx context.Context, chatID, message string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE agent2_chat_sessions SET initial_message = $1 WHERE id = $2`,
		message, chatID,
	)
	return err
}
