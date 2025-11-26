// Package pubsub implements Redis Pub/Sub functionality.
package pubsub

import (
	"path"
	"sync"
)

// MessageBufferSize is the size of the subscriber's message buffer.
// If the buffer is full, messages are dropped (Redis behavior).
const MessageBufferSize = 100

// Message represents a pub/sub message sent to subscribers.
type Message struct {
	Type    string // "message", "pmessage", "subscribe", "unsubscribe", "psubscribe", "punsubscribe"
	Channel string
	Pattern string // for pmessage (pattern-matched messages)
	Payload string
	Count   int // subscription count for subscribe/unsubscribe messages
}

// Subscriber represents a client subscribed to channels or patterns.
type Subscriber struct {
	ID       string
	Messages chan Message
}

// NewSubscriber creates a new subscriber with the given ID.
func NewSubscriber(id string) *Subscriber {
	return &Subscriber{
		ID:       id,
		Messages: make(chan Message, MessageBufferSize),
	}
}

// PubSub manages channel and pattern subscriptions.
type PubSub struct {
	mu       sync.RWMutex
	channels map[string]map[*Subscriber]struct{} // channel -> subscribers
	patterns map[string]map[*Subscriber]struct{} // pattern -> subscribers

	// Track per-subscriber subscriptions for counting
	subChannels map[*Subscriber]map[string]struct{} // subscriber -> channels
	subPatterns map[*Subscriber]map[string]struct{} // subscriber -> patterns
}

// New creates a new PubSub instance.
func New() *PubSub {
	return &PubSub{
		channels:    make(map[string]map[*Subscriber]struct{}),
		patterns:    make(map[string]map[*Subscriber]struct{}),
		subChannels: make(map[*Subscriber]map[string]struct{}),
		subPatterns: make(map[*Subscriber]map[string]struct{}),
	}
}

// subscriptionCount returns the total number of subscriptions for a subscriber.
func (ps *PubSub) subscriptionCount(sub *Subscriber) int {
	count := 0
	if chans, ok := ps.subChannels[sub]; ok {
		count += len(chans)
	}
	if pats, ok := ps.subPatterns[sub]; ok {
		count += len(pats)
	}
	return count
}

// Subscribe subscribes the subscriber to the given channels.
// Returns the total subscription count after subscribing.
func (ps *PubSub) Subscribe(sub *Subscriber, channels ...string) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Initialize subscriber's channel set if needed
	if _, ok := ps.subChannels[sub]; !ok {
		ps.subChannels[sub] = make(map[string]struct{})
	}

	for _, channel := range channels {
		// Skip if already subscribed
		if _, ok := ps.subChannels[sub][channel]; ok {
			// Still send the subscribe message with current count
			ps.sendMessage(sub, Message{
				Type:    "subscribe",
				Channel: channel,
				Count:   ps.subscriptionCount(sub),
			})
			continue
		}

		// Add to channel's subscriber set
		if _, ok := ps.channels[channel]; !ok {
			ps.channels[channel] = make(map[*Subscriber]struct{})
		}
		ps.channels[channel][sub] = struct{}{}

		// Track subscriber's channels
		ps.subChannels[sub][channel] = struct{}{}

		// Send subscribe confirmation
		ps.sendMessage(sub, Message{
			Type:    "subscribe",
			Channel: channel,
			Count:   ps.subscriptionCount(sub),
		})
	}

	return ps.subscriptionCount(sub)
}

// Unsubscribe unsubscribes the subscriber from the given channels.
// If no channels are provided, unsubscribes from all channels.
// Returns the total subscription count after unsubscribing.
func (ps *PubSub) Unsubscribe(sub *Subscriber, channels ...string) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// If no channels specified, unsubscribe from all
	if len(channels) == 0 {
		if subChans, ok := ps.subChannels[sub]; ok {
			channels = make([]string, 0, len(subChans))
			for ch := range subChans {
				channels = append(channels, ch)
			}
		}
		// If still no channels, send a single unsubscribe with count 0
		if len(channels) == 0 {
			ps.sendMessage(sub, Message{
				Type:    "unsubscribe",
				Channel: "",
				Count:   ps.subscriptionCount(sub),
			})
			return ps.subscriptionCount(sub)
		}
	}

	for _, channel := range channels {
		// Remove from channel's subscriber set
		if subs, ok := ps.channels[channel]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(ps.channels, channel)
			}
		}

		// Remove from subscriber's channel set
		if subChans, ok := ps.subChannels[sub]; ok {
			delete(subChans, channel)
			if len(subChans) == 0 {
				delete(ps.subChannels, sub)
			}
		}

		// Send unsubscribe confirmation
		ps.sendMessage(sub, Message{
			Type:    "unsubscribe",
			Channel: channel,
			Count:   ps.subscriptionCount(sub),
		})
	}

	return ps.subscriptionCount(sub)
}

// PSubscribe subscribes the subscriber to the given patterns.
// Returns the total subscription count after subscribing.
func (ps *PubSub) PSubscribe(sub *Subscriber, patterns ...string) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Initialize subscriber's pattern set if needed
	if _, ok := ps.subPatterns[sub]; !ok {
		ps.subPatterns[sub] = make(map[string]struct{})
	}

	for _, pattern := range patterns {
		// Skip if already subscribed
		if _, ok := ps.subPatterns[sub][pattern]; ok {
			// Still send the psubscribe message with current count
			ps.sendMessage(sub, Message{
				Type:    "psubscribe",
				Pattern: pattern,
				Count:   ps.subscriptionCount(sub),
			})
			continue
		}

		// Add to pattern's subscriber set
		if _, ok := ps.patterns[pattern]; !ok {
			ps.patterns[pattern] = make(map[*Subscriber]struct{})
		}
		ps.patterns[pattern][sub] = struct{}{}

		// Track subscriber's patterns
		ps.subPatterns[sub][pattern] = struct{}{}

		// Send psubscribe confirmation
		ps.sendMessage(sub, Message{
			Type:    "psubscribe",
			Pattern: pattern,
			Count:   ps.subscriptionCount(sub),
		})
	}

	return ps.subscriptionCount(sub)
}

// PUnsubscribe unsubscribes the subscriber from the given patterns.
// If no patterns are provided, unsubscribes from all patterns.
// Returns the total subscription count after unsubscribing.
func (ps *PubSub) PUnsubscribe(sub *Subscriber, patterns ...string) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// If no patterns specified, unsubscribe from all
	if len(patterns) == 0 {
		if subPats, ok := ps.subPatterns[sub]; ok {
			patterns = make([]string, 0, len(subPats))
			for pat := range subPats {
				patterns = append(patterns, pat)
			}
		}
		// If still no patterns, send a single punsubscribe with count
		if len(patterns) == 0 {
			ps.sendMessage(sub, Message{
				Type:    "punsubscribe",
				Pattern: "",
				Count:   ps.subscriptionCount(sub),
			})
			return ps.subscriptionCount(sub)
		}
	}

	for _, pattern := range patterns {
		// Remove from pattern's subscriber set
		if subs, ok := ps.patterns[pattern]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(ps.patterns, pattern)
			}
		}

		// Remove from subscriber's pattern set
		if subPats, ok := ps.subPatterns[sub]; ok {
			delete(subPats, pattern)
			if len(subPats) == 0 {
				delete(ps.subPatterns, sub)
			}
		}

		// Send punsubscribe confirmation
		ps.sendMessage(sub, Message{
			Type:    "punsubscribe",
			Pattern: pattern,
			Count:   ps.subscriptionCount(sub),
		})
	}

	return ps.subscriptionCount(sub)
}

// Publish publishes a message to a channel.
// Returns the number of clients that received the message.
func (ps *PubSub) Publish(channel, message string) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	count := 0

	// Send to exact channel subscribers
	if subs, ok := ps.channels[channel]; ok {
		for sub := range subs {
			if ps.sendMessage(sub, Message{
				Type:    "message",
				Channel: channel,
				Payload: message,
			}) {
				count++
			}
		}
	}

	// Send to pattern subscribers
	for pattern, subs := range ps.patterns {
		if matchPattern(pattern, channel) {
			for sub := range subs {
				if ps.sendMessage(sub, Message{
					Type:    "pmessage",
					Pattern: pattern,
					Channel: channel,
					Payload: message,
				}) {
					count++
				}
			}
		}
	}

	return count
}

// sendMessage sends a message to a subscriber.
// Returns true if the message was sent, false if the buffer was full (dropped).
func (ps *PubSub) sendMessage(sub *Subscriber, msg Message) bool {
	select {
	case sub.Messages <- msg:
		return true
	default:
		// Buffer full, drop the message (Redis behavior)
		return false
	}
}

// matchPattern matches a channel name against a Redis glob pattern.
// Supports: * (any sequence), ? (any single char), [abc] (character class)
func matchPattern(pattern, channel string) bool {
	// path.Match implements glob patterns with *, ?, and [abc]
	matched, err := path.Match(pattern, channel)
	if err != nil {
		return false
	}
	return matched
}

// GetChannelSubscribers returns the number of subscribers for a channel.
func (ps *PubSub) GetChannelSubscribers(channel string) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if subs, ok := ps.channels[channel]; ok {
		return len(subs)
	}
	return 0
}

// GetPatternSubscribers returns the number of pattern subscriptions.
func (ps *PubSub) GetPatternSubscribers() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	count := 0
	for _, subs := range ps.patterns {
		count += len(subs)
	}
	return count
}

// GetSubscriberChannels returns the channels a subscriber is subscribed to.
func (ps *PubSub) GetSubscriberChannels(sub *Subscriber) []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if subChans, ok := ps.subChannels[sub]; ok {
		channels := make([]string, 0, len(subChans))
		for ch := range subChans {
			channels = append(channels, ch)
		}
		return channels
	}
	return nil
}

// GetSubscriberPatterns returns the patterns a subscriber is subscribed to.
func (ps *PubSub) GetSubscriberPatterns(sub *Subscriber) []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if subPats, ok := ps.subPatterns[sub]; ok {
		patterns := make([]string, 0, len(subPats))
		for pat := range subPats {
			patterns = append(patterns, pat)
		}
		return patterns
	}
	return nil
}
