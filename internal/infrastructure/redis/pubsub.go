package redisinfra

import (
	"context"
	"encoding/json"
	"fmt"

	"chat-service/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

type PubSub struct {
	client *Client
}

type subscription struct {
	pubSub   *goredis.PubSub
	messages chan domain.Message
}

func NewPubSub(client *Client) *PubSub {
	return &PubSub{client: client}
}

func (p *PubSub) PublishMessage(ctx context.Context, roomID string, message domain.Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return domain.WrapAppError(domain.ErrUnexpectedState, "failed to encode message event", err)
	}

	if err := p.client.Raw().Publish(ctx, roomMessageChannel(roomID), payload).Err(); err != nil {
		return domain.WrapAppError(domain.ErrDependency, "failed to publish message event", err)
	}

	return nil
}

func (p *PubSub) SubscribeMessages(ctx context.Context, roomID string) (domain.MessageSubscription, error) {
	pubSub := p.client.Raw().Subscribe(ctx, roomMessageChannel(roomID))
	if _, err := pubSub.Receive(ctx); err != nil {
		_ = pubSub.Close()
		return nil, domain.WrapAppError(domain.ErrDependency, "failed to subscribe to message events", err)
	}

	sub := &subscription{
		pubSub:   pubSub,
		messages: make(chan domain.Message),
	}

	go sub.forward(ctx)
	return sub, nil
}

func (s *subscription) Messages() <-chan domain.Message {
	return s.messages
}

func (s *subscription) Close() error {
	if s == nil || s.pubSub == nil {
		return nil
	}
	return s.pubSub.Close()
}

func (s *subscription) forward(ctx context.Context) {
	defer close(s.messages)

	channel := s.pubSub.Channel()
	for {
		select {
		case <-ctx.Done():
			_ = s.Close()
			return
		case event, ok := <-channel:
			if !ok {
				return
			}

			var message domain.Message
			if err := json.Unmarshal([]byte(event.Payload), &message); err != nil {
				continue
			}

			select {
			case s.messages <- message:
			case <-ctx.Done():
				_ = s.Close()
				return
			}
		}
	}
}

func roomMessageChannel(roomID string) string {
	return fmt.Sprintf("rooms:%s:messages", roomID)
}
