package server

import (
	"testing"
	"time"

	"github.com/scotro/mini-redis/internal/pubsub"
	"github.com/scotro/mini-redis/internal/resp"
)

func TestNewPubSubHandler(t *testing.T) {
	ps := pubsub.New()
	handler := NewPubSubHandler(ps)
	if handler == nil {
		t.Error("expected handler to not be nil")
	}
}

func TestHandleSubscribe(t *testing.T) {
	tests := []struct {
		name        string
		args        []resp.Value
		expectError bool
	}{
		{
			name: "subscribe to one channel",
			args: []resp.Value{
				{Type: resp.TypeBulkString, Str: "channel1"},
			},
			expectError: false,
		},
		{
			name: "subscribe to multiple channels",
			args: []resp.Value{
				{Type: resp.TypeBulkString, Str: "channel1"},
				{Type: resp.TypeBulkString, Str: "channel2"},
			},
			expectError: false,
		},
		{
			name:        "no arguments",
			args:        []resp.Value{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := pubsub.New()
			handler := NewPubSubHandler(ps)
			sub := pubsub.NewSubscriber("test-sub")

			result := handler.HandleSubscribe(sub, tt.args)

			if tt.expectError {
				if result.Type != resp.TypeError {
					t.Errorf("expected error response, got type %c", result.Type)
				}
			} else {
				// Verify messages are sent to subscriber
				for range tt.args {
					select {
					case msg := <-sub.Messages:
						if msg.Type != "subscribe" {
							t.Errorf("expected type 'subscribe', got '%s'", msg.Type)
						}
					case <-time.After(100 * time.Millisecond):
						t.Error("timed out waiting for subscribe message")
					}
				}
			}
		})
	}
}

func TestHandleUnsubscribe(t *testing.T) {
	ps := pubsub.New()
	handler := NewPubSubHandler(ps)
	sub := pubsub.NewSubscriber("test-sub")

	// First subscribe
	handler.HandleSubscribe(sub, []resp.Value{
		{Type: resp.TypeBulkString, Str: "channel1"},
		{Type: resp.TypeBulkString, Str: "channel2"},
	})
	// Drain subscribe messages
	<-sub.Messages
	<-sub.Messages

	// Unsubscribe from one channel
	handler.HandleUnsubscribe(sub, []resp.Value{
		{Type: resp.TypeBulkString, Str: "channel1"},
	})

	select {
	case msg := <-sub.Messages:
		if msg.Type != "unsubscribe" {
			t.Errorf("expected type 'unsubscribe', got '%s'", msg.Type)
		}
		if msg.Channel != "channel1" {
			t.Errorf("expected channel 'channel1', got '%s'", msg.Channel)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for unsubscribe message")
	}
}

func TestHandleUnsubscribeAll(t *testing.T) {
	ps := pubsub.New()
	handler := NewPubSubHandler(ps)
	sub := pubsub.NewSubscriber("test-sub")

	// First subscribe
	handler.HandleSubscribe(sub, []resp.Value{
		{Type: resp.TypeBulkString, Str: "channel1"},
		{Type: resp.TypeBulkString, Str: "channel2"},
	})
	// Drain subscribe messages
	<-sub.Messages
	<-sub.Messages

	// Unsubscribe from all
	handler.HandleUnsubscribe(sub, []resp.Value{})

	// Should receive 2 unsubscribe messages
	for i := 0; i < 2; i++ {
		select {
		case msg := <-sub.Messages:
			if msg.Type != "unsubscribe" {
				t.Errorf("expected type 'unsubscribe', got '%s'", msg.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timed out waiting for unsubscribe message")
		}
	}
}

func TestHandlePSubscribe(t *testing.T) {
	tests := []struct {
		name        string
		args        []resp.Value
		expectError bool
	}{
		{
			name: "psubscribe to one pattern",
			args: []resp.Value{
				{Type: resp.TypeBulkString, Str: "news.*"},
			},
			expectError: false,
		},
		{
			name: "psubscribe to multiple patterns",
			args: []resp.Value{
				{Type: resp.TypeBulkString, Str: "news.*"},
				{Type: resp.TypeBulkString, Str: "weather.*"},
			},
			expectError: false,
		},
		{
			name:        "no arguments",
			args:        []resp.Value{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := pubsub.New()
			handler := NewPubSubHandler(ps)
			sub := pubsub.NewSubscriber("test-sub")

			result := handler.HandlePSubscribe(sub, tt.args)

			if tt.expectError {
				if result.Type != resp.TypeError {
					t.Errorf("expected error response, got type %c", result.Type)
				}
			} else {
				// Verify messages are sent to subscriber
				for range tt.args {
					select {
					case msg := <-sub.Messages:
						if msg.Type != "psubscribe" {
							t.Errorf("expected type 'psubscribe', got '%s'", msg.Type)
						}
					case <-time.After(100 * time.Millisecond):
						t.Error("timed out waiting for psubscribe message")
					}
				}
			}
		})
	}
}

func TestHandlePUnsubscribe(t *testing.T) {
	ps := pubsub.New()
	handler := NewPubSubHandler(ps)
	sub := pubsub.NewSubscriber("test-sub")

	// First psubscribe
	handler.HandlePSubscribe(sub, []resp.Value{
		{Type: resp.TypeBulkString, Str: "news.*"},
		{Type: resp.TypeBulkString, Str: "weather.*"},
	})
	// Drain psubscribe messages
	<-sub.Messages
	<-sub.Messages

	// Punsubscribe from one pattern
	handler.HandlePUnsubscribe(sub, []resp.Value{
		{Type: resp.TypeBulkString, Str: "news.*"},
	})

	select {
	case msg := <-sub.Messages:
		if msg.Type != "punsubscribe" {
			t.Errorf("expected type 'punsubscribe', got '%s'", msg.Type)
		}
		if msg.Pattern != "news.*" {
			t.Errorf("expected pattern 'news.*', got '%s'", msg.Pattern)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for punsubscribe message")
	}
}

func TestHandlePublish(t *testing.T) {
	tests := []struct {
		name          string
		args          []resp.Value
		setupSub      bool
		expectError   bool
		expectedCount int
	}{
		{
			name: "publish with subscriber",
			args: []resp.Value{
				{Type: resp.TypeBulkString, Str: "channel1"},
				{Type: resp.TypeBulkString, Str: "hello"},
			},
			setupSub:      true,
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "publish without subscriber",
			args: []resp.Value{
				{Type: resp.TypeBulkString, Str: "channel1"},
				{Type: resp.TypeBulkString, Str: "hello"},
			},
			setupSub:      false,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "publish with wrong number of args",
			args: []resp.Value{
				{Type: resp.TypeBulkString, Str: "channel1"},
			},
			setupSub:    false,
			expectError: true,
		},
		{
			name:        "publish with no args",
			args:        []resp.Value{},
			setupSub:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := pubsub.New()
			handler := NewPubSubHandler(ps)

			var sub *pubsub.Subscriber
			if tt.setupSub {
				sub = pubsub.NewSubscriber("test-sub")
				ps.Subscribe(sub, "channel1")
				// Drain subscribe message
				<-sub.Messages
			}

			result := handler.HandlePublish(tt.args)

			if tt.expectError {
				if result.Type != resp.TypeError {
					t.Errorf("expected error response, got type %c", result.Type)
				}
			} else {
				if result.Type != resp.TypeInteger {
					t.Errorf("expected integer response, got type %c", result.Type)
				}
				if result.Num != tt.expectedCount {
					t.Errorf("expected count %d, got %d", tt.expectedCount, result.Num)
				}
			}
		})
	}
}

func TestFormatSubscribeMessage(t *testing.T) {
	msg := pubsub.Message{
		Type:    "subscribe",
		Channel: "mychan",
		Count:   1,
	}

	result := FormatSubscribeMessage(msg)

	if result.Type != resp.TypeArray {
		t.Errorf("expected array type, got %c", result.Type)
	}
	if len(result.Array) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result.Array))
	}
	if result.Array[0].Str != "subscribe" {
		t.Errorf("expected 'subscribe', got '%s'", result.Array[0].Str)
	}
	if result.Array[1].Str != "mychan" {
		t.Errorf("expected 'mychan', got '%s'", result.Array[1].Str)
	}
	if result.Array[2].Num != 1 {
		t.Errorf("expected 1, got %d", result.Array[2].Num)
	}
}

func TestFormatPSubscribeMessage(t *testing.T) {
	msg := pubsub.Message{
		Type:    "psubscribe",
		Pattern: "news.*",
		Count:   2,
	}

	result := FormatPSubscribeMessage(msg)

	if result.Type != resp.TypeArray {
		t.Errorf("expected array type, got %c", result.Type)
	}
	if len(result.Array) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result.Array))
	}
	if result.Array[0].Str != "psubscribe" {
		t.Errorf("expected 'psubscribe', got '%s'", result.Array[0].Str)
	}
	if result.Array[1].Str != "news.*" {
		t.Errorf("expected 'news.*', got '%s'", result.Array[1].Str)
	}
	if result.Array[2].Num != 2 {
		t.Errorf("expected 2, got %d", result.Array[2].Num)
	}
}

func TestFormatPublishedMessage(t *testing.T) {
	msg := pubsub.Message{
		Type:    "message",
		Channel: "mychan",
		Payload: "hello world",
	}

	result := FormatPublishedMessage(msg)

	if result.Type != resp.TypeArray {
		t.Errorf("expected array type, got %c", result.Type)
	}
	if len(result.Array) != 3 {
		t.Errorf("expected 3 elements, got %d", len(result.Array))
	}
	if result.Array[0].Str != "message" {
		t.Errorf("expected 'message', got '%s'", result.Array[0].Str)
	}
	if result.Array[1].Str != "mychan" {
		t.Errorf("expected 'mychan', got '%s'", result.Array[1].Str)
	}
	if result.Array[2].Str != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", result.Array[2].Str)
	}
}

func TestFormatPMessage(t *testing.T) {
	msg := pubsub.Message{
		Type:    "pmessage",
		Pattern: "news.*",
		Channel: "news.tech",
		Payload: "Tech news!",
	}

	result := FormatPMessage(msg)

	if result.Type != resp.TypeArray {
		t.Errorf("expected array type, got %c", result.Type)
	}
	if len(result.Array) != 4 {
		t.Errorf("expected 4 elements, got %d", len(result.Array))
	}
	if result.Array[0].Str != "pmessage" {
		t.Errorf("expected 'pmessage', got '%s'", result.Array[0].Str)
	}
	if result.Array[1].Str != "news.*" {
		t.Errorf("expected 'news.*', got '%s'", result.Array[1].Str)
	}
	if result.Array[2].Str != "news.tech" {
		t.Errorf("expected 'news.tech', got '%s'", result.Array[2].Str)
	}
	if result.Array[3].Str != "Tech news!" {
		t.Errorf("expected 'Tech news!', got '%s'", result.Array[3].Str)
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name       string
		msg        pubsub.Message
		expectType byte
		expectLen  int
	}{
		{
			name:       "subscribe",
			msg:        pubsub.Message{Type: "subscribe", Channel: "ch", Count: 1},
			expectType: resp.TypeArray,
			expectLen:  3,
		},
		{
			name:       "unsubscribe",
			msg:        pubsub.Message{Type: "unsubscribe", Channel: "ch", Count: 0},
			expectType: resp.TypeArray,
			expectLen:  3,
		},
		{
			name:       "psubscribe",
			msg:        pubsub.Message{Type: "psubscribe", Pattern: "p.*", Count: 1},
			expectType: resp.TypeArray,
			expectLen:  3,
		},
		{
			name:       "punsubscribe",
			msg:        pubsub.Message{Type: "punsubscribe", Pattern: "p.*", Count: 0},
			expectType: resp.TypeArray,
			expectLen:  3,
		},
		{
			name:       "message",
			msg:        pubsub.Message{Type: "message", Channel: "ch", Payload: "hi"},
			expectType: resp.TypeArray,
			expectLen:  3,
		},
		{
			name:       "pmessage",
			msg:        pubsub.Message{Type: "pmessage", Pattern: "p.*", Channel: "p.x", Payload: "hi"},
			expectType: resp.TypeArray,
			expectLen:  4,
		},
		{
			name:       "unknown",
			msg:        pubsub.Message{Type: "unknown"},
			expectType: resp.TypeError,
			expectLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMessage(tt.msg)
			if result.Type != tt.expectType {
				t.Errorf("expected type %c, got %c", tt.expectType, result.Type)
			}
			if tt.expectType == resp.TypeArray && len(result.Array) != tt.expectLen {
				t.Errorf("expected len %d, got %d", tt.expectLen, len(result.Array))
			}
		})
	}
}

func TestIsSubscriptionCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"SUBSCRIBE", true},
		{"UNSUBSCRIBE", true},
		{"PSUBSCRIBE", true},
		{"PUNSUBSCRIBE", true},
		{"PING", true},
		{"QUIT", true},
		{"GET", false},
		{"SET", false},
		{"PUBLISH", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := IsSubscriptionCommand(tt.cmd)
			if result != tt.expected {
				t.Errorf("IsSubscriptionCommand(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestMessageSerializationFormat(t *testing.T) {
	// Test that message format matches Redis specification
	ps := pubsub.New()
	handler := NewPubSubHandler(ps)
	sub := pubsub.NewSubscriber("test-sub")

	// Subscribe and get the message
	handler.HandleSubscribe(sub, []resp.Value{
		{Type: resp.TypeBulkString, Str: "mychan"},
	})

	msg := <-sub.Messages
	formatted := FormatSubscribeMessage(msg)
	serialized := formatted.Serialize()

	// Expected: *3\r\n$9\r\nsubscribe\r\n$6\r\nmychan\r\n:1\r\n
	expected := "*3\r\n$9\r\nsubscribe\r\n$6\r\nmychan\r\n:1\r\n"
	if string(serialized) != expected {
		t.Errorf("expected %q, got %q", expected, string(serialized))
	}
}

func TestPublishedMessageSerializationFormat(t *testing.T) {
	ps := pubsub.New()
	sub := pubsub.NewSubscriber("test-sub")

	ps.Subscribe(sub, "mychan")
	<-sub.Messages // drain subscribe message

	ps.Publish("mychan", "hello")

	msg := <-sub.Messages
	formatted := FormatPublishedMessage(msg)
	serialized := formatted.Serialize()

	// Expected: *3\r\n$7\r\nmessage\r\n$6\r\nmychan\r\n$5\r\nhello\r\n
	expected := "*3\r\n$7\r\nmessage\r\n$6\r\nmychan\r\n$5\r\nhello\r\n"
	if string(serialized) != expected {
		t.Errorf("expected %q, got %q", expected, string(serialized))
	}
}
