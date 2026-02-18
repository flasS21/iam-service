package session

import (
	"context"
	"time"
)

// type Session struct {
// 	SessionID string
// 	UserID    string
// 	ExpiresAt time.Time
// }

type Session struct {
	SessionID         string
	UserID            string
	CreatedAt         time.Time
	AbsoluteExpiresAt time.Time
	ExpiresAt         time.Time // current effective expiry (idle-adjusted)
}

type Store interface {
	Create(ctx context.Context, s Session) error
	Get(ctx context.Context, sessionID string) (*Session, error)
	Update(ctx context.Context, s Session) error
	Delete(ctx context.Context, sessionID string) error
}
