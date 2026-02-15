package main

import (
	"log"
	"os"

	"github.com/christopherjohns/chatsphere/internal/server"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := server.New(addr)
	log.Printf("Starting ChatSphere server on %s", addr)
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
