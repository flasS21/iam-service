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
	// Fetch session to know user_id
	s, err := r.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if s == nil {
		return nil
	}

	userKey := r.userKey(s.UserID)

	pipe := r.client.TxPipeline()

	delCmd := pipe.Del(ctx, r.key(sessionID))
	pipe.SRem(ctx, userKey, sessionID)
	cardCmd := pipe.SCard(ctx, userKey)

	log.Printf("[SESSION_DELETE] sid=%s user_id=%s",
		sessionID,
		s.UserID,
	)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// Now safe to inspect cardinality
	if cardCmd.Val() == 0 {
		return r.client.Del(ctx, userKey).Err()
	}

	_ = delCmd // optional, avoids unused warning
	return nil

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

func (r *RedisStore) DeleteAllUserSessions(ctx context.Context, userID string) error {

	userKey := r.userKey(userID)

	// get all session ids for this user
	sessionIDs, err := r.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return err
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := r.client.TxPipeline()

	for _, sid := range sessionIDs {
		pipe.Del(ctx, r.key(sid))
	}

	pipe.Del(ctx, userKey)

	log.Printf("[SESSION_LOGOUT_ALL] user_id=%s sessions=%d",
		userID,
		len(sessionIDs),
	)

	_, err = pipe.Exec(ctx)
	return err
}
