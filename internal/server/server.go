// Package server implements the Redis TCP server.
package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

// Config holds server configuration.
type Config struct {
	Port int
}

// DefaultConfig returns the default server configuration.
func DefaultConfig() Config {
	return Config{
		Port: 6379,
	}
}

// Server represents a Redis-compatible TCP server.
type Server struct {
	config      Config
	store       store.Store
	listStore   store.ListStore
	hashStore   store.HashStore
	setStore    store.SetStore
	listHandler *ListCommandHandler
	hashHandler *HashCommands
	listener    net.Listener
	wg          sync.WaitGroup
	quit        chan struct{}
}

// New creates a new server with the given stores and configuration.
func New(s store.Store, listStore store.ListStore, hashStore store.HashStore, setStore store.SetStore, cfg Config) *Server {
	srv := &Server{
		config:    cfg,
		store:     s,
		listStore: listStore,
		hashStore: hashStore,
		setStore:  setStore,
		quit:      make(chan struct{}),
	}
	// Initialize command handlers
	srv.listHandler = NewListCommandHandler(listStore, s)
	srv.hashHandler = NewHashCommands(hashStore, s)
	return srv
}

// Start begins listening for connections.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener
	log.Printf("Mini-Redis server listening on %s", addr)

	go s.acceptConnections()
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
	}
	s.wg.Wait()
}

// Addr returns the server's listener address (useful for testing).
func (s *Server) Addr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.quit:
			return
		default:
		}

		value, err := resp.Parse(reader)
		if err != nil {
			if err != resp.ErrUnexpectedEOF {
				log.Printf("Error parsing command: %v", err)
			}
			return
		}

		response := s.executeCommand(value)
		if _, err := conn.Write(response.Serialize()); err != nil {
			log.Printf("Error writing response: %v", err)
			return
		}
	}
}

func (s *Server) executeCommand(value resp.Value) resp.Value {
	if value.Type != resp.TypeArray || len(value.Array) == 0 {
		return respError("ERR invalid command format")
	}

	cmdVal := value.Array[0]
	if cmdVal.Type != resp.TypeBulkString {
		return respError("ERR invalid command name")
	}

	cmd := strings.ToUpper(cmdVal.Str)
	args := value.Array[1:]

	switch cmd {
	case "PING":
		return s.handlePing(args)
	case "ECHO":
		return s.handleEcho(args)
	case "GET":
		return s.handleGet(args)
	case "SET":
		return s.handleSet(args)
	case "DEL":
		return s.handleDel(args)
	case "EXPIRE":
		return s.handleExpire(args)
	case "TTL":
		return s.handleTTL(args)
	// List commands
	case "LPUSH":
		return s.listHandler.HandleLPush(args)
	case "RPUSH":
		return s.listHandler.HandleRPush(args)
	case "LPOP":
		return s.listHandler.HandleLPop(args)
	case "RPOP":
		return s.listHandler.HandleRPop(args)
	case "LRANGE":
		return s.listHandler.HandleLRange(args)
	case "LLEN":
		return s.listHandler.HandleLLen(args)
	// Hash commands
	case "HSET":
		return s.hashHandler.HandleHSet(args)
	case "HGET":
		return s.hashHandler.HandleHGet(args)
	case "HDEL":
		return s.hashHandler.HandleHDel(args)
	case "HGETALL":
		return s.hashHandler.HandleHGetAll(args)
	case "HKEYS":
		return s.hashHandler.HandleHKeys(args)
	case "HLEN":
		return s.hashHandler.HandleHLen(args)
	// Set commands
	case "SADD":
		return s.handleSAdd(args)
	case "SREM":
		return s.handleSRem(args)
	case "SMEMBERS":
		return s.handleSMembers(args)
	case "SISMEMBER":
		return s.handleSIsMember(args)
	case "SCARD":
		return s.handleSCard(args)
	case "SINTER":
		return s.handleSInter(args)
	default:
		return respError(fmt.Sprintf("ERR unknown command '%s'", cmd))
	}
}

// RESP helper functions
func respSimpleString(s string) resp.Value {
	return resp.Value{Type: resp.TypeSimpleString, Str: s}
}

func respError(s string) resp.Value {
	return resp.Value{Type: resp.TypeError, Str: s}
}

func respInteger(n int) resp.Value {
	return resp.Value{Type: resp.TypeInteger, Num: n}
}

func respBulkString(s string) resp.Value {
	return resp.Value{Type: resp.TypeBulkString, Str: s}
}

func respNullBulkString() resp.Value {
	return resp.Value{Type: resp.TypeBulkString, Null: true}
}

func (s *Server) handlePing(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return respSimpleString("PONG")
	}
	if len(args) == 1 {
		return respBulkString(args[0].Str)
	}
	return respError("ERR wrong number of arguments for 'ping' command")
}

func (s *Server) handleEcho(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'echo' command")
	}
	return respBulkString(args[0].Str)
}

func (s *Server) handleGet(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'get' command")
	}

	key := args[0].Str
	value, exists := s.store.Get(key)
	if !exists {
		return respNullBulkString()
	}
	return respBulkString(value)
}

func (s *Server) handleSet(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return respError("ERR wrong number of arguments for 'set' command")
	}

	key := args[0].Str
	value := args[1].Str

	// Parse options (EX seconds)
	if len(args) > 2 {
		opt := strings.ToUpper(args[2].Str)
		switch opt {
		case "EX":
			if len(args) < 4 {
				return respError("ERR syntax error")
			}
			seconds, err := strconv.Atoi(args[3].Str)
			if err != nil || seconds <= 0 {
				return respError("ERR invalid expire time in 'set' command")
			}
			s.store.SetWithTTL(key, value, time.Duration(seconds)*time.Second)
			return respSimpleString("OK")
		default:
			return respError("ERR syntax error")
		}
	}

	s.store.Set(key, value)
	return respSimpleString("OK")
}

func (s *Server) handleDel(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return respError("ERR wrong number of arguments for 'del' command")
	}

	deleted := 0
	for _, arg := range args {
		if s.store.Delete(arg.Str) {
			deleted++
		}
	}
	return respInteger(deleted)
}

func (s *Server) handleExpire(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return respError("ERR wrong number of arguments for 'expire' command")
	}

	key := args[0].Str
	seconds, err := strconv.Atoi(args[1].Str)
	if err != nil {
		return respError("ERR value is not an integer or out of range")
	}

	// Get current value, then set with TTL
	value, exists := s.store.Get(key)
	if !exists {
		return respInteger(0)
	}

	s.store.SetWithTTL(key, value, time.Duration(seconds)*time.Second)
	return respInteger(1)
}

func (s *Server) handleTTL(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return respError("ERR wrong number of arguments for 'ttl' command")
	}

	key := args[0].Str

	// First check if key exists
	_, exists := s.store.Get(key)
	if !exists {
		return respInteger(-2) // key does not exist
	}

	// Check TTL
	ttl, hasTTL := s.store.TTL(key)
	if !hasTTL {
		return respInteger(-1) // key exists but has no TTL
	}

	return respInteger(int(ttl.Seconds()))
}
