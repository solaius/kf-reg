package authz

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// mockAuthorizer is a test Authorizer that counts calls and returns a configurable result.
type mockAuthorizer struct {
	allowed bool
	err     error
	calls   atomic.Int64
}

func (m *mockAuthorizer) Authorize(_ context.Context, _ AuthzRequest) (bool, error) {
	m.calls.Add(1)
	return m.allowed, m.err
}

func TestCachedAuthorizer_CacheHit(t *testing.T) {
	inner := &mockAuthorizer{allowed: true}
	cached := NewCachedAuthorizer(inner, 1*time.Minute)

	req := AuthzRequest{
		User:      "alice",
		Resource:  ResourceAssets,
		Verb:      VerbGet,
		Namespace: "team-a",
	}

	// First call — cache miss, calls inner.
	allowed, err := cached.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true")
	}
	if inner.calls.Load() != 1 {
		t.Errorf("inner calls = %d, want 1", inner.calls.Load())
	}

	// Second call — cache hit, should NOT call inner again.
	allowed, err = cached.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true from cache")
	}
	if inner.calls.Load() != 1 {
		t.Errorf("inner calls = %d, want 1 (cache hit should not call inner)", inner.calls.Load())
	}
}

func TestCachedAuthorizer_CacheExpiry(t *testing.T) {
	inner := &mockAuthorizer{allowed: true}
	cached := NewCachedAuthorizer(inner, 10*time.Millisecond)

	req := AuthzRequest{
		User:      "alice",
		Resource:  ResourceAssets,
		Verb:      VerbGet,
		Namespace: "team-a",
	}

	// First call — cache miss.
	_, _ = cached.Authorize(context.Background(), req)
	if inner.calls.Load() != 1 {
		t.Fatalf("inner calls = %d, want 1", inner.calls.Load())
	}

	// Wait for cache to expire.
	time.Sleep(20 * time.Millisecond)

	// Third call — cache expired, calls inner again.
	_, _ = cached.Authorize(context.Background(), req)
	if inner.calls.Load() != 2 {
		t.Errorf("inner calls = %d, want 2 after cache expiry", inner.calls.Load())
	}
}

func TestCachedAuthorizer_DifferentKeys(t *testing.T) {
	inner := &mockAuthorizer{allowed: true}
	cached := NewCachedAuthorizer(inner, 1*time.Minute)

	req1 := AuthzRequest{User: "alice", Resource: ResourceAssets, Verb: VerbGet, Namespace: "team-a"}
	req2 := AuthzRequest{User: "bob", Resource: ResourceAssets, Verb: VerbGet, Namespace: "team-a"}

	_, _ = cached.Authorize(context.Background(), req1)
	_, _ = cached.Authorize(context.Background(), req2)

	// Both are cache misses (different users), so inner should be called twice.
	if inner.calls.Load() != 2 {
		t.Errorf("inner calls = %d, want 2 (different cache keys)", inner.calls.Load())
	}
}

func TestCachedAuthorizer_CachesDenials(t *testing.T) {
	inner := &mockAuthorizer{allowed: false}
	cached := NewCachedAuthorizer(inner, 1*time.Minute)

	req := AuthzRequest{User: "alice", Resource: ResourceAssets, Verb: VerbDelete, Namespace: "team-a"}

	// First call — denied, cached.
	allowed, _ := cached.Authorize(context.Background(), req)
	if allowed {
		t.Error("expected allowed=false")
	}

	// Second call — cache hit, should return cached denial.
	allowed, _ = cached.Authorize(context.Background(), req)
	if allowed {
		t.Error("expected allowed=false from cache")
	}
	if inner.calls.Load() != 1 {
		t.Errorf("inner calls = %d, want 1 (denial should be cached)", inner.calls.Load())
	}
}
