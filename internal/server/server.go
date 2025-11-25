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
	config   Config
	store    store.Store
	listener net.Listener
	wg       sync.WaitGroup
	quit     chan struct{}
}

// New creates a new server with the given store and configuration.
func New(s store.Store, cfg Config) *Server {
	return &Server{
		config: cfg,
		store:  s,
		quit:   make(chan struct{}),
	}
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
		s.listener.Close()
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
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.quit:
			return
		default:
		}

		value, err := resp.Parse(reader)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("Error parsing command: %v", err)
			}
			return
		}

		response := s.executeCommand(value)
		conn.Write(response.Serialize())
	}
}

func (s *Server) executeCommand(value resp.Value) resp.Value {
	if value.Type != resp.TypeArray || len(value.Array) == 0 {
		return resp.Error("ERR invalid command format")
	}

	cmdVal := value.Array[0]
	if cmdVal.Type != resp.TypeBulkString {
		return resp.Error("ERR invalid command name")
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
	default:
		return resp.Error(fmt.Sprintf("ERR unknown command '%s'", cmd))
	}
}

func (s *Server) handlePing(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return resp.SimpleString("PONG")
	}
	if len(args) == 1 {
		return resp.BulkString(args[0].Str)
	}
	return resp.Error("ERR wrong number of arguments for 'ping' command")
}

func (s *Server) handleEcho(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Error("ERR wrong number of arguments for 'echo' command")
	}
	return resp.BulkString(args[0].Str)
}

func (s *Server) handleGet(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Error("ERR wrong number of arguments for 'get' command")
	}

	key := args[0].Str
	value, exists := s.store.Get(key)
	if !exists {
		return resp.NullBulkString()
	}
	return resp.BulkString(value)
}

func (s *Server) handleSet(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Error("ERR wrong number of arguments for 'set' command")
	}

	key := args[0].Str
	value := args[1].Str

	// Parse options (EX seconds)
	if len(args) > 2 {
		for i := 2; i < len(args); i++ {
			opt := strings.ToUpper(args[i].Str)
			switch opt {
			case "EX":
				if i+1 >= len(args) {
					return resp.Error("ERR syntax error")
				}
				seconds, err := strconv.Atoi(args[i+1].Str)
				if err != nil || seconds <= 0 {
					return resp.Error("ERR invalid expire time in 'set' command")
				}
				s.store.SetWithTTL(key, value, time.Duration(seconds)*time.Second)
				return resp.SimpleString("OK")
			default:
				return resp.Error("ERR syntax error")
			}
		}
	}

	s.store.Set(key, value)
	return resp.SimpleString("OK")
}

func (s *Server) handleDel(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return resp.Error("ERR wrong number of arguments for 'del' command")
	}

	deleted := 0
	for _, arg := range args {
		if s.store.Delete(arg.Str) {
			deleted++
		}
	}
	return resp.Integer(deleted)
}

func (s *Server) handleExpire(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return resp.Error("ERR wrong number of arguments for 'expire' command")
	}

	key := args[0].Str
	seconds, err := strconv.Atoi(args[1].Str)
	if err != nil {
		return resp.Error("ERR value is not an integer or out of range")
	}

	if s.store.Expire(key, time.Duration(seconds)*time.Second) {
		return resp.Integer(1)
	}
	return resp.Integer(0)
}

func (s *Server) handleTTL(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Error("ERR wrong number of arguments for 'ttl' command")
	}

	key := args[0].Str
	ttl, exists := s.store.TTL(key)
	if !exists {
		return resp.Integer(-2) // key does not exist
	}
	if ttl < 0 {
		return resp.Integer(-1) // key exists but has no TTL
	}
	return resp.Integer(int(ttl.Seconds()))
}
