package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

/*
redis.New() initializes a Redis client with connection validation.
Connects to Redis server at specified address with authentication, pings to verify
connectivity within 2-second timeout. Returns initialized Client or error if connection fails.
*/
type Client struct {
	*goredis.Client
}

func New(addr, password string) (*Client, error) {

	client := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()

	if err != nil {
		return nil, err
	}

	return &Client{Client: client}, nil

}
