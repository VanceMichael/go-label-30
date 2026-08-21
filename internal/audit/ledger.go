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
	l.events = append(l.events, event)
	l.mu.Unlock()
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
