package audit

import (
	"context"
	"sync"
)

type Ledger struct {
	mu     sync.Mutex
	events []Event
}

func NewLedger() *Ledger { return &Ledger{} }

// Append appends a draft event to the hash chain tail.
//
// The predecessor is captured without holding the lock during authorization,
// so concurrent appenders can authorize in parallel rather than serializing
// behind the chain tail. That deliberately does not pin the predecessor: if a
// concurrent appender advances the tail while we wait for authorization, the
// captured predecessor is stale. At commit time we re-read the tail and, when
// it no longer matches, relink the event against the current tail before
// appending. The authorization decision covers the actor/action/object fields
// carried by the draft, which are unaffected by chain relinking, so it is not
// repeated. The result is that every successful append lands on a single
// continuous hash chain instead of forking the tail.
func (l *Ledger) Append(ctx context.Context, draft Event, authorizer Authorizer) error {
	l.mu.Lock()
	var previous Event
	if len(l.events) > 0 {
		previous = l.events[len(l.events)-1]
	}
	l.mu.Unlock()

	event, err := NewEvent(draft.ID, draft.TenantID, draft.ActorID, draft.Action, draft.ObjectType, draft.ObjectID, draft.Outcome, draft.RequestID, draft.Details, draft.OccurredAt, previous)
	if err != nil {
		return err
	}
	if authorizer != nil {
		if err := authorizer.Approve(ctx, event); err != nil {
			return err
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	var tail Event
	if len(l.events) > 0 {
		tail = l.events[len(l.events)-1]
	}
	if tail.ID != previous.ID || tail.Hash != previous.Hash {
		// The tail advanced while authorization was in flight. Relink against
		// the current tail so this event extends the chain instead of forking
		// off the stale predecessor.
		event, err = NewEvent(draft.ID, draft.TenantID, draft.ActorID, draft.Action, draft.ObjectType, draft.ObjectID, draft.Outcome, draft.RequestID, draft.Details, draft.OccurredAt, tail)
		if err != nil {
			return err
		}
	}
	l.events = append(l.events, event)
	return nil
}

func (l *Ledger) Snapshot() []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := append([]Event(nil), l.events...)
	for index := range result {
		result[index].Details = cloneDetails(result[index].Details)
	}
	return result
}
