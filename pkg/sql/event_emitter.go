package sql

import (
	"context"
	"sync"

	"github.com/cockroachdb/cockroach/pkg/util/log"
)

// TableChangeEvent represents a change to a table
type TableChangeEvent struct {
	TableName  string
	Operation  string            // "insert", "update", "delete"
	PrimaryKey map[string]string // PK column -> value
}

// EventEmitter manages listeners for table changes
type EventEmitter struct {
	mu        sync.RWMutex
	listeners map[string][]chan TableChangeEvent // tableName -> channels
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		listeners: make(map[string][]chan TableChangeEvent),
	}
}

// Subscribe adds a listener for a specific table
func (e *EventEmitter) Subscribe(ctx context.Context, tableName string) chan TableChangeEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan TableChangeEvent, 1000000) // 1 million buffer
	e.listeners[tableName] = append(e.listeners[tableName], ch)
	log.VEventf(ctx, 1, "REACTIVE Subscribe: table=%s total_listeners=%d", tableName, len(e.listeners[tableName]))
	return ch
}

// Unsubscribe removes a listener
func (e *EventEmitter) Unsubscribe(tableName string, ch chan TableChangeEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	listeners := e.listeners[tableName]
	for i, listener := range listeners {
		if listener == ch {
			e.listeners[tableName] = append(listeners[:i], listeners[i+1:]...)
			close(ch)
			break
		}
	}
}

// Emit sends an event to all listeners of a table
func (e *EventEmitter) Emit(ctx context.Context, tableName, operation string, primaryKey map[string]string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	event := TableChangeEvent{
		TableName:  tableName,
		Operation:  operation,
		PrimaryKey: primaryKey,
	}

	listeners := e.listeners[tableName]
	log.VEventf(ctx, 1, "REACTIVE Emit: table=%s operation=%s pk=%v listeners=%d", 
		tableName, operation, primaryKey, len(listeners))
	
	if len(listeners) == 0 {
		return
	}

	for i, ch := range listeners {
		select {
		case ch <- event:
			log.VEventf(ctx, 1, "REACTIVE: Event sent to listener %d", i)
		default:
			log.VEventf(ctx, 1, "REACTIVE: Channel full for listener %d", i)
		}
	}
}
