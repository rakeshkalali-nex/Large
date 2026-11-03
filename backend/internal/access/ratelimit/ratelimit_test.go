package ratelimit

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"buildestate/backend/internal/platform/database"
	"buildestate/backend/internal/testsupport"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }
func newLimiter(t *testing.T) Limiter {
	t.Helper()
	return Limiter{DB: testsupport.Database(t), Key: bytes.Repeat([]byte{3}, 32), Clock: &testClock{time.Date(2026, 1, 1, 12, 3, 0, 0, time.UTC)}}
}

func TestBudgetsAndExpiry(t *testing.T) {
	for _, op := range []string{Login, Reset, Issuance} {
		t.Run(op, func(t *testing.T) {
			l := newLimiter(t)
			ctx := context.Background()
			for i := 0; i < 5; i++ {
				if d, err := l.Attempt(ctx, op, " User@Example.test ", "peer"); err != nil || d != 0 {
					t.Fatalf("attempt %d: %s %v", i, d, err)
				}
			}
			window := 15 * time.Minute
			if op == Issuance {
				window = time.Hour
			}
			// A new instance shares persistent budgets; normalization cannot bypass.
			restarted := Limiter{DB: l.DB, Key: l.Key, Clock: l.Clock}
			if d, err := restarted.Attempt(ctx, op, "user@example.test", "other-peer"); err != nil || d != window {
				t.Fatalf("expected throttle: %s %v", d, err)
			}
			clock := l.Clock.(*testClock)
			clock.now = clock.now.Add(window - time.Millisecond)
			if d, err := l.Attempt(ctx, op, "USER@example.test", "peer"); err != nil || d != time.Millisecond {
				t.Fatalf("boundary: %s %v", d, err)
			}
			clock.now = clock.now.Add(time.Millisecond)
			if d, err := l.Attempt(ctx, op, "user@example.test", "peer"); err != nil || d != 0 {
				t.Fatalf("expiry: %s %v", d, err)
			}
		})
	}
}

func TestOriginBudgetAndOperationIsolation(t *testing.T) {
	l := newLimiter(t)
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		if d, err := l.Attempt(ctx, Login, fmt.Sprint(i), "peer"); err != nil || d != 0 {
			t.Fatalf("attempt %d: %s %v", i, d, err)
		}
	}
	if d, err := l.Attempt(ctx, Login, "fresh", "peer"); err != nil || d <= 0 {
		t.Fatalf("origin not limited: %s %v", d, err)
	}
	for _, op := range []string{Reset, Issuance} {
		if d, err := l.Attempt(ctx, op, "fresh", "peer"); err != nil || d != 0 {
			t.Fatalf("operation interference: %s %v", d, err)
		}
	}
	if d, err := l.Attempt(ctx, Login, "fresh", "other"); err != nil || d != 0 {
		t.Fatalf("origin interference: %s %v", d, err)
	}
	for _, op := range []string{Login, Reset, Issuance} {
		var leaked int
		if err := l.DB.QueryRow(`SELECT count(*) FROM rate_limit_buckets WHERE operation=? AND subject_digest IN (?,?,?)`, op, "fresh", "peer", "other").Scan(&leaked); err != nil || leaked != 0 {
			t.Fatal("unhashed fixture subjects", leaked, err)
		}
		var hashed int
		if err := l.DB.QueryRow(`SELECT count(*) FROM rate_limit_buckets WHERE operation=? AND subject_digest IN (?,?)`, op, l.digest("identifier", "fresh"), l.digest("origin", "peer")).Scan(&hashed); err != nil || hashed != 2 {
			t.Fatal("missing hashed fixture subjects", hashed, err)
		}
	}
}

func TestConcurrentConnections(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "limiter.sqlite")
	a, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = database.Migrate(ctx, a); err != nil {
		t.Fatal(err)
	}
	b, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	l := Limiter{DB: a, Key: bytes.Repeat([]byte{4}, 32), Clock: &testClock{time.Now()}}
	l2 := l
	l2.DB = b
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			limiter := l
			if i%2 == 1 {
				limiter = l2
			}
			d, e := limiter.Attempt(ctx, Login, "same", fmt.Sprint(i))
			if e != nil {
				t.Error(e)
			} else if d == 0 {
				allowed.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != 5 {
		t.Fatalf("allowed %d, want 5", allowed.Load())
	}
}

func TestAtomicRollbackAndErrors(t *testing.T) {
	l := newLimiter(t)
	ctx := context.Background()
	// Fail the second subject insert to prove the first counter rolls back.
	if _, err := l.DB.Exec(`CREATE TRIGGER reject_origin BEFORE INSERT ON rate_limit_buckets WHEN NEW.operation='login' AND NEW.subject_digest='` + l.digest("origin", "peer") + `' BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Attempt(ctx, Login, "user", "peer"); err == nil {
		t.Fatal("expected injected error")
	}
	var count int
	if err := l.DB.QueryRow(`SELECT count(*) FROM rate_limit_buckets WHERE operation=? AND subject_digest IN (?,?)`, Login, l.digest("identifier", "user"), l.digest("origin", "peer")).Scan(&count); err != nil || count != 0 {
		t.Fatal("partial counters", count, err)
	}
	if _, err := l.Attempt(ctx, "unknown", "user", "peer"); err != ErrOperation {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := l.Attempt(cancelled, Login, "user", "peer"); err == nil {
		t.Fatal("cancelled attempt accepted")
	}
	if _, err := (Limiter{}).Attempt(ctx, Login, "", ""); err != ErrConfiguration {
		t.Fatal(err)
	}
}
