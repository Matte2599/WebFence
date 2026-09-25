package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/mattn/go-sqlite3"
)

const MaxStoredRunsPerProject = 100
const MaxStoredVisits = 256

// ScanVisit contains only stable rule/discovery codes and counts. Target URLs,
// response headers, bodies, credentials and raw error strings are forbidden.
type ScanVisit struct {
	VisitIndex, Depth, StatusCode     int
	RuleID                            string
	RuleRevision                      int
	Outcome, EvidenceCode             string
	DiscoveryStatus, DiscoveryReason  string
	Links, Forms, OutOfScope, Invalid int
}

type ScanRun struct {
	ID, ProjectID                               string
	AuthorizationRevision                       uint64
	Mode, State                                 string
	StartedAt                                   time.Time
	FinishedAt                                  time.Time
	PlannedSeeds, CompletedVisits, RequestsUsed int
	CoverageLimited                             bool
	StopCode                                    string
	Visits                                      []ScanVisit
}

func allowedCode(value string, values ...string) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}
	return false
}

func validVisit(v ScanVisit) bool {
	return v.VisitIndex >= 0 && v.VisitIndex < MaxStoredVisits && v.Depth >= 0 && v.Depth <= 5 &&
		v.StatusCode >= 100 && v.StatusCode <= 599 &&
		v.RuleID == "HTTP-XCTO-001" && v.RuleRevision == 1 &&
		allowedCode(v.Outcome, "observed", "not_observed", "inconclusive", "skipped") &&
		allowedCode(v.EvidenceCode, "nosniff_absent", "nosniff_present", "nosniff_unrecognized", "content_type_unknown", "non_html_response", "http_status_not_applicable") &&
		allowedCode(v.DiscoveryStatus, "observed", "skipped", "incomplete") &&
		allowedCode(v.DiscoveryReason, "html_observed", "content_type_unknown", "non_html_response", "http_status_not_applicable", "unsupported_charset", "parser_limit_or_encoding") &&
		v.Links >= 0 && v.Links <= 512 && v.Forms >= 0 && v.Forms <= 512 &&
		v.OutOfScope >= 0 && v.OutOfScope <= 512 && v.Invalid >= 0 && v.Invalid <= 512
}

func validStopCode(code string) bool {
	return allowedCode(code, "", "process_interrupted", "context canceled", "context deadline exceeded",
		"project_authorization_expired", "project_authorization_revoked", "scope_invalid_url", "scope_out_of_scope",
		"scope_invalid_request_policy", "scope_method_not_allowed", "scope_path_not_allowed", "scope_ambiguous_path",
		"transport_address_not_allowed", "transport_resolution_failed", "transport_request_budget_exhausted",
		"transport_network_failed", "transport_invalid_redirect", "transport_redirect_limit", "transport_body_limit",
		"transport_encoding_not_supported", "scanner_fetch_failed", "storage_unavailable", "storage_quota_exceeded")
}

// StartScanRun records a managed authorization snapshot before the first
// request. The run quota prevents unbounded accumulation of metadata.
func (s *Store) StartScanRun(ctx context.Context, permit project.RunScope, mode string, plannedSeeds int) (string, error) {
	if s == nil || s.db == nil || ctx == nil || !allowedCode(mode, "loopback", "pinned_public") || plannedSeeds < 1 || plannedSeeds > 32 {
		return "", ErrInvalidRun
	}
	if err := permit.Validate(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", ErrUnavailable
	}
	if s.activeScanID != "" {
		return "", ErrBusy
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", storageError(ctx, err)
	}
	defer tx.Rollback()
	var revision int64
	var revoked int
	err = tx.QueryRowContext(ctx, `SELECT p.current_revision, a.revoked FROM projects p
		JOIN authorization_revisions a ON a.project_id=p.id AND a.revision=p.current_revision
		WHERE p.id=?`, permit.ProjectID()).Scan(&revision, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", storageError(ctx, err)
	}
	if uint64(revision) != permit.Revision() || revoked != 0 {
		return "", project.ErrAuthorizationRevoked
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM scan_runs WHERE project_id=?`, permit.ProjectID()).Scan(&count); err != nil {
		return "", storageError(ctx, err)
	}
	if count >= MaxStoredRunsPerProject {
		return "", ErrQuota
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", ErrUnavailable
	}
	id := hex.EncodeToString(bytes)
	_, err = tx.ExecContext(ctx, `INSERT INTO scan_runs(id, project_id, authorization_revision, mode, state, started_at, planned_seeds)
		VALUES(?, ?, ?, ?, 'running', ?, ?)`, id, permit.ProjectID(), revision, mode, time.Now().UTC().Format(time.RFC3339Nano), plannedSeeds)
	if err != nil {
		return "", storageError(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return "", storageError(ctx, err)
	}
	s.activeScanID = id
	return id, nil
}

func (s *Store) AppendScanVisit(ctx context.Context, runID string, v ScanVisit, requestsUsed int) error {
	if s == nil || s.db == nil || ctx == nil || !validVisit(v) || requestsUsed < 0 || requestsUsed > 10000 {
		return ErrInvalidRun
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storageError(ctx, err)
	}
	defer tx.Rollback()
	var count int
	err = tx.QueryRowContext(ctx, `SELECT completed_visits FROM scan_runs WHERE id=? AND state='running'`, runID).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidRun
	}
	if err != nil {
		return storageError(ctx, err)
	}
	if count >= MaxStoredVisits {
		return ErrQuota
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO scan_visits(run_id, visit_index, depth, status_code, rule_id, rule_revision, outcome, evidence_code,
		discovery_status, discovery_reason, links, forms, out_of_scope, invalid) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID, v.VisitIndex, v.Depth, v.StatusCode, v.RuleID, v.RuleRevision, v.Outcome, v.EvidenceCode,
		v.DiscoveryStatus, v.DiscoveryReason, v.Links, v.Forms, v.OutOfScope, v.Invalid)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrInvalidRun
		}
		return storageError(ctx, err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE scan_runs SET completed_visits=completed_visits+1, requests_used=? WHERE id=?`, requestsUsed, runID)
	if err != nil {
		return storageError(ctx, err)
	}
	if err := tx.Commit(); err != nil {
		return storageError(ctx, err)
	}
	return nil
}

func (s *Store) FinishScanRun(ctx context.Context, runID, state string, requestsUsed int, coverageLimited bool, stopCode string) error {
	if s == nil || s.db == nil || ctx == nil || !allowedCode(state, "complete", "interrupted", "error") ||
		!validStopCode(stopCode) || requestsUsed < 0 || requestsUsed > 10000 || state == "complete" && stopCode != "" {
		return ErrInvalidRun
	}
	if state != "complete" {
		coverageLimited = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrUnavailable
	}
	if s.activeScanID != runID {
		return ErrInvalidRun
	}
	result, err := s.db.ExecContext(ctx, `UPDATE scan_runs SET state=?, finished_at=?, requests_used=?, coverage_limited=?, stop_code=?
		WHERE id=? AND state='running'`, state, time.Now().UTC().Format(time.RFC3339Nano), requestsUsed, coverageLimited, stopCode, runID)
	if err != nil {
		return storageError(ctx, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return storageError(ctx, err)
	}
	if count != 1 {
		s.activeScanID = ""
		return ErrInvalidRun
	}
	s.activeScanID = ""
	return nil
}

func (s *Store) ListScanRuns(ctx context.Context, projectID string) ([]ScanRun, error) {
	if s == nil || s.db == nil || ctx == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, authorization_revision, mode, state, started_at,
		finished_at, planned_seeds, completed_visits, requests_used, coverage_limited, stop_code
		FROM scan_runs WHERE project_id=? ORDER BY started_at DESC`, projectID)
	if err != nil {
		return nil, storageError(ctx, err)
	}
	defer rows.Close()
	var runs []ScanRun
	for rows.Next() {
		var run ScanRun
		var revision int64
		var started string
		var finished sql.NullString
		var limited int
		if err := rows.Scan(&run.ID, &run.ProjectID, &revision, &run.Mode, &run.State, &started,
			&finished, &run.PlannedSeeds, &run.CompletedVisits, &run.RequestsUsed, &limited, &run.StopCode); err != nil {
			return nil, storageError(ctx, err)
		}
		decodedID, idErr := hex.DecodeString(run.ID)
		if len(run.ID) != 32 || idErr != nil || len(decodedID) != 16 || hex.EncodeToString(decodedID) != run.ID ||
			revision < 1 || run.PlannedSeeds < 1 || run.PlannedSeeds > 32 || run.CompletedVisits < 0 ||
			run.CompletedVisits > MaxStoredVisits || run.RequestsUsed < 0 || run.RequestsUsed > 10000 {
			return nil, ErrCorrupt
		}
		run.AuthorizationRevision = uint64(revision)
		run.StartedAt, err = time.Parse(time.RFC3339Nano, started)
		if err != nil {
			return nil, ErrCorrupt
		}
		if finished.Valid {
			run.FinishedAt, err = time.Parse(time.RFC3339Nano, finished.String)
			if err != nil {
				return nil, ErrCorrupt
			}
		}
		run.CoverageLimited = limited == 1
		if !allowedCode(run.Mode, "loopback", "pinned_public") || !allowedCode(run.State, "running", "complete", "interrupted", "error") || !validStopCode(run.StopCode) || limited < 0 || limited > 1 ||
			run.State == "running" && finished.Valid || run.State != "running" && !finished.Valid ||
			run.State == "complete" && run.StopCode != "" || run.State != "complete" && !run.CoverageLimited {
			return nil, ErrCorrupt
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, storageError(ctx, err)
	}
	return runs, nil
}

func (s *Store) LoadScanRun(ctx context.Context, runID string) (ScanRun, error) {
	if s == nil || s.db == nil || ctx == nil {
		return ScanRun{}, ErrUnavailable
	}
	var projectID string
	if err := s.db.QueryRowContext(ctx, `SELECT project_id FROM scan_runs WHERE id=?`, runID).Scan(&projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ScanRun{}, ErrNotFound
		}
		return ScanRun{}, storageError(ctx, err)
	}
	runs, err := s.ListScanRuns(ctx, projectID)
	if err != nil {
		return ScanRun{}, err
	}
	var result ScanRun
	found := false
	for _, run := range runs {
		if run.ID == runID {
			result, found = run, true
			break
		}
	}
	if !found {
		return ScanRun{}, ErrNotFound
	}
	rows, err := s.db.QueryContext(ctx, `SELECT visit_index, depth, status_code, rule_id, rule_revision, outcome, evidence_code,
		discovery_status, discovery_reason, links, forms, out_of_scope, invalid FROM scan_visits
		WHERE run_id=? ORDER BY visit_index`, runID)
	if err != nil {
		return ScanRun{}, storageError(ctx, err)
	}
	defer rows.Close()
	for rows.Next() {
		var v ScanVisit
		if err := rows.Scan(&v.VisitIndex, &v.Depth, &v.StatusCode, &v.RuleID, &v.RuleRevision, &v.Outcome, &v.EvidenceCode,
			&v.DiscoveryStatus, &v.DiscoveryReason, &v.Links, &v.Forms, &v.OutOfScope, &v.Invalid); err != nil {
			return ScanRun{}, storageError(ctx, err)
		}
		if !validVisit(v) {
			return ScanRun{}, ErrCorrupt
		}
		result.Visits = append(result.Visits, v)
	}
	if err := rows.Err(); err != nil {
		return ScanRun{}, storageError(ctx, err)
	}
	if len(result.Visits) != result.CompletedVisits {
		return ScanRun{}, ErrCorrupt
	}
	return result, nil
}
