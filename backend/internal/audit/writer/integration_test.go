package writer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"buildestate/backend/internal/audit/writer"
	"buildestate/backend/internal/contracts"
	tr "buildestate/backend/internal/platform/transactions"
	"buildestate/backend/internal/testsupport"
	fixtures "buildestate/backend/internal/testsupport/transactions"
)

func TestRealWriterReviewCommitAndTriggerRollback(t *testing.T) {
	for _, fault := range []bool{false, true} {
		name := "commit"
		if fault {
			name = "trigger rollback"
		}
		t.Run(name, func(t *testing.T) {
			db := testsupport.Database(t)
			f := fixtures.Seed(t, db)
			if fault {
				if _, err := db.Exec(`CREATE TRIGGER audit_test_fault AFTER INSERT ON audit_events BEGIN SELECT RAISE(ABORT,'internal /private/database failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			digest, err := tr.Digest(struct{ Version int64 }{1})
			if err != nil {
				t.Fatal(err)
			}
			reason := "Please correct the reported quantities before resubmission."
			id := contracts.MutationIdentity{ActorID: f.Reviewer, RequestID: tr.NewID(), Operation: "site-log.return", IdempotencyKey: tr.NewID(), RequestDigest: digest}
			service := tr.Service{Transactions: tr.Manager{DB: db}, Idempotency: tr.Store{}, Audit: writer.New()}
			_, err = service.Mutate(context.Background(), id,
				func(context.Context, contracts.Transaction) error { return nil },
				func(ctx context.Context, tx contracts.Transaction) (contracts.CommittedResult, contracts.AuditEvent, error) {
					return tr.Decide(ctx, tx, tr.Review{Kind: "site-log", ID: f.Log, ActorID: f.Reviewer, RequestID: id.RequestID, Decision: "return", Reason: reason, Version: 1, At: time.Now()},
						func(ctx context.Context, tx contracts.Transaction, project, from, to string) error {
							_, err := tx.ExecContext(ctx, `UPDATE site_logs SET status=?,version=version+1,updated_at=? WHERE id=? AND project_id=? AND status=? AND version=1`, to, tr.Timestamp(time.Now()), f.Log, project, from)
							return err
						})
				})
			if fault && !errors.Is(err, writer.ErrAppend) || !fault && err != nil {
				t.Fatalf("mutation error: %v", err)
			}
			wantCount, wantVersion, wantStatus := 1, 2, "returned"
			if fault {
				wantCount, wantVersion, wantStatus = 0, 1, "submitted"
			}
			var version int
			var status string
			if err := db.QueryRow(`SELECT version,status FROM site_logs WHERE id=?`, f.Log).Scan(&version, &status); err != nil {
				t.Fatal(err)
			}
			if version != wantVersion || status != wantStatus {
				t.Fatalf("business state: %d %s", version, status)
			}
			for _, check := range [][2]string{{`SELECT count(*) FROM review_decisions WHERE site_log_id=?`, f.Log}, {`SELECT count(*) FROM audit_events WHERE record_id=?`, f.Log}, {`SELECT count(*) FROM idempotency_records WHERE key=?`, id.IdempotencyKey}} {
				var n int
				if err := db.QueryRow(check[0], check[1]).Scan(&n); err != nil || n != wantCount {
					t.Fatalf("atomic evidence count %d, want %d: %v", n, wantCount, err)
				}
			}
			if !fault {
				var saved string
				if err := db.QueryRow(`SELECT reason FROM audit_events WHERE record_id=?`, f.Log).Scan(&saved); err != nil || saved != reason {
					t.Fatalf("review prose not retained: %q %v", saved, err)
				}
			}
		})
	}
}
