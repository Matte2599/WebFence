package reporting

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/storage"
)

func TestLoadUsesHistoricalAuthorizationAndRedactedLedger(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "projects.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	expires := time.Now().Add(24 * time.Hour)
	p, err := project.New(project.Draft{ID: "report-fixture", Name: "Fixture", TargetOwner: "Original owner", AuthorizationReference: "synthetic-ref",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: expires, Origins: []string{"http://127.0.0.1:8765"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	managed, err := store.BeginRun(ctx, p.ID())
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.StartScanRun(managed.Context(), managed.Scope(), "loopback", 1)
	if err != nil {
		t.Fatal(err)
	}
	visit := storage.ScanVisit{VisitIndex: 0, Depth: 0, StatusCode: 200, RuleID: "HTTP-XCTO-001", RuleRevision: 1, Outcome: "observed",
		EvidenceCode: "nosniff_absent", DiscoveryStatus: "observed", DiscoveryReason: "html_observed"}
	if err := store.AppendScanVisit(managed.Context(), id, visit, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishScanRun(ctx, id, "complete", 1, false, ""); err != nil {
		t.Fatal(err)
	}
	managed.Close()
	_, err = store.ReviseAuthorization(ctx, p.ID(), 1, project.AuthorizationDraft{TargetOwner: "New owner", AuthorizationReference: "new-ref",
		AuthorizationConfirmed: true, AuthorizationExpiresAt: expires.Add(24 * time.Hour), Origins: []string{"http://127.0.0.1:9000"}})
	if err != nil {
		t.Fatal(err)
	}
	s, err := Load(ctx, store, id)
	if err != nil {
		t.Fatal(err)
	}
	if s.TargetOwner != "Original owner" || s.AuthorizationRevision != 1 || len(s.Origins) != 1 || s.Origins[0] != "http://127.0.0.1:8765" || len(s.Run.Visits) != 1 {
		t.Fatalf("historical snapshot: %+v", s)
	}
	report, err := buildReport(s, "report-id", "it", time.Now(), false, "", true, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage.Observed != 1 || report.Coverage.WholeSiteAttested || !report.Coverage.UnknownUnexecuted {
		t.Fatalf("coverage: %+v", report.Coverage)
	}
}
