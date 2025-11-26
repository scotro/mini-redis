package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/scotro/mini-redis/internal/persistence"
	"github.com/scotro/mini-redis/internal/pubsub"
	"github.com/scotro/mini-redis/internal/server"
	"github.com/scotro/mini-redis/internal/store"
)

const defaultSnapshotPath = "dump.rdb"

func main() {
	port := flag.Int("port", 6379, "Port to listen on")
	snapshotPath := flag.String("dbfilename", defaultSnapshotPath, "Path to RDB snapshot file")
	flag.Parse()

	// Create stores for all data types
	stringStore := store.New()
	listStore := store.NewListStore()
	hashStore := store.NewHashStore()
	setStore := store.NewSetStore()

	// Create persistence manager
	stores := persistence.Stores{
		Strings: store.AsSnapshottable(stringStore),
		Lists:   store.AsSnapshottable(listStore),
		Hashes:  store.AsSnapshottable(hashStore),
		Sets:    store.AsSnapshottable(setStore),
	}
	persistMgr := persistence.NewManager(*snapshotPath, stores)

	// Load existing snapshot if present
	if persistMgr.Exists() {
		log.Println("Loading snapshot...")
		result, err := persistMgr.Load()
		if err != nil {
			if !errors.Is(err, persistence.ErrNoSnapshot) {
				log.Printf("Warning: failed to load snapshot: %v", err)
			}
		} else {
			log.Printf("Loaded %d keys (strings=%d, lists=%d, hashes=%d, sets=%d)",
				result.TotalKeys(),
				result.StringKeys,
				result.ListKeys,
				result.HashKeys,
				result.SetKeys,
			)
		}
	}

	// Create PubSub instance
	ps := pubsub.New()

	// Create server with all stores and features
	cfg := server.Config{Port: *port}
	srv := server.New(stringStore, listStore, hashStore, setStore, persistMgr, ps, cfg)

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
