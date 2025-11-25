package server

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/scotro/mini-redis/internal/resp"
	"github.com/scotro/mini-redis/internal/store"
)

func startTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	st := store.New()
	cfg := Config{Port: 0} // Use port 0 to get a random available port
	srv := New(st, cfg)

	if err := srv.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	t.Cleanup(func() {
		srv.Stop()
		st.Close()
	})

	return srv, srv.Addr().String()
}

func sendCommand(t *testing.T, conn net.Conn, args ...string) resp.Value {
	t.Helper()

	// Build array of bulk strings
	cmdArray := make([]resp.Value, len(args))
	for i, arg := range args {
		cmdArray[i] = resp.Value{Type: resp.TypeBulkString, Str: arg}
	}
	cmd := resp.Value{Type: resp.TypeArray, Array: cmdArray}

	_, err := conn.Write(cmd.Serialize())
	if err != nil {
		t.Fatalf("Failed to send command: %v", err)
	}

	reader := bufio.NewReader(conn)
	response, err := resp.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	return response
}

func TestPing(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Test PING without argument
	response := sendCommand(t, conn, "PING")
	if response.Type != resp.TypeSimpleString || response.Str != "PONG" {
		t.Errorf("Expected +PONG, got %v", response)
	}

	// Test PING with argument
	response = sendCommand(t, conn, "PING", "hello")
	if response.Type != resp.TypeBulkString || response.Str != "hello" {
		t.Errorf("Expected $hello, got %v", response)
	}
}

func TestEcho(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	response := sendCommand(t, conn, "ECHO", "hello world")
	if response.Type != resp.TypeBulkString || response.Str != "hello world" {
		t.Errorf("Expected $hello world, got %v", response)
	}
}

func TestGetSet(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Test GET on non-existent key
	response := sendCommand(t, conn, "GET", "mykey")
	if !response.Null {
		t.Errorf("Expected null bulk string, got %v", response)
	}

	// Test SET
	response = sendCommand(t, conn, "SET", "mykey", "myvalue")
	if response.Type != resp.TypeSimpleString || response.Str != "OK" {
		t.Errorf("Expected +OK, got %v", response)
	}

	// Test GET on existing key
	response = sendCommand(t, conn, "GET", "mykey")
	if response.Type != resp.TypeBulkString || response.Str != "myvalue" {
		t.Errorf("Expected $myvalue, got %v", response)
	}
}

func TestSetWithEX(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Set key with 2 second expiry
	response := sendCommand(t, conn, "SET", "expkey", "expvalue", "EX", "2")
	if response.Type != resp.TypeSimpleString || response.Str != "OK" {
		t.Errorf("Expected +OK, got %v", response)
	}

	// Key should exist
	response = sendCommand(t, conn, "GET", "expkey")
	if response.Type != resp.TypeBulkString || response.Str != "expvalue" {
		t.Errorf("Expected $expvalue, got %v", response)
	}

	// TTL should be positive
	response = sendCommand(t, conn, "TTL", "expkey")
	if response.Type != resp.TypeInteger || response.Num <= 0 || response.Num > 2 {
		t.Errorf("Expected TTL between 1 and 2, got %v", response.Num)
	}
}

func TestDel(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Set some keys
	sendCommand(t, conn, "SET", "key1", "val1")
	sendCommand(t, conn, "SET", "key2", "val2")
	sendCommand(t, conn, "SET", "key3", "val3")

	// Delete two keys
	response := sendCommand(t, conn, "DEL", "key1", "key2", "nonexistent")
	if response.Type != resp.TypeInteger || response.Num != 2 {
		t.Errorf("Expected :2, got %v", response)
	}

	// Verify key1 is gone
	response = sendCommand(t, conn, "GET", "key1")
	if !response.Null {
		t.Errorf("Expected null bulk string, got %v", response)
	}

	// Verify key3 still exists
	response = sendCommand(t, conn, "GET", "key3")
	if response.Type != resp.TypeBulkString || response.Str != "val3" {
		t.Errorf("Expected $val3, got %v", response)
	}
}

func TestExpire(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Set a key
	sendCommand(t, conn, "SET", "mykey", "myvalue")

	// Set expire
	response := sendCommand(t, conn, "EXPIRE", "mykey", "10")
	if response.Type != resp.TypeInteger || response.Num != 1 {
		t.Errorf("Expected :1, got %v", response)
	}

	// Check TTL
	response = sendCommand(t, conn, "TTL", "mykey")
	if response.Type != resp.TypeInteger || response.Num <= 0 || response.Num > 10 {
		t.Errorf("Expected TTL between 1 and 10, got %v", response.Num)
	}

	// Expire on non-existent key
	response = sendCommand(t, conn, "EXPIRE", "nonexistent", "10")
	if response.Type != resp.TypeInteger || response.Num != 0 {
		t.Errorf("Expected :0, got %v", response)
	}
}

func TestTTL(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// TTL on non-existent key
	response := sendCommand(t, conn, "TTL", "nonexistent")
	if response.Type != resp.TypeInteger || response.Num != -2 {
		t.Errorf("Expected :-2, got %v", response)
	}

	// Set a key without TTL
	sendCommand(t, conn, "SET", "mykey", "myvalue")

	// TTL on key without expiry
	response = sendCommand(t, conn, "TTL", "mykey")
	if response.Type != resp.TypeInteger || response.Num != -1 {
		t.Errorf("Expected :-1, got %v", response)
	}
}

func TestUnknownCommand(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	response := sendCommand(t, conn, "UNKNOWN")
	if response.Type != resp.TypeError {
		t.Errorf("Expected error response, got %v", response)
	}
}

func TestConcurrentConnections(t *testing.T) {
	_, addr := startTestServer(t)

	numClients := 10
	done := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", clientID, err)
				done <- false
				return
			}
			defer conn.Close()

			key := fmt.Sprintf("key%d", clientID)
			value := fmt.Sprintf("value%d", clientID)

			// Set a key
			response := sendCommand(t, conn, "SET", key, value)
			if response.Type != resp.TypeSimpleString || response.Str != "OK" {
				t.Errorf("Client %d: SET failed", clientID)
				done <- false
				return
			}

			// Get the key back
			response = sendCommand(t, conn, "GET", key)
			if response.Type != resp.TypeBulkString || response.Str != value {
				t.Errorf("Client %d: GET returned wrong value", clientID)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all clients
	for i := 0; i < numClients; i++ {
		select {
		case success := <-done:
			if !success {
				t.Error("One or more clients failed")
			}
		case <-time.After(5 * time.Second):
			t.Error("Test timed out")
		}
	}
}

func TestSetWrongArgs(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// SET with only key
	response := sendCommand(t, conn, "SET", "key")
	if response.Type != resp.TypeError {
		t.Errorf("Expected error response, got %v", response)
	}
}

func TestCaseInsensitiveCommands(t *testing.T) {
	_, addr := startTestServer(t)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Test lowercase
	response := sendCommand(t, conn, "ping")
	if response.Type != resp.TypeSimpleString || response.Str != "PONG" {
		t.Errorf("Expected +PONG, got %v", response)
	}

	// Test mixed case
	response = sendCommand(t, conn, "PiNg")
	if response.Type != resp.TypeSimpleString || response.Str != "PONG" {
		t.Errorf("Expected +PONG, got %v", response)
	}
}
