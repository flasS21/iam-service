package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

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
