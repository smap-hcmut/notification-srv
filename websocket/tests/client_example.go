package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/gorilla/websocket"
)

// Example WebSocket client for testing
// Usage: go test -run TestClientExample tests/client_example.go <JWT_TOKEN>
func TestClientExample(t *testing.T) {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run client_example.go <JWT_TOKEN>")
	}

	token := os.Args[1]
	url := fmt.Sprintf("ws://localhost:8081/ws?token=%s", token)

	log.Printf("Connecting to %s...", url)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	log.Println("Connected successfully!")

	// Listen for messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				return
			}
			log.Printf("Received: %s", string(message))
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Disconnecting...")
}
