package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ren0503/tycoon/internal/application/audit"
	"github.com/ren0503/tycoon/internal/application/auth"
	"github.com/ren0503/tycoon/internal/infrastructure/security/local"
)

// FakeVerifier returns configured Firebase identities keyed by ID token.
type FakeVerifier struct {
	Identities map[string]auth.VerifiedIdentity
	Errors     map[string]error
	Err        error

	mu    sync.Mutex
	calls int
}

// Verify implements auth.Verifier without contacting Firebase.
func (f *FakeVerifier) Verify(_ context.Context, idToken string) (auth.VerifiedIdentity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.Err != nil {
		return auth.VerifiedIdentity{}, f.Err
	}
	if err, ok := f.Errors[idToken]; ok {
		return auth.VerifiedIdentity{}, err
	}
	identity, ok := f.Identities[idToken]
	if !ok {
		return auth.VerifiedIdentity{}, auth.ErrInvalidFirebaseToken
	}
	return identity, nil
}

// Calls returns the number of verification attempts.
func (f *FakeVerifier) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// FakeClock is a concurrency-safe deterministic auth.Clock.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewFakeClock returns a clock fixed at now.
func NewFakeClock(now time.Time) *FakeClock {
	return &FakeClock{now: now.UTC()}
}

// Now returns the configured time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward.
func (c *FakeClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

// Set changes the clock's current time.
func (c *FakeClock) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now.UTC()
}

// NewTestKeySet returns a random local RSA signer and static key set.
func NewTestKeySet(t *testing.T, kid string) (*local.LocalSigner, *local.StaticKeySet) {
	t.Helper()
	signer, err := local.NewRandomLocalSigner(kid)
	if err != nil {
		t.Fatalf("NewRandomLocalSigner: %v", err)
	}
	keys, err := local.NewStaticKeySet(kid, signer)
	if err != nil {
		t.Fatalf("NewStaticKeySet: %v", err)
	}
	return signer, keys
}

// AuditExecer is the subset of a transaction the recorder accepts.
type AuditExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// RateLimitCall records one fake limiter invocation.
type RateLimitCall struct {
	Key    string
	Limit  int
	Window time.Duration
}

// FakeRateLimiter is a scriptable rate limiter.
type FakeRateLimiter struct {
	Decision auth.RateLimitDecision
	Err      error

	mu    sync.Mutex
	calls []RateLimitCall
}

// Allow records the call and returns the configured result.
func (f *FakeRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (auth.RateLimitDecision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, RateLimitCall{Key: key, Limit: limit, Window: window})
	if f.Err != nil {
		return auth.RateLimitDecision{}, f.Err
	}
	decision := f.Decision
	if !decision.Allowed && decision.RetryAfter == 0 {
		decision = auth.RateLimitDecision{Allowed: true, Remaining: 1}
	}
	return decision, nil
}

func (f *FakeRateLimiter) Close() error {
	return nil
}

// Calls returns a copy of recorded limiter calls.
func (f *FakeRateLimiter) Calls() []RateLimitCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RateLimitCall(nil), f.calls...)
}

// AuditRecorder captures audit entries without a postgres.
type AuditRecorder struct {
	Err error

	mu      sync.Mutex
	entries []audit.Entry
}

// AppendTx records the entry.
func (a *AuditRecorder) AppendTx(_ context.Context, _ AuditExecer, entry audit.Entry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.Err != nil {
		return a.Err
	}
	a.entries = append(a.entries, entry)
	return nil
}

// Entries returns a copy of recorded audit entries.
func (a *AuditRecorder) Entries() []audit.Entry {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]audit.Entry(nil), a.entries...)
}

// Actions returns the recorded audit action names.
func (a *AuditRecorder) Actions() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	actions := make([]string, len(a.entries))
	for i, entry := range a.entries {
		actions[i] = entry.Action
	}
	return actions
}

var (
	_ auth.Verifier    = (*FakeVerifier)(nil)
	_ auth.Clock       = (*FakeClock)(nil)
	_ auth.RateLimiter = (*FakeRateLimiter)(nil)
)
