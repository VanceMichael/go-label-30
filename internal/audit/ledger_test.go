package audit

import (
	"context"
	"sync"
	"testing"
	"time"
)

type authorizerFunc func(context.Context, Event) error

func (fn authorizerFunc) Approve(ctx context.Context, event Event) error { return fn(ctx, event) }

func TestConcurrentAppendKeepsSingleAuditChain(t *testing.T) {
	ledger := NewLedger()
	var entered sync.WaitGroup
	entered.Add(2)
	release := make(chan struct{})
	authorizer := authorizerFunc(func(context.Context, Event) error {
		entered.Done()
		<-release
		return nil
	})
	now := time.Now().UTC()
	drafts := []Event{
		{ID: "event-a", TenantID: "farm", ActorID: "operator-a", Action: "feed.start", ObjectType: "plan", ObjectID: "plan-a", Outcome: "ok", RequestID: "request-a", OccurredAt: now},
		{ID: "event-b", TenantID: "farm", ActorID: "operator-b", Action: "feed.start", ObjectType: "plan", ObjectID: "plan-b", Outcome: "ok", RequestID: "request-b", OccurredAt: now},
	}
	errors := make(chan error, 2)
	for _, draft := range drafts {
		go func(draft Event) { errors <- ledger.Append(context.Background(), draft, authorizer) }(draft)
	}
	entered.Wait()
	close(release)
	for range drafts {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
	events := ledger.Snapshot()
	if len(events) != 2 {
		t.Fatalf("events = %+v", events)
	}
	if err := Verify(events); err != nil {
		t.Fatalf("concurrent append forked audit chain: %v; events=%+v", err, events)
	}
}
