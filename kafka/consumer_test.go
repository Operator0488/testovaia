package kafka

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type mockReader struct {
	fetchCount  int32
	commitCount int32
	closeCount  int32
	fetchMsgs   []Message
	fetchErrs   []error
	commitErrs  []error
	afterFetch  func(ctx context.Context)
}

func (m *mockReader) FetchMessage(ctx context.Context) (Message, error) {
	idx := int(atomic.AddInt32(&m.fetchCount, 1) - 1)
	if m.afterFetch != nil {
		m.afterFetch(ctx)
	}
	if idx < len(m.fetchErrs) && m.fetchErrs[idx] != nil {
		return Message{}, m.fetchErrs[idx]
	}
	if idx < len(m.fetchMsgs) {
		return m.fetchMsgs[idx], nil
	}
	// If nothing left, wait for cancellation
	<-ctx.Done()
	return Message{}, ctx.Err()
}

func (m *mockReader) CommitMessages(ctx context.Context, msgs ...Message) error {
	idx := int(atomic.AddInt32(&m.commitCount, 1) - 1)
	if idx < len(m.commitErrs) && m.commitErrs[idx] != nil {
		return m.commitErrs[idx]
	}
	return nil
}

func (m *mockReader) Close() error {
	atomic.AddInt32(&m.closeCount, 1)
	return nil
}

func TestNewConsumer_Validation(t *testing.T) {
	_, err := newConsumer("", "group", func(context.Context, Message) error { return nil })
	if err == nil {
		t.Fatalf("expected error for empty topic")
	}

	_, err = newConsumer("topic", "group", nil)
	if err == nil {
		t.Fatalf("expected error for nil handler")
	}

	c, err := newConsumer("topic", "group", func(context.Context, Message) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatalf("expected consumer instance")
	}
}

func TestConsumer_Start_Table(t *testing.T) {
	tests := []struct {
		name          string
		setupReader   func(cancel context.CancelFunc) *mockReader
		handler       ConsumeHandler
		expectFetch   int32
		expectCommit  int32
		expectHandler int32
	}{
		{
			name: "fetch error then cancel",
			setupReader: func(cancel context.CancelFunc) *mockReader {
				return &mockReader{
					fetchErrs:  []error{errors.New("fetch failed")},
					afterFetch: func(ctx context.Context) { cancel() },
				}
			},
			handler:     func(ctx context.Context, msg Message) error { return nil },
			expectFetch: 1, expectCommit: 0, expectHandler: 0,
		},
		{
			name: "handler error skips commit",
			setupReader: func(cancel context.CancelFunc) *mockReader {
				return &mockReader{
					fetchMsgs:  []Message{{Topic: "t", Partition: 0, Offset: 1}},
					afterFetch: func(ctx context.Context) { cancel() },
				}
			},
			handler:     func(ctx context.Context, msg Message) error { return errors.New("handler failed") },
			expectFetch: 1, expectCommit: 0, expectHandler: 1,
		},
		{
			name: "commit error is logged and ignored",
			setupReader: func(cancel context.CancelFunc) *mockReader {
				return &mockReader{
					fetchMsgs:  []Message{{Topic: "t", Partition: 0, Offset: 1}},
					commitErrs: []error{errors.New("commit failed")},
					afterFetch: func(ctx context.Context) { cancel() },
				}
			},
			handler:     func(ctx context.Context, msg Message) error { return nil },
			expectFetch: 1, expectCommit: 1, expectHandler: 1,
		},
		{
			name: "happy path one cycle",
			setupReader: func(cancel context.CancelFunc) *mockReader {
				return &mockReader{
					fetchMsgs:  []Message{{Topic: "t", Partition: 0, Offset: 1}},
					afterFetch: func(ctx context.Context) { cancel() },
				}
			},
			handler:     func(ctx context.Context, msg Message) error { return nil },
			expectFetch: 1, expectCommit: 1, expectHandler: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m := tt.setupReader(cancel)
			cons, err := newConsumer("topic", "group", func(ctx context.Context, msg Message) error {
				return tt.handler(ctx, msg)
			})
			if err != nil {
				t.Fatalf("unexpected error creating consumer: %v", err)
			}

			// Inject mock reader
			c := cons
			c.reader = m

			done := make(chan struct{})
			go func() {
				_ = c.Run(ctx)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Start did not exit on time")
			}

			if atomic.LoadInt32(&m.fetchCount) != tt.expectFetch {
				t.Fatalf("fetch count = %d, want %d", m.fetchCount, tt.expectFetch)
			}
			if atomic.LoadInt32(&m.commitCount) != tt.expectCommit {
				t.Fatalf("commit count = %d, want %d", m.commitCount, tt.expectCommit)
			}
			// Count handler invocations via commit+handler logic: we can't directly read handler count.
			// So we re-run with a counting handler for precision in next test.
		})
	}
}

func TestConsumer_Start_HandlerCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &mockReader{
		fetchMsgs:  []Message{{Topic: "t"}},
		afterFetch: func(ctx context.Context) { cancel() },
	}
	var handlerCount int32
	cons, err := newConsumer("topic", "group", func(ctx context.Context, msg Message) error {
		atomic.AddInt32(&handlerCount, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := cons
	c.reader = m

	done := make(chan struct{})
	go func() {
		_ = c.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not exit on time")
	}

	if atomic.LoadInt32(&handlerCount) != 1 {
		t.Fatalf("handler count = %d, want 1", handlerCount)
	}
}

func TestConsumer_Start_ImmediateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m := &mockReader{}
	cons, err := newConsumer("topic", "group", func(ctx context.Context, msg Message) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := cons
	c.reader = m

	if err := c.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestConsumer_Close(t *testing.T) {
	// nil reader
	c := &consumer{}
	if err := c.Close(); err != nil {
		t.Fatalf("unexpected error on nil reader close: %v", err)
	}

	// with reader
	m := &mockReader{}
	c.reader = m
	if err := c.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&m.closeCount) != 1 {
		t.Fatalf("close count = %d, want 1", m.closeCount)
	}
}
