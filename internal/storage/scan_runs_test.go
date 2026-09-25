package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-sqlite3"
)

func TestV3MigratesToRedactedLedger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v3.sqlite")
	s := openFixture(t, path)
	p := syntheticProject(t, "v3-fixture", []string{"http://127.0.0.1:8765"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), `DROP TABLE scan_visits; DROP TABLE scan_runs; PRAGMA user_version=3`); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = openFixture(t, path)
	if _, err := s.LoadProject(t.Context(), p.ID()); err != nil {
		t.Fatal(err)
	}
	var version int
	if err := s.db.QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version); err != nil || version != 4 {
		t.Fatalf("migration: version=%d err=%v", version, err)
	}
	if runs, err := s.ListScanRuns(t.Context(), p.ID()); err != nil || len(runs) != 0 {
		t.Fatalf("new ledger: %v %v", runs, err)
	}
}

func TestSQLiteMainFileQuotaRejectsOversizedWrite(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "quota.sqlite"))
	if _, err := s.db.ExecContext(t.Context(), "CREATE TABLE quota_probe(payload BLOB)"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, "INSERT INTO quota_probe(payload) VALUES(zeroblob(70000000))")
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code != sqlite3.ErrFull {
		t.Fatalf("oversized database write: %v", err)
	}
	if !errors.Is(storageError(ctx, err), ErrQuota) {
		t.Fatal("SQLite full was not mapped to quota error")
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM quota_probe").Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed write was retained: %d %v", count, err)
	}
}

func TestScanLedgerSurvivesRestartAndCascadesOnProjectDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.sqlite")
	s := openFixture(t, path)
	p := syntheticProject(t, "fixture-ledger", []string{"http://127.0.0.1:8765"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.StartScanRun(run.Context(), run.Scope(), "loopback", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartScanRun(run.Context(), run.Scope(), "loopback", 1); !errors.Is(err, ErrBusy) {
		t.Fatalf("parallel crawler accepted: %v", err)
	}
	v := ScanVisit{VisitIndex: 0, Depth: 0, StatusCode: 200, RuleID: "HTTP-XCTO-001", RuleRevision: 1, Outcome: "not_observed",
		EvidenceCode: "nosniff_absent", DiscoveryStatus: "observed", DiscoveryReason: "html_observed", Links: 1}
	if err := s.AppendScanVisit(run.Context(), id, v, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Backup(t.Context(), filepath.Join(t.TempDir(), "during-scan.sqlite")); !errors.Is(err, ErrBusy) {
		t.Fatalf("backup during active scan: %v", err)
	}
	if err := s.FinishScanRun(t.Context(), id, "complete", 1, false, ""); err != nil {
		t.Fatal(err)
	}
	run.Close()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = openFixture(t, path)
	loaded, err := s.LoadScanRun(t.Context(), id)
	if err != nil || loaded.State != "complete" || len(loaded.Visits) != 1 || loaded.Visits[0] != v || loaded.CoverageLimited {
		t.Fatalf("loaded ledger: %+v err=%v", loaded, err)
	}
	encoded, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "127.0.0.1") {
		t.Fatal("target URL leaked into result")
	}
	backupPath := filepath.Join(t.TempDir(), "snapshot.sqlite")
	if err := s.Backup(t.Context(), backupPath); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(backupPath)
		if err != nil || info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("backup mode: %v %v", info, err)
		}
	}
	backup := openFixture(t, backupPath)
	if restored, err := backup.LoadScanRun(t.Context(), id); err != nil || len(restored.Visits) != 1 {
		t.Fatalf("backup: %+v %v", restored, err)
	}
	if err := backup.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProject(t.Context(), p.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadScanRun(t.Context(), id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ledger survived project deletion: %v", err)
	}
}

func TestUnfinishedRunRecoversAsInterruptedAndStoreIsExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recovery.sqlite")
	s := openFixture(t, path)
	if _, err := Open(t.Context(), path); !errors.Is(err, ErrBusy) {
		t.Fatalf("second opener: %v", err)
	}
	p := syntheticProject(t, "fixture-crash", []string{"http://127.0.0.1:8765"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.StartScanRun(run.Context(), run.Scope(), "loopback", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = openFixture(t, path)
	loaded, err := s.LoadScanRun(t.Context(), id)
	if err != nil || loaded.State != "interrupted" || !loaded.CoverageLimited || loaded.StopCode != "process_interrupted" || loaded.FinishedAt.IsZero() {
		t.Fatalf("recovered run: %+v %v", loaded, err)
	}
	run.Close()
}

func TestStoreLockCrossProcess(t *testing.T) {
	if probe := os.Getenv("WEBFENCE_LOCK_PROBE"); probe != "" {
		s, err := Open(t.Context(), os.Getenv("WEBFENCE_LOCK_PATH"))
		if probe == "busy" {
			if !errors.Is(err, ErrBusy) {
				t.Fatalf("other process opened locked DB: %v", err)
			}
			return
		}
		if err != nil {
			t.Fatalf("released DB remained locked: %v", err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "cross-process.sqlite")
	s := openFixture(t, path)
	probe := func(expected string) {
		command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestStoreLockCrossProcess$")
		command.Env = append(os.Environ(), "WEBFENCE_LOCK_PROBE="+expected, "WEBFENCE_LOCK_PATH="+path)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s probe: %v\n%s", expected, err, output)
		}
	}
	probe("busy")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	probe("free")
}

func TestLedgerRejectsUnredactedVisitAndImmutableCompletion(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "redaction.sqlite"))
	p := syntheticProject(t, "fixture-redaction", []string{"http://127.0.0.1:8765"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	id, err := s.StartScanRun(run.Context(), run.Scope(), "loopback", 1)
	if err != nil {
		t.Fatal(err)
	}
	v := ScanVisit{VisitIndex: 0, StatusCode: 200, RuleID: "HTTP-XCTO-001", RuleRevision: 1, Outcome: "not_observed", EvidenceCode: "secret-token", DiscoveryStatus: "observed", DiscoveryReason: "html_observed"}
	if err := s.AppendScanVisit(run.Context(), id, v, 1); !errors.Is(err, ErrInvalidRun) {
		t.Fatalf("unredacted value accepted: %v", err)
	}
	v.EvidenceCode = "nosniff_absent"
	if err := s.AppendScanVisit(run.Context(), id, v, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishScanRun(t.Context(), id, "complete", 1, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendScanVisit(run.Context(), id, v, 2); !errors.Is(err, ErrInvalidRun) {
		t.Fatalf("completed run changed: %v", err)
	}
	if err := s.FinishScanRun(t.Context(), id, "complete", 1, false, ""); !errors.Is(err, ErrInvalidRun) {
		t.Fatalf("completed run reopened: %v", err)
	}
}

func TestLedgerRejectsCorruptRunIdentityBeforeDesktopReadsIt(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "tampered-run.sqlite"))
	p := syntheticProject(t, "fixture-tamper", []string{"http://127.0.0.1:8765"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	id, err := s.StartScanRun(run.Context(), run.Scope(), "loopback", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(t.Context(), "UPDATE scan_runs SET id='short' WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListScanRuns(t.Context(), p.ID()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("short run ID reached UI: %v", err)
	}
}

func TestPerProjectRunQuota(t *testing.T) {
	s := openFixture(t, filepath.Join(t.TempDir(), "run-quota.sqlite"))
	p := syntheticProject(t, "quota-fixture", []string{"http://127.0.0.1:8765"})
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	run, err := s.BeginRun(t.Context(), p.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer run.Close()
	for i := 0; i < MaxStoredRunsPerProject; i++ {
		id, err := s.StartScanRun(run.Context(), run.Scope(), "loopback", 1)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		if err := s.FinishScanRun(t.Context(), id, "complete", 0, false, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.StartScanRun(run.Context(), run.Scope(), "loopback", 1); !errors.Is(err, ErrQuota) {
		t.Fatalf("101st run accepted: %v", err)
	}
}
