package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/christopherjohns/chatsphere/internal/server"
	"github.com/redis/go-redis/v9"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	var opts []server.Option
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Fatalf("Failed to connect to Redis at %s: %v", redisAddr, err)
		}
		log.Printf("Connected to Redis at %s", redisAddr)
		opts = append(opts, server.WithRedis(rdb))
	}

	srv := server.New(addr, opts...)
	log.Printf("Starting ChatSphere server on %s", addr)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
