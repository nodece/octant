/*
 * Copyright (c) 2020 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package log

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/rand"
)

// Message is an Octant log message.
type Message struct {
	// Date is the seconds since epoch.
	Date int64
	// LogLevel is the log level.
	LogLevel string
	// Location is the source location.
	Location string
	// Text is the actual message.
	Text string
	// JSON is the JSON payload.
	JSON string
}

// ListenCancelFunc is a function for canceling a sink listener.
type ListenCancelFunc func()

// OctantSinkOption is an option for configuring OctantSink.
type OctantSinkOption func(o *OctantSink)

// OctantSink is an Octant log sink for zap. It creates a method that
// allows multiple loggers to listen to message.
type OctantSink struct {
	listeners map[string]chan Message
	converter func(b []byte) (Message, error)

	mu sync.RWMutex
}

var _ zap.Sink = &OctantSink{}

// NewOctantSink creates an instance of OctantSink.
func NewOctantSink(options ...OctantSinkOption) *OctantSink {
	o := &OctantSink{
		listeners: map[string]chan Message{},
		converter: ConvertBytesToMessage,
	}

	for _, option := range options {
		option(o)
	}

	return o
}

// Write converts the message to a Message and sends it to all listeners.
// The message format is IS8061 date[\t]level[\t]location[\t]text[\t]optional payload[\n]
func (o *OctantSink) Write(p []byte) (n int, err error) {
	m, err := o.converter(p)
	if err != nil {
		return 0, fmt.Errorf("convert bytes to message: %w", err)
	}

	o.send(m)

	return len(p), nil
}

func (o *OctantSink) send(m Message) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for _, ch := range o.listeners {
		ch <- m
	}
}

// Sync is a no-op as.
func (o *OctantSink) Sync() error {
	return nil
}

// Close closes the sink and its listeners.
func (o *OctantSink) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	for k, ch := range o.listeners {
		close(ch)
		delete(o.listeners, k)
	}

	return nil
}

// Listen creates a channel for listening for messages and cancel func.
func (o *OctantSink) Listen() (<-chan Message, ListenCancelFunc) {
	o.mu.Lock()
	defer o.mu.Unlock()

	id := rand.String(6)
	ch := make(chan Message, 1000)
	o.listeners[id] = ch

	return ch, func() {
		o.mu.Lock()
		defer o.mu.Unlock()

		close(ch)

		delete(o.listeners, id)
	}
}

// ConvertBytesToMessage converts a zap message string to a Message instance.
func ConvertBytesToMessage(b []byte) (Message, error) {
	parts := strings.Split(strings.TrimSpace(string(b)), "\t")
	pLen := len(parts)

	if pLen < 4 || pLen > 5 {
		return Message{}, errors.New("unknown log message format")
	}

	t, err := time.Parse("2006-01-02T15:04:05.000Z0700", parts[0])
	if err != nil {
		return Message{}, fmt.Errorf("invalid log timestamp: %w", err)
	}

	m := Message{
		Date:     t.Unix(),
		LogLevel: parts[1],
		Location: parts[2],
		Text:     parts[3],
	}

	if pLen > 4 {
		m.JSON = parts[4]
	}

	return m, nil
}
