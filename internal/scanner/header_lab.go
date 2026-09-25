// Package scanner contains bounded M1 scanning experiments. HeaderLab only
// visits explicit loopback seeds through a managed, authorized transport; its
// HTML discovery observes destinations but does not visit them.
package scanner

import (
	"context"
	"errors"
	"mime"
	"net/netip"
	"strings"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/storage"
	"github.com/Matte2599/WebFence/internal/transport"
)

const MaxHeaderLabSeeds = 32
const MaxHeaderLabGrants = 32
const MaxHeaderLabAddressesPerGrant = 16
const HeaderRuleID = "HTTP-XCTO-001"
const HeaderRuleRevision = 1

var (
	ErrInvalidPlan = errors.New("scanner_invalid_lab_plan")
	ErrFetchFailed = errors.New("scanner_fetch_failed")
)

type Outcome string

const (
	Observed     Outcome = "observed"
	NotObserved  Outcome = "not_observed"
	Inconclusive Outcome = "inconclusive"
	Skipped      Outcome = "skipped"
	CheckError   Outcome = "error"
)

// HeaderLabPlan names GET-only seed URLs. The caller must choose safe fixture
// paths; even GET can have effects. Grants remain loopback-only and cannot
// widen the project's current authorization revision.
type HeaderLabPlan struct {
	ProjectID string
	URLs      []string
	Grants    []transport.Grant
	Limits    transport.Limits
	Resolver  transport.Resolver
}

// Check contains only a stable rule/result code and status, never a target
// URL, response body, cookie, or raw response-header value.
type Check struct {
	SeedIndex    int
	RuleID       string
	RuleRevision int
	StatusCode   int
	Outcome      Outcome
	EvidenceCode string
}

// DiscoverySummary contains counts only; candidate URLs and form actions
// remain ephemeral and are never copied into a Report.
type DiscoverySummary struct {
	SeedIndex  int
	Status     string // observed, skipped or incomplete
	ReasonCode string
	Links      int
	Forms      int
	OutOfScope int
	Invalid    int
}

type Report struct {
	ProjectID             string
	AuthorizationRevision uint64
	PlannedSeeds          int
	CompletedSeeds        int
	RequestsUsed          int
	SeedsComplete         bool // all explicit seeds returned; not site-wide coverage
	StopCode              string
	Checks                []Check
	Discovery             []DiscoverySummary
}

// RunHeaderLab begins a managed Store run, validates every explicit seed
// before any socket, then performs one sequential GET per seed. The broker
// independently rechecks scope, DNS/IP, redirect hops and budgets. On a
// transport failure the partial report is returned with a redacted error.
func RunHeaderLab(ctx context.Context, store *storage.Store, plan HeaderLabPlan) (Report, error) {
	if ctx == nil || store == nil || plan.ProjectID == "" || len(plan.URLs) == 0 ||
		len(plan.URLs) > MaxHeaderLabSeeds || len(plan.Grants) == 0 ||
		len(plan.Grants) > MaxHeaderLabGrants {
		return Report{}, ErrInvalidPlan
	}
	// Own every mutable plan slice before validating it. Otherwise the caller
	// could replace a checked URL while an earlier seed is in flight.
	urls := append([]string(nil), plan.URLs...)
	grants := make([]transport.Grant, len(plan.Grants))
	for i, grant := range plan.Grants {
		if len(grant.Addresses) == 0 || len(grant.Addresses) > MaxHeaderLabAddressesPerGrant {
			return Report{}, ErrInvalidPlan
		}
		grants[i] = transport.Grant{Origin: grant.Origin,
			Addresses: append([]netip.Addr(nil), grant.Addresses...)}
	}
	run, err := store.BeginRun(ctx, plan.ProjectID)
	if err != nil {
		return Report{}, err
	}
	defer run.Close()
	report := Report{ProjectID: run.Scope().ProjectID(), AuthorizationRevision: run.Scope().Revision(),
		PlannedSeeds: len(urls), Checks: make([]Check, 0, len(urls))}
	seen := make(map[string]struct{}, len(urls))
	for _, raw := range urls {
		u, checkErr := run.Scope().CheckOrigin(raw)
		if checkErr != nil {
			return Report{}, checkErr
		}
		key := u.String()
		if _, duplicate := seen[key]; duplicate {
			return Report{}, ErrInvalidPlan
		}
		seen[key] = struct{}{}
	}
	broker, err := transport.NewAuthorizedLab(run.Context(), run.Scope(), grants, plan.Limits, plan.Resolver)
	if err != nil {
		return Report{}, err
	}
	defer broker.Close()
	for index, raw := range urls {
		response, fetchErr := broker.Fetch(run.Context(), raw)
		report.RequestsUsed = broker.RequestsUsed()
		if fetchErr != nil {
			stopped := safeStopError(fetchErr)
			report.StopCode = stopped.Error()
			report.Checks = append(report.Checks, Check{SeedIndex: index, RuleID: HeaderRuleID,
				RuleRevision: HeaderRuleRevision, Outcome: CheckError, EvidenceCode: report.StopCode})
			return report, stopped
		}
		report.CompletedSeeds++
		report.Checks = append(report.Checks, checkXContentTypeOptions(index, response))
		discovery, observeErr := observeResponse(run.Scope(), index, response)
		if observeErr == nil {
			observeErr = run.Scope().Validate()
		}
		if observeErr != nil {
			stopped := safeStopError(observeErr)
			report.StopCode = stopped.Error()
			return report, stopped
		}
		report.Discovery = append(report.Discovery, discovery)
	}
	report.SeedsComplete = true
	return report, nil
}

func observeResponse(permit project.RunScope, index int, response transport.Result) (DiscoverySummary, error) {
	summary, _, err := observeResponseSurface(permit, index, response)
	return summary, err
}

func observeResponseSurface(permit project.RunScope, index int, response transport.Result) (DiscoverySummary, Surface, error) {
	summary := DiscoverySummary{SeedIndex: index, Status: "skipped"}
	if response.StatusCode < 200 || response.StatusCode >= 300 ||
		response.StatusCode == 204 || response.StatusCode == 205 {
		summary.ReasonCode = "http_status_not_applicable"
		return summary, Surface{}, nil
	}
	types := response.Header.Values("Content-Type")
	if len(types) != 1 {
		summary.Status, summary.ReasonCode = "incomplete", "content_type_unknown"
		return summary, Surface{}, nil
	}
	mediaType, params, err := mime.ParseMediaType(types[0])
	if err != nil {
		summary.Status, summary.ReasonCode = "incomplete", "content_type_unknown"
		return summary, Surface{}, nil
	}
	if mediaType != "text/html" {
		summary.ReasonCode = "non_html_response"
		return summary, Surface{}, nil
	}
	if charset := params["charset"]; charset != "" && !strings.EqualFold(charset, "utf-8") {
		summary.Status, summary.ReasonCode = "incomplete", "unsupported_charset"
		return summary, Surface{}, nil
	}
	surface, err := ObserveHTML(permit, response.FinalURL, response.Body)
	if err != nil {
		return DiscoverySummary{}, Surface{}, err
	}
	summary.Links, summary.Forms = len(surface.Links), len(surface.Forms)
	summary.OutOfScope, summary.Invalid = surface.OutOfScope, surface.Invalid
	if surface.Incomplete {
		summary.Status, summary.ReasonCode = "incomplete", "parser_limit_or_encoding"
	} else {
		summary.Status, summary.ReasonCode = "observed", "html_observed"
	}
	return summary, surface, nil
}

func checkXContentTypeOptions(index int, response transport.Result) Check {
	check := Check{SeedIndex: index, RuleID: HeaderRuleID, RuleRevision: HeaderRuleRevision,
		StatusCode: response.StatusCode}
	if response.StatusCode < 200 || response.StatusCode >= 300 ||
		response.StatusCode == 204 || response.StatusCode == 205 {
		check.Outcome, check.EvidenceCode = Skipped, "http_status_not_applicable"
		return check
	}
	types := response.Header.Values("Content-Type")
	if len(types) != 1 {
		check.Outcome, check.EvidenceCode = Inconclusive, "content_type_unknown"
		return check
	}
	contentType, _, err := mime.ParseMediaType(types[0])
	if err != nil {
		check.Outcome, check.EvidenceCode = Inconclusive, "content_type_unknown"
		return check
	}
	if contentType != "text/html" {
		check.Outcome, check.EvidenceCode = Skipped, "non_html_response"
		return check
	}
	values := response.Header.Values("X-Content-Type-Options")
	switch {
	case len(values) == 0:
		check.Outcome, check.EvidenceCode = NotObserved, "nosniff_absent"
	case len(values) == 1 && strings.EqualFold(strings.TrimSpace(values[0]), "nosniff"):
		check.Outcome, check.EvidenceCode = Observed, "nosniff_present"
	default:
		check.Outcome, check.EvidenceCode = Inconclusive, "nosniff_unrecognized"
	}
	return check
}

func safeStopError(err error) error {
	for _, known := range []error{
		context.Canceled, context.DeadlineExceeded,
		project.ErrAuthorizationExpired, project.ErrAuthorizationRevoked,
		scope.ErrInvalidURL, scope.ErrOutOfScope,
		scope.ErrRequestPolicy, scope.ErrMethodDenied, scope.ErrPathDenied, scope.ErrAmbiguousPath,
		transport.ErrAddress, transport.ErrResolve, transport.ErrBudget,
		transport.ErrNetwork, transport.ErrRedirect, transport.ErrRedirectLimit,
		transport.ErrBodyLimit, transport.ErrEncoding,
		storage.ErrUnavailable, storage.ErrQuota,
	} {
		if errors.Is(err, known) {
			return known
		}
	}
	return ErrFetchFailed
}
