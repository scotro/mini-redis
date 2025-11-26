package pubsub

import (
	"testing"
	"time"
)

func TestNewSubscriber(t *testing.T) {
	sub := NewSubscriber("test-id")
	if sub.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", sub.ID)
	}
	if cap(sub.Messages) != MessageBufferSize {
		t.Errorf("expected buffer size %d, got %d", MessageBufferSize, cap(sub.Messages))
	}
}

func TestNew(t *testing.T) {
	ps := New()
	if ps.channels == nil {
		t.Error("channels map should not be nil")
	}
	if ps.patterns == nil {
		t.Error("patterns map should not be nil")
	}
	if ps.subChannels == nil {
		t.Error("subChannels map should not be nil")
	}
	if ps.subPatterns == nil {
		t.Error("subPatterns map should not be nil")
	}
}

func TestSubscribe(t *testing.T) {
	tests := []struct {
		name          string
		channels      []string
		expectedCount int
	}{
		{
			name:          "subscribe to one channel",
			channels:      []string{"channel1"},
			expectedCount: 1,
		},
		{
			name:          "subscribe to multiple channels",
			channels:      []string{"channel1", "channel2", "channel3"},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := New()
			sub := NewSubscriber("sub1")

			count := ps.Subscribe(sub, tt.channels...)
			if count != tt.expectedCount {
				t.Errorf("expected count %d, got %d", tt.expectedCount, count)
			}

			// Verify subscription messages
			for i, ch := range tt.channels {
				select {
				case msg := <-sub.Messages:
					if msg.Type != "subscribe" {
						t.Errorf("expected type 'subscribe', got '%s'", msg.Type)
					}
					if msg.Channel != ch {
						t.Errorf("expected channel '%s', got '%s'", ch, msg.Channel)
					}
					if msg.Count != i+1 {
						t.Errorf("expected count %d, got %d", i+1, msg.Count)
					}
				case <-time.After(100 * time.Millisecond):
					t.Error("timed out waiting for subscribe message")
				}
			}
		})
	}
}

func TestSubscribeDuplicate(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	// Subscribe twice to the same channel
	ps.Subscribe(sub, "channel1")
	// Drain the first message
	<-sub.Messages

	count := ps.Subscribe(sub, "channel1")
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Should still get a subscribe message
	select {
	case msg := <-sub.Messages:
		if msg.Type != "subscribe" {
			t.Errorf("expected type 'subscribe', got '%s'", msg.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for subscribe message")
	}
}

func TestUnsubscribe(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	// Subscribe to channels
	ps.Subscribe(sub, "ch1", "ch2", "ch3")
	// Drain subscribe messages
	for i := 0; i < 3; i++ {
		<-sub.Messages
	}

	// Unsubscribe from one channel
	count := ps.Unsubscribe(sub, "ch2")
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Verify unsubscribe message
	select {
	case msg := <-sub.Messages:
		if msg.Type != "unsubscribe" {
			t.Errorf("expected type 'unsubscribe', got '%s'", msg.Type)
		}
		if msg.Channel != "ch2" {
			t.Errorf("expected channel 'ch2', got '%s'", msg.Channel)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for unsubscribe message")
	}
}

func TestUnsubscribeAll(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	// Subscribe to channels
	ps.Subscribe(sub, "ch1", "ch2")
	// Drain subscribe messages
	for i := 0; i < 2; i++ {
		<-sub.Messages
	}

	// Unsubscribe from all channels
	count := ps.Unsubscribe(sub)
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Should receive 2 unsubscribe messages
	receivedChannels := make(map[string]bool)
	for i := 0; i < 2; i++ {
		select {
		case msg := <-sub.Messages:
			if msg.Type != "unsubscribe" {
				t.Errorf("expected type 'unsubscribe', got '%s'", msg.Type)
			}
			receivedChannels[msg.Channel] = true
		case <-time.After(100 * time.Millisecond):
			t.Error("timed out waiting for unsubscribe message")
		}
	}

	if !receivedChannels["ch1"] || !receivedChannels["ch2"] {
		t.Errorf("expected to receive unsubscribe for both channels, got %v", receivedChannels)
	}
}

func TestUnsubscribeNoSubscriptions(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	// Unsubscribe with no subscriptions
	count := ps.Unsubscribe(sub)
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Should receive an unsubscribe message with empty channel
	select {
	case msg := <-sub.Messages:
		if msg.Type != "unsubscribe" {
			t.Errorf("expected type 'unsubscribe', got '%s'", msg.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for unsubscribe message")
	}
}

func TestPSubscribe(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	count := ps.PSubscribe(sub, "news.*", "weather.*")
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Verify psubscribe messages
	for i, pattern := range []string{"news.*", "weather.*"} {
		select {
		case msg := <-sub.Messages:
			if msg.Type != "psubscribe" {
				t.Errorf("expected type 'psubscribe', got '%s'", msg.Type)
			}
			if msg.Pattern != pattern {
				t.Errorf("expected pattern '%s', got '%s'", pattern, msg.Pattern)
			}
			if msg.Count != i+1 {
				t.Errorf("expected count %d, got %d", i+1, msg.Count)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timed out waiting for psubscribe message")
		}
	}
}

func TestPUnsubscribe(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	ps.PSubscribe(sub, "news.*", "weather.*")
	// Drain psubscribe messages
	for i := 0; i < 2; i++ {
		<-sub.Messages
	}

	count := ps.PUnsubscribe(sub, "news.*")
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

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

func TestPUnsubscribeAll(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	ps.PSubscribe(sub, "news.*", "weather.*")
	// Drain psubscribe messages
	for i := 0; i < 2; i++ {
		<-sub.Messages
	}

	count := ps.PUnsubscribe(sub)
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Should receive 2 punsubscribe messages
	receivedPatterns := make(map[string]bool)
	for i := 0; i < 2; i++ {
		select {
		case msg := <-sub.Messages:
			if msg.Type != "punsubscribe" {
				t.Errorf("expected type 'punsubscribe', got '%s'", msg.Type)
			}
			receivedPatterns[msg.Pattern] = true
		case <-time.After(100 * time.Millisecond):
			t.Error("timed out waiting for punsubscribe message")
		}
	}

	if !receivedPatterns["news.*"] || !receivedPatterns["weather.*"] {
		t.Errorf("expected to receive punsubscribe for both patterns, got %v", receivedPatterns)
	}
}

func TestPublish(t *testing.T) {
	ps := New()
	sub1 := NewSubscriber("sub1")
	sub2 := NewSubscriber("sub2")

	// Subscribe both to the same channel
	ps.Subscribe(sub1, "channel1")
	ps.Subscribe(sub2, "channel1")
	// Drain subscribe messages
	<-sub1.Messages
	<-sub2.Messages

	// Publish a message
	count := ps.Publish("channel1", "hello")
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Both should receive the message
	for _, sub := range []*Subscriber{sub1, sub2} {
		select {
		case msg := <-sub.Messages:
			if msg.Type != "message" {
				t.Errorf("expected type 'message', got '%s'", msg.Type)
			}
			if msg.Channel != "channel1" {
				t.Errorf("expected channel 'channel1', got '%s'", msg.Channel)
			}
			if msg.Payload != "hello" {
				t.Errorf("expected payload 'hello', got '%s'", msg.Payload)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("timed out waiting for message")
		}
	}
}

func TestPublishNoSubscribers(t *testing.T) {
	ps := New()

	count := ps.Publish("channel1", "hello")
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestPublishToPatternSubscribers(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	ps.PSubscribe(sub, "news.*")
	// Drain psubscribe message
	<-sub.Messages

	count := ps.Publish("news.tech", "Tech news!")
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	select {
	case msg := <-sub.Messages:
		if msg.Type != "pmessage" {
			t.Errorf("expected type 'pmessage', got '%s'", msg.Type)
		}
		if msg.Pattern != "news.*" {
			t.Errorf("expected pattern 'news.*', got '%s'", msg.Pattern)
		}
		if msg.Channel != "news.tech" {
			t.Errorf("expected channel 'news.tech', got '%s'", msg.Channel)
		}
		if msg.Payload != "Tech news!" {
			t.Errorf("expected payload 'Tech news!', got '%s'", msg.Payload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for pmessage")
	}
}

func TestPublishToBothChannelAndPattern(t *testing.T) {
	ps := New()
	channelSub := NewSubscriber("channel-sub")
	patternSub := NewSubscriber("pattern-sub")

	ps.Subscribe(channelSub, "news.tech")
	ps.PSubscribe(patternSub, "news.*")
	// Drain subscribe messages
	<-channelSub.Messages
	<-patternSub.Messages

	count := ps.Publish("news.tech", "hello")
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Channel subscriber should get a "message"
	select {
	case msg := <-channelSub.Messages:
		if msg.Type != "message" {
			t.Errorf("expected type 'message', got '%s'", msg.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for message")
	}

	// Pattern subscriber should get a "pmessage"
	select {
	case msg := <-patternSub.Messages:
		if msg.Type != "pmessage" {
			t.Errorf("expected type 'pmessage', got '%s'", msg.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timed out waiting for pmessage")
	}
}

func TestPatternMatching(t *testing.T) {
	tests := []struct {
		pattern string
		channel string
		match   bool
	}{
		{"*", "anything", true},
		{"news.*", "news.tech", true},
		{"news.*", "news.sports", true},
		{"news.*", "weather.today", false},
		{"news.?", "news.a", true},
		{"news.?", "news.ab", false},
		{"news.[abc]", "news.a", true},
		{"news.[abc]", "news.d", false},
		{"*news*", "breaking-news-today", true},
		{"[abc]", "a", true},
		{"[abc]", "d", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.channel, func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.channel)
			if result != tt.match {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.channel, result, tt.match)
			}
		})
	}
}

func TestMessageBufferFull(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	ps.Subscribe(sub, "channel1")
	// Don't drain the subscribe message

	// Fill the buffer
	for i := 0; i < MessageBufferSize; i++ {
		ps.Publish("channel1", "msg")
	}

	// This publish should be dropped (buffer full)
	count := ps.Publish("channel1", "dropped")
	if count != 0 {
		t.Errorf("expected count 0 (message dropped), got %d", count)
	}
}

func TestGetChannelSubscribers(t *testing.T) {
	ps := New()
	sub1 := NewSubscriber("sub1")
	sub2 := NewSubscriber("sub2")

	if ps.GetChannelSubscribers("channel1") != 0 {
		t.Error("expected 0 subscribers for non-existent channel")
	}

	ps.Subscribe(sub1, "channel1")
	ps.Subscribe(sub2, "channel1")
	// Drain messages
	<-sub1.Messages
	<-sub2.Messages

	if ps.GetChannelSubscribers("channel1") != 2 {
		t.Errorf("expected 2 subscribers, got %d", ps.GetChannelSubscribers("channel1"))
	}
}

func TestGetPatternSubscribers(t *testing.T) {
	ps := New()
	sub1 := NewSubscriber("sub1")
	sub2 := NewSubscriber("sub2")

	if ps.GetPatternSubscribers() != 0 {
		t.Error("expected 0 pattern subscribers")
	}

	ps.PSubscribe(sub1, "news.*")
	ps.PSubscribe(sub2, "weather.*")
	// Drain messages
	<-sub1.Messages
	<-sub2.Messages

	if ps.GetPatternSubscribers() != 2 {
		t.Errorf("expected 2 pattern subscribers, got %d", ps.GetPatternSubscribers())
	}
}

func TestGetSubscriberChannels(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	channels := ps.GetSubscriberChannels(sub)
	if channels != nil {
		t.Error("expected nil channels for unsubscribed subscriber")
	}

	ps.Subscribe(sub, "ch1", "ch2")
	// Drain messages
	<-sub.Messages
	<-sub.Messages

	channels = ps.GetSubscriberChannels(sub)
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
}

func TestGetSubscriberPatterns(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	patterns := ps.GetSubscriberPatterns(sub)
	if patterns != nil {
		t.Error("expected nil patterns for unsubscribed subscriber")
	}

	ps.PSubscribe(sub, "news.*", "weather.*")
	// Drain messages
	<-sub.Messages
	<-sub.Messages

	patterns = ps.GetSubscriberPatterns(sub)
	if len(patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(patterns))
	}
}

func TestMixedSubscriptions(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	// Subscribe to both channels and patterns
	ps.Subscribe(sub, "ch1")
	ps.PSubscribe(sub, "news.*")
	// Drain messages
	<-sub.Messages
	<-sub.Messages

	// Total count should be 2
	channels := ps.GetSubscriberChannels(sub)
	patterns := ps.GetSubscriberPatterns(sub)
	if len(channels)+len(patterns) != 2 {
		t.Errorf("expected total of 2 subscriptions, got %d", len(channels)+len(patterns))
	}

	// Unsubscribe from channel
	count := ps.Unsubscribe(sub, "ch1")
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
	<-sub.Messages

	// Unsubscribe from pattern
	count = ps.PUnsubscribe(sub, "news.*")
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestConcurrentPublish(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	ps.Subscribe(sub, "channel1")
	// Drain subscribe message
	<-sub.Messages

	// Publish from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			ps.Publish("channel1", "msg")
			done <- true
		}(i)
	}

	// Wait for all publishes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Count received messages
	count := 0
	timeout := time.After(500 * time.Millisecond)
loop:
	for {
		select {
		case <-sub.Messages:
			count++
		case <-timeout:
			break loop
		}
	}

	if count != 10 {
		t.Errorf("expected 10 messages, got %d", count)
	}
}

func TestConcurrentSubscribeUnsubscribe(t *testing.T) {
	ps := New()
	sub := NewSubscriber("sub1")

	done := make(chan bool)

	// Subscribe and unsubscribe concurrently
	go func() {
		for i := 0; i < 100; i++ {
			ps.Subscribe(sub, "channel1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ps.Unsubscribe(sub, "channel1")
		}
		done <- true
	}()

	// Wait for both to complete without deadlock or panic
	<-done
	<-done
}
