// Package server provides pub/sub command handlers for the Redis server.
package server

import (
	"github.com/scotro/mini-redis/internal/pubsub"
	"github.com/scotro/mini-redis/internal/resp"
)

// PubSubHandler handles Redis pub/sub commands.
// This struct is designed to be integrated with Server during the integration phase.
type PubSubHandler struct {
	ps *pubsub.PubSub
}

// NewPubSubHandler creates a new PubSubHandler.
func NewPubSubHandler(ps *pubsub.PubSub) *PubSubHandler {
	return &PubSubHandler{ps: ps}
}

// HandleSubscribe handles the SUBSCRIBE command.
// SUBSCRIBE channel [channel ...]
// Subscribes the client to the specified channels.
func (h *PubSubHandler) HandleSubscribe(sub *pubsub.Subscriber, args []resp.Value) resp.Value {
	if len(args) < 1 {
		return respError("ERR wrong number of arguments for 'subscribe' command")
	}

	channels := make([]string, len(args))
	for i, arg := range args {
		channels[i] = arg.Str
	}

	h.ps.Subscribe(sub, channels...)

	// Response is sent via the subscriber's message channel
	// Return nil/empty response since messages are delivered asynchronously
	return resp.Value{Type: resp.TypeArray, Array: nil}
}

// HandleUnsubscribe handles the UNSUBSCRIBE command.
// UNSUBSCRIBE [channel ...]
// Unsubscribes the client from the specified channels, or all channels if none specified.
func (h *PubSubHandler) HandleUnsubscribe(sub *pubsub.Subscriber, args []resp.Value) resp.Value {
	channels := make([]string, len(args))
	for i, arg := range args {
		channels[i] = arg.Str
	}

	h.ps.Unsubscribe(sub, channels...)

	// Response is sent via the subscriber's message channel
	return resp.Value{Type: resp.TypeArray, Array: nil}
}

// HandlePSubscribe handles the PSUBSCRIBE command.
// PSUBSCRIBE pattern [pattern ...]
// Subscribes the client to the specified patterns.
func (h *PubSubHandler) HandlePSubscribe(sub *pubsub.Subscriber, args []resp.Value) resp.Value {
	if len(args) < 1 {
		return respError("ERR wrong number of arguments for 'psubscribe' command")
	}

	patterns := make([]string, len(args))
	for i, arg := range args {
		patterns[i] = arg.Str
	}

	h.ps.PSubscribe(sub, patterns...)

	// Response is sent via the subscriber's message channel
	return resp.Value{Type: resp.TypeArray, Array: nil}
}

// HandlePUnsubscribe handles the PUNSUBSCRIBE command.
// PUNSUBSCRIBE [pattern ...]
// Unsubscribes the client from the specified patterns, or all patterns if none specified.
func (h *PubSubHandler) HandlePUnsubscribe(sub *pubsub.Subscriber, args []resp.Value) resp.Value {
	patterns := make([]string, len(args))
	for i, arg := range args {
		patterns[i] = arg.Str
	}

	h.ps.PUnsubscribe(sub, patterns...)

	// Response is sent via the subscriber's message channel
	return resp.Value{Type: resp.TypeArray, Array: nil}
}

// HandlePublish handles the PUBLISH command.
// PUBLISH channel message
// Publishes a message to a channel and returns the number of clients that received the message.
func (h *PubSubHandler) HandlePublish(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return respError("ERR wrong number of arguments for 'publish' command")
	}

	channel := args[0].Str
	message := args[1].Str

	count := h.ps.Publish(channel, message)
	return respInteger(count)
}

// FormatSubscribeMessage formats a subscribe/unsubscribe confirmation message as RESP.
func FormatSubscribeMessage(msg pubsub.Message) resp.Value {
	return resp.Value{
		Type: resp.TypeArray,
		Array: []resp.Value{
			respBulkString(msg.Type),
			respBulkString(msg.Channel),
			respInteger(msg.Count),
		},
	}
}

// FormatPSubscribeMessage formats a psubscribe/punsubscribe confirmation message as RESP.
func FormatPSubscribeMessage(msg pubsub.Message) resp.Value {
	return resp.Value{
		Type: resp.TypeArray,
		Array: []resp.Value{
			respBulkString(msg.Type),
			respBulkString(msg.Pattern),
			respInteger(msg.Count),
		},
	}
}

// FormatPublishedMessage formats a published message as RESP.
func FormatPublishedMessage(msg pubsub.Message) resp.Value {
	return resp.Value{
		Type: resp.TypeArray,
		Array: []resp.Value{
			respBulkString(msg.Type),
			respBulkString(msg.Channel),
			respBulkString(msg.Payload),
		},
	}
}

// FormatPMessage formats a pattern-matched published message as RESP.
func FormatPMessage(msg pubsub.Message) resp.Value {
	return resp.Value{
		Type: resp.TypeArray,
		Array: []resp.Value{
			respBulkString(msg.Type),
			respBulkString(msg.Pattern),
			respBulkString(msg.Channel),
			respBulkString(msg.Payload),
		},
	}
}

// FormatMessage formats any pub/sub message as RESP based on its type.
func FormatMessage(msg pubsub.Message) resp.Value {
	switch msg.Type {
	case "subscribe", "unsubscribe":
		return FormatSubscribeMessage(msg)
	case "psubscribe", "punsubscribe":
		return FormatPSubscribeMessage(msg)
	case "message":
		return FormatPublishedMessage(msg)
	case "pmessage":
		return FormatPMessage(msg)
	default:
		return respError("ERR unknown message type")
	}
}

// IsSubscriptionCommand returns true if the command is valid in subscription mode.
// In subscription mode, only SUBSCRIBE, UNSUBSCRIBE, PSUBSCRIBE, PUNSUBSCRIBE, PING, and QUIT are valid.
func IsSubscriptionCommand(cmd string) bool {
	switch cmd {
	case "SUBSCRIBE", "UNSUBSCRIBE", "PSUBSCRIBE", "PUNSUBSCRIBE", "PING", "QUIT":
		return true
	default:
		return false
	}
}
