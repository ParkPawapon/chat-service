package redisinfra

import (
	"context"

	"chat-service/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	client *goredis.Client
}

func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &Client{client: client}, nil
}

func (c *Client) Raw() *goredis.Client {
	return c.client
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}
