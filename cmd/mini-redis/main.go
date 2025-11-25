package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/scotro/mini-redis/internal/server"
	"github.com/scotro/mini-redis/internal/store"
)

func main() {
	port := flag.Int("port", 6379, "Port to listen on")
	flag.Parse()

	// Create store and server
	st := store.New()
	cfg := server.Config{Port: *port}
	srv := server.New(st, cfg)

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	srv.Stop()
	log.Println("Server stopped")
}
