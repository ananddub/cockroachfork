// Copyright 2025 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package sql

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/kv/kvpb"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
	"github.com/cockroachdb/cockroach/pkg/util/uuid"
)

// QuerySubscription represents an active subscription to query results.
type QuerySubscription struct {
	id             uuid.UUID
	stmt           tree.Statement
	affectedRanges []roachpb.RangeID
	startTime      hlc.Timestamp
	mu             struct {
		syncutil.Mutex
		active bool
	}
}

// SubscriptionManager manages active query subscriptions and coordinates
// with rangefeed to trigger re-execution on data changes.
type SubscriptionManager struct {
	mu struct {
		syncutil.RWMutex
		subscriptions map[uuid.UUID]*QuerySubscription
		rangeToSubs   map[roachpb.RangeID]map[uuid.UUID]struct{}
	}
}

// NewSubscriptionManager creates a new subscription manager.
func NewSubscriptionManager() *SubscriptionManager {
	sm := &SubscriptionManager{}
	sm.mu.subscriptions = make(map[uuid.UUID]*QuerySubscription)
	sm.mu.rangeToSubs = make(map[roachpb.RangeID]map[uuid.UUID]struct{})
	return sm
}

// Subscribe registers a new query subscription.
func (sm *SubscriptionManager) Subscribe(
	ctx context.Context, stmt tree.Statement, ranges []roachpb.RangeID,
) (uuid.UUID, error) {
	sub := &QuerySubscription{
		id:             uuid.MakeV4(),
		stmt:           stmt,
		affectedRanges: ranges,
	}
	sub.mu.active = true

	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.mu.subscriptions[sub.id] = sub
	for _, rangeID := range ranges {
		if sm.mu.rangeToSubs[rangeID] == nil {
			sm.mu.rangeToSubs[rangeID] = make(map[uuid.UUID]struct{})
		}
		sm.mu.rangeToSubs[rangeID][sub.id] = struct{}{}
	}

	return sub.id, nil
}

// Unsubscribe removes a subscription.
func (sm *SubscriptionManager) Unsubscribe(subID uuid.UUID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, ok := sm.mu.subscriptions[subID]
	if !ok {
		return
	}

	sub.mu.Lock()
	sub.mu.active = false
	sub.mu.Unlock()

	for _, rangeID := range sub.affectedRanges {
		delete(sm.mu.rangeToSubs[rangeID], subID)
		if len(sm.mu.rangeToSubs[rangeID]) == 0 {
			delete(sm.mu.rangeToSubs, rangeID)
		}
	}
	delete(sm.mu.subscriptions, subID)
}

// OnRangeChange is called when a range has changes, triggering re-execution
// of affected subscriptions.
func (sm *SubscriptionManager) OnRangeChange(
	ctx context.Context, rangeID roachpb.RangeID, event *kvpb.RangeFeedEvent,
) {
	sm.mu.RLock()
	subs := sm.mu.rangeToSubs[rangeID]
	sm.mu.RUnlock()

	if len(subs) == 0 {
		return
	}

	// TODO: Trigger re-execution for affected subscriptions
	// This will be implemented in Phase 4
}

// GetSubscription returns a subscription by ID.
func (sm *SubscriptionManager) GetSubscription(subID uuid.UUID) (*QuerySubscription, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sub, ok := sm.mu.subscriptions[subID]
	return sub, ok
}
