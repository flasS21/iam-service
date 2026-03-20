package session

import (
	"context"
	"time"
)

type Session struct {
	SessionID         string
	UserID            string
	CreatedAt         time.Time
	AbsoluteExpiresAt time.Time
	ExpiresAt         time.Time
	Version           int
	CSRFToken         string `json:"csrf_token"`
}

type Store interface {
	Create(ctx context.Context, s Session) error
	Get(ctx context.Context, sessionID string) (*Session, error)
	Update(ctx context.Context, s Session) error
	Delete(ctx context.Context, sessionID string) error
	DeleteAllUserSessions(ctx context.Context, userID string) error
}
