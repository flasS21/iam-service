package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore creates a Redis-backed session store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
		prefix: "session:",
	}
}

func (r *RedisStore) key(sessionID string) string {
	return r.prefix + sessionID
}

func (r *RedisStore) Create(ctx context.Context, s Session) error {
	if s.SessionID == "" || s.UserID == "" {
		return fmt.Errorf("session: missing session_id or user_id")
	}

	ttl := time.Until(s.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session: expires_at must be in the future")
	}

	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("session: failed to marshal: %w", err)
	}

	pipe := r.client.TxPipeline()

	pipe.Set(ctx, r.key(s.SessionID), data, ttl)
	pipe.SAdd(ctx, r.userKey(s.UserID), s.SessionID)

	log.Printf("[SESSION_CREATE] sid=%s user_id=%s expires_at=%s",
		s.SessionID,
		s.UserID,
		s.ExpiresAt.UTC(),
	)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	val, err := r.client.Get(ctx, r.key(sessionID)).Result()
	if err == redis.Nil {
		return nil, nil // not found
	}
	if err != nil {
		return nil, err
	}

	var s Session
	if err := json.Unmarshal([]byte(val), &s); err != nil {
		return nil, fmt.Errorf("session: failed to unmarshal: %w", err)
	}

	return &s, nil
}

func (r *RedisStore) Delete(ctx context.Context, sessionID string) error {
	// First fetch session to know user_id
	s, err := r.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if s == nil {
		return nil // already gone (idempotent)
	}

	pipe := r.client.TxPipeline()

	pipe.Del(ctx, r.key(sessionID))
	pipe.SRem(ctx, r.userKey(s.UserID), sessionID)

	log.Printf("[SESSION_DELETE] sid=%s user_id=%s",
		sessionID,
		s.UserID,
	)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisStore) Update(ctx context.Context, s Session) error {
	if s.SessionID == "" {
		return fmt.Errorf("session: missing session_id")
	}

	ttl := time.Until(s.ExpiresAt)
	if ttl <= 0 {
		// If expired, delete session instead of extending
		return r.client.Del(ctx, r.key(s.SessionID)).Err()
	}

	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("session: failed to marshal: %w", err)
	}

	log.Printf("[SESSION_UPDATE] sid=%s new_expiry=%s",
		s.SessionID,
		s.ExpiresAt.UTC(),
	)

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, r.key(s.SessionID), data, ttl)
	_, err = pipe.Exec(ctx)

	return err
}

func (r *RedisStore) userKey(userID string) string {
	return "user_sessions:" + userID
}

func (r *RedisStore) Client() *redis.Client {
	return r.client
}
