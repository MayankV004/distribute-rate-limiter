package tier

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockResolver is a simple mock for testing the cache.
type mockResolver struct {
	calls  int
	result string
	err    error
}

func (m *mockResolver) Resolve(ctx context.Context, identity string) (string, error) {
	m.calls++
	return m.result, m.err
}

func TestCachedResolver_HitAndMiss(t *testing.T) {
	mock := &mockResolver{result: "premium"}
	cache := NewCached(mock, 1*time.Minute)

	// Miss
	res, err := cache.Resolve(context.Background(), "user1")
	if err != nil || res != "premium" {
		t.Fatalf("expected premium, got %v err: %v", res, err)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls)
	}

	// Hit
	res, err = cache.Resolve(context.Background(), "user1")
	if err != nil || res != "premium" {
		t.Fatalf("expected premium, got %v err: %v", res, err)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call on hit, got %d", mock.calls)
	}
}

func TestCachedResolver_Expiration(t *testing.T) {
	mock := &mockResolver{result: "basic"}
	// Very short TTL
	cache := NewCached(mock, 10*time.Millisecond)

	// Miss
	cache.Resolve(context.Background(), "user2")
	if mock.calls != 1 {
		t.Fatalf("expected 1 call")
	}

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// Miss again (expired)
	cache.Resolve(context.Background(), "user2")
	if mock.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls)
	}
}

func TestCachedResolver_ErrUnknown_NotCached(t *testing.T) {
	mock := &mockResolver{err: ErrUnknown}
	cache := NewCached(mock, 1*time.Minute)

	// First call
	_, err := cache.Resolve(context.Background(), "user3")
	if !errors.Is(err, ErrUnknown) {
		t.Fatalf("expected ErrUnknown, got %v", err)
	}

	// Second call
	_, err = cache.Resolve(context.Background(), "user3")
	if !errors.Is(err, ErrUnknown) {
		t.Fatalf("expected ErrUnknown, got %v", err)
	}

	if mock.calls != 2 {
		t.Fatalf("expected ErrUnknown to not be cached, got %d calls", mock.calls)
	}
}
