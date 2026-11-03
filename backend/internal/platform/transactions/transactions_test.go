package transactions_test

import (
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/platform/database"
	tr "buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
	fixtures "buildestate/backend/internal/testsupport/transactions"
	"context"
	"database/sql"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type audit struct{ fail bool }

func (a audit) Append(ctx context.Context, tx contracts.Transaction, e contracts.AuditEvent) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(id,actor_id,occurred_at,action,project_id,record_type,record_id,request_id,changes_json,retention_class) VALUES(?,?,?,?,?,?,?,?,'[]','business')`, tr.NewID(), e.ActorID, tr.Timestamp(e.OccurredAt), e.Action, e.ProjectID, e.RecordType, e.RecordID, e.RequestID)
	if err != nil {
		return err
	}
	if a.fail {
		return errors.New("injected after audit insert")
	}
	return nil
}
func identity(actor string) contracts.MutationIdentity {
	digest, _ := tr.Digest(struct{ Version int64 }{1})
	return contracts.MutationIdentity{ActorID: actor, RequestID: tr.NewID(), Operation: "site-log.approve", IdempotencyKey: tr.NewID(), RequestDigest: digest}
}
func allow(context.Context, contracts.Transaction) error { return nil }
func review(f fixtures.Fixture, id contracts.MutationIdentity) func(context.Context, contracts.Transaction) (contracts.CommittedResult, contracts.AuditEvent, error) {
	return func(ctx context.Context, tx contracts.Transaction) (contracts.CommittedResult, contracts.AuditEvent, error) {
		return tr.Decide(ctx, tx, tr.Review{Kind: "site-log", ID: f.Log, ActorID: id.ActorID, RequestID: id.RequestID, Decision: "approve", Version: 1, At: time.Now()}, func(ctx context.Context, tx contracts.Transaction, project, from, to string) error {
			result, err := tx.ExecContext(ctx, `UPDATE site_logs SET status=?,version=version+1,updated_at=? WHERE id=? AND project_id=? AND status=? AND version=1`, to, tr.Timestamp(time.Now()), f.Log, project, from)
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return tr.Conflict("STALE_VERSION")
			}
			return nil
		})
	}
}
func count(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAtomicReviewReplayAndRollback(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	id := identity(f.Reviewer)
	service := tr.Service{Transactions: tr.Manager{DB: db}, Idempotency: tr.Store{}, Audit: audit{true}}
	if _, err := service.Mutate(context.Background(), id, allow, review(f, id)); err == nil {
		t.Fatal("fault committed")
	}
	if count(t, db, "SELECT version FROM site_logs WHERE id=?", f.Log) != 1 || count(t, db, "SELECT count(*) FROM review_decisions WHERE site_log_id=?", f.Log) != 0 || count(t, db, "SELECT count(*) FROM audit_events WHERE record_id=?", f.Log) != 0 || count(t, db, "SELECT count(*) FROM idempotency_records WHERE key=?", id.IdempotencyKey) != 0 {
		t.Fatal("partial commit")
	}
	service.Audit = audit{}
	first, err := service.Mutate(context.Background(), id, allow, review(f, id))
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Mutate(context.Background(), id, allow, review(f, id))
	if err != nil || first != second {
		t.Fatalf("replay: %+v %v", second, err)
	}
	if count(t, db, "SELECT count(*) FROM review_decisions WHERE site_log_id=?", f.Log) != 1 || count(t, db, "SELECT count(*) FROM audit_events WHERE record_id=?", f.Log) != 1 {
		t.Fatal("duplicate effect")
	}
	denied := errors.New("current access denied")
	if _, err = service.Mutate(context.Background(), id, func(context.Context, contracts.Transaction) error { return denied }, review(f, id)); !errors.Is(err, denied) {
		t.Fatal("replay bypassed access")
	}
	id.RequestDigest = strings.Repeat("a", 64)
	_, err = service.Mutate(context.Background(), id, allow, review(f, id))
	assertCode(t, err, "IDEMPOTENCY_MISMATCH")
	id = identity(f.Reviewer)
	_, err = service.Mutate(context.Background(), id, allow, review(f, id))
	assertCode(t, err, "STALE_VERSION")
	if _, err = db.Exec("DELETE FROM projects WHERE id=?", f.Project); err == nil {
		t.Fatal("lost referenced project")
	}
}
func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	var e *tr.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

func TestCreatorAndQueuePermissions(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	service := tr.Service{Transactions: tr.Manager{DB: db}, Idempotency: tr.Store{}, Audit: audit{}}
	id := identity(f.Creator)
	_, err := service.Mutate(context.Background(), id, allow, review(f, id))
	assertCode(t, err, "INDEPENDENT_REVIEW_REQUIRED")
	eligible := func(actor string) int {
		p, args, err := tr.ReviewPredicate("site-log", actor)
		if err != nil {
			t.Fatal(err)
		}
		args = append([]any{f.Log}, args...)
		return count(t, db, "SELECT count(*) FROM site_logs r WHERE r.id=? AND "+p, args...)
	}
	if eligible(f.Creator) != 0 || eligible(f.Reviewer) != 1 {
		t.Fatal("administrator separation")
	}
	if _, err = db.Exec("UPDATE users SET role_code='project-manager' WHERE id=?", f.Reviewer); err != nil {
		t.Fatal(err)
	}
	if eligible(f.Reviewer) != 0 {
		t.Fatal("unassigned reviewer")
	}
	if _, err = db.Exec("INSERT INTO user_projects(user_id,project_id,created_by) VALUES(?,?,?)", f.Reviewer, f.Project, f.Creator); err != nil {
		t.Fatal(err)
	}
	if eligible(f.Reviewer) != 1 {
		t.Fatal("assigned reviewer excluded")
	}
	if _, err = db.Exec("UPDATE users SET status='deactivated' WHERE id=?", f.Reviewer); err != nil {
		t.Fatal(err)
	}
	if eligible(f.Reviewer) != 0 {
		t.Fatal("inactive reviewer")
	}
}

func TestConcurrentDuplicateAcrossConnections(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	var seq int
	var name, path string
	if err := db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
		t.Fatal(err)
	}
	other, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	id := identity(f.Reviewer)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, conn := range []*sql.DB{db, other} {
		wg.Add(1)
		go func(conn *sql.DB) {
			defer wg.Done()
			<-start
			s := tr.Service{Transactions: tr.Manager{DB: conn}, Idempotency: tr.Store{}, Audit: audit{}}
			_, e := s.Mutate(context.Background(), id, allow, review(f, id))
			results <- e
		}(conn)
	}
	close(start)
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatal(e)
		}
	}
	if count(t, db, "SELECT count(*) FROM review_decisions WHERE site_log_id=?", f.Log) != 1 {
		t.Fatal("concurrent duplicate")
	}
	// A new pool reconstructs the durable outcome, with no process-local cache.
	if err = (tr.Manager{DB: other}).WithinWrite(context.Background(), func(c context.Context, tx contracts.Transaction) error {
		r, e := (tr.Store{}).Lookup(c, tx, id)
		if e != nil {
			return e
		}
		if r == nil || r.Version != 2 {
			t.Fatal("lost outcome")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPanicCancellationAndMetadata(t *testing.T) {
	db := testsupport.Database(t)
	f := fixtures.Seed(t, db)
	m := tr.Manager{DB: db}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("missing panic")
			}
		}()
		_ = m.WithinWrite(context.Background(), func(ctx context.Context, tx contracts.Transaction) error {
			if _, ok := tx.(interface{ Commit() error }); ok {
				t.Fatal("commit exposed")
			}
			if _, e := tx.ExecContext(ctx, "UPDATE site_logs SET notes='lost' WHERE id=?", f.Log); e != nil {
				t.Fatal(e)
			}
			panic("fault")
		})
	}()
	if count(t, db, "SELECT count(*) FROM site_logs WHERE id=? AND notes=''", f.Log) != 1 {
		t.Fatal("panic persisted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	err := m.WithinWrite(ctx, func(ctx context.Context, tx contracts.Transaction) error {
		_, e := tx.ExecContext(ctx, "UPDATE site_logs SET notes='lost' WHERE id=?", f.Log)
		cancel()
		return e
	})
	if err == nil {
		t.Fatal("cancel committed")
	}
	md, err := tr.NewMetadata(f.Creator, &f.Project, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	next, err := md.Advance(1, time.Now().Add(time.Second))
	if err != nil || next.ID != md.ID || next.CreatedAt != md.CreatedAt || next.CreatedBy != md.CreatedBy || next.Version != 2 {
		t.Fatal("metadata loss", err)
	}
	if err = tr.CheckRetirement("active", "archived", "reason", []tr.Edge{{From: "active", To: "archived"}}); err != nil {
		t.Fatal(err)
	}
	assertCode(t, tr.CheckTransition("approved", "draft", "", nil), "INVALID_TRANSITION")
	recorder := httptest.NewRecorder()
	tr.WriteError(recorder, httptest.NewRequest("POST", "/", nil), tr.Conflict("STALE_VERSION"))
	if recorder.Code != 409 {
		t.Fatal(recorder.Code)
	}
	recorder = httptest.NewRecorder()
	tr.WriteError(recorder, httptest.NewRequest("POST", "/", nil), errors.New("private SQL path"))
	if strings.Contains(recorder.Body.String(), "private") {
		t.Fatal("leaked internals")
	}
}
