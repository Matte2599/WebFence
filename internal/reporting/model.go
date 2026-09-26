// Package reporting exports immutable, redacted M1 scan results as bilingual
// JSON/HTML bundles. It never reconstructs traffic absent from the ledger.
package reporting

import (
	"context"
	"errors"
	"time"

	"github.com/Matte2599/WebFence/internal/intelligence"
	"github.com/Matte2599/WebFence/internal/storage"
)

var (
	ErrInvalid          = errors.New("report_invalid_input")
	ErrUnavailable      = errors.New("report_unavailable")
	ErrInvalidBundle    = errors.New("report_invalid_bundle")
	ErrUnsigned         = errors.New("report_unsigned")
	ErrUntrustedKey     = errors.New("report_untrusted_key")
	ErrRevokedKey       = errors.New("report_revoked_key")
	ErrInvalidSignature = errors.New("report_invalid_signature")
)

// Snapshot is a detached source for one export. The authorization revision is
// the one associated with the run, not the project's possibly newer revision.
type Snapshot struct {
	ProjectID, ProjectName, TargetOwner string
	Origins                             []string
	AuthorizationRevision               uint64
	AuthorizationExpiresAt              time.Time
	Run                                 storage.ScanRun
	Intelligence                        *intelligence.Snapshot
	Assessments                         []intelligence.Assessment
}

func Load(ctx context.Context, store *storage.Store, runID string) (Snapshot, error) {
	if ctx == nil || store == nil || runID == "" {
		return Snapshot{}, ErrInvalid
	}
	run, err := store.LoadScanRun(ctx, runID)
	if err != nil {
		return Snapshot{}, err
	}
	if run.State == "running" {
		return Snapshot{}, ErrInvalid
	}
	project, err := store.LoadProject(ctx, run.ProjectID)
	if err != nil {
		return Snapshot{}, err
	}
	revisions, err := store.ListAuthorizationRevisions(ctx, run.ProjectID)
	if err != nil {
		return Snapshot{}, err
	}
	for _, rev := range revisions {
		if rev.Revision == run.AuthorizationRevision {
			return Snapshot{ProjectID: project.ID(), ProjectName: project.Name(), TargetOwner: rev.TargetOwner,
				Origins: append([]string(nil), rev.Origins...), AuthorizationRevision: rev.Revision,
				AuthorizationExpiresAt: rev.ExpiresAt, Run: run}, nil
		}
	}
	return Snapshot{}, ErrInvalid
}

type Report struct {
	Schema          string                    `json:"schema"`
	ReportID        string                    `json:"report_id"`
	Language        string                    `json:"language"`
	GeneratedAt     string                    `json:"generated_at_utc"`
	SignatureStatus string                    `json:"signature_status"`
	SignerName      string                    `json:"signer_name,omitempty"`
	Project         ReportProject             `json:"project"`
	Run             ReportRun                 `json:"run"`
	Coverage        ReportCoverage            `json:"coverage"`
	Intelligence    ReportIntelligence        `json:"intelligence"`
	Assessments     []intelligence.Assessment `json:"assessments"`
	Visits          []ReportVisit             `json:"visits"`
}

type ReportProject struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	TargetOwner            string   `json:"target_owner"`
	AuthorizedOrigins      []string `json:"authorized_origins"`
	AuthorizationRevision  uint64   `json:"authorization_revision"`
	AuthorizationExpiresAt string   `json:"authorization_expires_at_utc"`
}

type ReportRun struct {
	ID              string `json:"id"`
	Mode            string `json:"mode"`
	State           string `json:"state"`
	StartedAtUTC    string `json:"started_at_utc"`
	FinishedAtUTC   string `json:"finished_at_utc"`
	PlannedSeeds    int    `json:"planned_seeds"`
	CompletedVisits int    `json:"completed_visits"`
	RequestsUsed    int    `json:"requests_used"`
	StopCode        string `json:"stop_code,omitempty"`
}

type ReportCoverage struct {
	Limited               bool   `json:"limited"`
	WholeSiteAttested     bool   `json:"whole_site_attested"`
	UnknownUnexecuted     bool   `json:"unknown_unexecuted"`
	UnexecutedSeedMinimum int    `json:"unexecuted_seed_minimum"`
	UnexecutedReason      string `json:"unexecuted_reason"`
	Observed              int    `json:"observed"`
	NotObserved           int    `json:"not_observed"`
	Inconclusive          int    `json:"inconclusive"`
	Skipped               int    `json:"skipped"`
	Message               string `json:"message"`
}

type ReportIntelligence struct {
	Status      string `json:"status"`
	Source      string `json:"source,omitempty"`
	WindowStart string `json:"window_start_utc,omitempty"`
	Watermark   string `json:"watermark_utc,omitempty"`
	LastSuccess string `json:"last_success_utc,omitempty"`
	Records     int    `json:"records,omitempty"`
}

type ReportVisit struct {
	Index           int    `json:"index"`
	Depth           int    `json:"depth"`
	HTTPStatus      int    `json:"http_status"`
	RuleID          string `json:"rule_id"`
	RuleRevision    int    `json:"rule_revision"`
	Outcome         string `json:"outcome"`
	EvidenceCode    string `json:"evidence_code"`
	DiscoveryStatus string `json:"discovery_status"`
	DiscoveryReason string `json:"discovery_reason"`
	Links           int    `json:"links"`
	Forms           int    `json:"forms"`
	OutOfScope      int    `json:"out_of_scope"`
	Invalid         int    `json:"invalid"`
	Title           string `json:"title"`
	OutcomeText     string `json:"outcome_text"`
	EvidenceText    string `json:"evidence_text"`
}
