package message

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisKey returns the Redis key for a room's message list.
func redisKey(roomID string) string {
	return "room:" + roomID + ":messages"
}

// RedisStore persists messages in Redis using a list per room.
type RedisStore struct {
	client  redis.Cmdable
	maxSize int64
}

// NewRedisStore creates a RedisStore that retains up to maxSize messages per room.
func NewRedisStore(client redis.Cmdable, maxSize int) *RedisStore {
	return &RedisStore{
		client:  client,
		maxSize: int64(maxSize),
	}
}

// Append adds a message to the room's list in Redis, trimming to maxSize.
func (s *RedisStore) Append(msg *Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("redis: failed to marshal message: %v", err)
		return
	}

	key := redisKey(msg.RoomID)
	pipe := s.client.Pipeline()
	pipe.RPush(ctx, key, data)
	pipe.LTrim(ctx, key, -s.maxSize, -1)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("redis: failed to append message: %v", err)
	}
}

// After returns all messages in a room stored after the message with the given ID.
func (s *RedisStore) After(roomID, afterID string) []*Message {
	if afterID == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	vals, err := s.client.LRange(ctx, redisKey(roomID), 0, -1).Result()
	if err != nil {
		log.Printf("redis: failed to read messages: %v", err)
		return nil
	}

	msgs := make([]*Message, 0, len(vals))
	for _, v := range vals {
		var m Message
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			continue
		}
		msgs = append(msgs, &m)
	}

	for i, m := range msgs {
		if m.ID == afterID {
			result := make([]*Message, len(msgs)-i-1)
			copy(result, msgs[i+1:])
			return result
		}
	}
	return nil
}

// Recent returns the last n messages for a room.
func (s *RedisStore) Recent(roomID string, n int) []*Message {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	vals, err := s.client.LRange(ctx, redisKey(roomID), int64(-n), -1).Result()
	if err != nil {
		log.Printf("redis: failed to read recent messages: %v", err)
		return nil
	}

	if len(vals) == 0 {
		return nil
	}

	msgs := make([]*Message, 0, len(vals))
	for _, v := range vals {
		var m Message
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			continue
		}
		msgs = append(msgs, &m)
	}
	return msgs
}

// DeleteRoom removes all stored messages for a room.
func (s *RedisStore) DeleteRoom(roomID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.client.Del(ctx, redisKey(roomID)).Err(); err != nil {
		log.Printf("redis: failed to delete room messages: %v", err)
	}
}

// Count returns the number of stored messages for a room.
func (s *RedisStore) Count(roomID string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	n, err := s.client.LLen(ctx, redisKey(roomID)).Result()
	if err != nil {
		log.Printf("redis: failed to count messages: %v", err)
		return 0
	}
	return int(n)
}
