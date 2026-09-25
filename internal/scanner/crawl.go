package scanner

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/Matte2599/WebFence/internal/project"
	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/storage"
	"github.com/Matte2599/WebFence/internal/transport"
)

const MaxCrawlPages = 256
const MaxCrawlDepth = 5

type CrawlMode string

const (
	CrawlLoopback     CrawlMode = "loopback"
	CrawlPinnedPublic CrawlMode = "pinned_public"
)

// CrawlPlan is an opt-in, GET-only visit plan. Every seed and every discovered
// candidate must pass both the managed origin scope and the route policy.
// Public mode additionally requires exact public IP pins per origin.
type CrawlPlan struct {
	ProjectID   string
	SeedURLs    []string
	Mode        CrawlMode
	Grants      []transport.Grant
	Policy      scope.RequestPolicy
	Limits      transport.Limits
	Resolver    transport.Resolver
	MaxPages    int
	MaxDepth    int
	FollowLinks bool
	// OnProgress receives only a completed-visit count; callers must marshal
	// any UI updates onto the GUI thread themselves.
	OnProgress func(completed int)
}

// CrawlVisit has no URL, query, response header or body. VisitIndex refers to
// BFS order, not to a seed index. The discovery summary contains counts only.
type CrawlVisit struct {
	VisitIndex   int
	Depth        int
	StatusCode   int
	RuleID       string
	RuleRevision int
	Outcome      Outcome
	EvidenceCode string
	Discovery    DiscoverySummary
}

type CrawlReport struct {
	RunID                 string
	ProjectID             string
	AuthorizationRevision uint64
	PlannedSeeds          int
	CompletedVisits       int
	RequestsUsed          int
	QueueDrained          bool // queued visits processed; never whole-site coverage
	CoverageLimited       bool
	StopCode              string
	PolicyOmitted         int
	DepthOmitted          int
	PageLimitOmitted      int
	DuplicateOmitted      int
	NotScheduled          int // links observed when FollowLinks is false
	Visits                []CrawlVisit
}

type crawlItem struct {
	url   string // ephemeral, may contain sensitive query data
	depth int
}

// RunCrawl uses only a Store-managed run and the pinned broker. It preflights
// all explicit seeds before networking, then visits a bounded BFS queue in
// order. Forms are observed but never scheduled or submitted. No browser,
// authentication or retry exists.
func RunCrawl(ctx context.Context, store *storage.Store, plan CrawlPlan) (report CrawlReport, runErr error) {
	if ctx == nil || store == nil || plan.ProjectID == "" || len(plan.SeedURLs) == 0 ||
		len(plan.SeedURLs) > MaxHeaderLabSeeds || len(plan.Grants) == 0 || len(plan.Grants) > MaxHeaderLabGrants ||
		!plan.Policy.Valid() || plan.MaxPages < len(plan.SeedURLs) || plan.MaxPages > MaxCrawlPages ||
		plan.MaxDepth < 0 || plan.MaxDepth > MaxCrawlDepth ||
		(plan.FollowLinks && plan.MaxDepth == 0) || (!plan.FollowLinks && plan.MaxDepth != 0) ||
		(plan.Mode != CrawlLoopback && plan.Mode != CrawlPinnedPublic) {
		return CrawlReport{}, ErrInvalidPlan
	}
	seeds := append([]string(nil), plan.SeedURLs...)
	grants := make([]transport.Grant, len(plan.Grants))
	for i, grant := range plan.Grants {
		if len(grant.Addresses) == 0 || len(grant.Addresses) > MaxHeaderLabAddressesPerGrant {
			return CrawlReport{}, ErrInvalidPlan
		}
		grants[i] = transport.Grant{Origin: grant.Origin, Addresses: append([]netip.Addr(nil), grant.Addresses...)}
	}
	run, err := store.BeginRun(ctx, plan.ProjectID)
	if err != nil {
		return CrawlReport{}, err
	}
	defer run.Close()
	permit := run.Scope()
	report = CrawlReport{ProjectID: permit.ProjectID(), AuthorizationRevision: permit.Revision(),
		PlannedSeeds: len(seeds), Visits: make([]CrawlVisit, 0, plan.MaxPages)}
	queue := make([]crawlItem, 0, plan.MaxPages)
	queued := make(map[string]struct{}, plan.MaxPages)
	for _, raw := range seeds {
		u, err := permit.CheckOrigin(raw)
		if err != nil {
			return CrawlReport{}, err
		}
		if err := plan.Policy.Check("GET", u); err != nil {
			return CrawlReport{}, err
		}
		key := u.String()
		if _, duplicate := queued[key]; duplicate {
			return CrawlReport{}, ErrInvalidPlan
		}
		queued[key] = struct{}{}
		queue = append(queue, crawlItem{url: key})
	}
	var broker *transport.Broker
	switch plan.Mode {
	case CrawlLoopback:
		broker, err = transport.NewAuthorizedLabWithPolicy(run.Context(), permit, grants, plan.Limits, plan.Resolver, plan.Policy)
	case CrawlPinnedPublic:
		broker, err = transport.NewAuthorizedPublic(run.Context(), permit, grants, plan.Limits, plan.Resolver, plan.Policy)
	}
	if err != nil {
		return CrawlReport{}, err
	}
	defer broker.Close()
	runID, err := store.StartScanRun(run.Context(), permit, string(plan.Mode), len(seeds))
	if err != nil {
		return CrawlReport{}, err
	}
	report.RunID = runID
	completedNormally := false
	defer func() {
		state := "complete"
		if !completedNormally && runErr == nil {
			state = "interrupted"
			report.StopCode = "process_interrupted"
			report.CoverageLimited = true
		} else if runErr != nil {
			state = "error"
			if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) ||
				errors.Is(runErr, project.ErrAuthorizationExpired) || errors.Is(runErr, project.ErrAuthorizationRevoked) {
				state = "interrupted"
			}
			if report.StopCode == "" {
				report.StopCode = safeStopError(runErr).Error()
			}
			report.CoverageLimited = true
		}
		finishCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if finishErr := store.FinishScanRun(finishCtx, runID, state, report.RequestsUsed, report.CoverageLimited, report.StopCode); finishErr != nil && runErr == nil {
			runErr = finishErr
			report.CoverageLimited = true
		}
	}()
	visited := make(map[string]struct{}, plan.MaxPages)
	for head := 0; head < len(queue); head++ {
		item := queue[head]
		if _, duplicate := visited[item.url]; duplicate {
			report.DuplicateOmitted++
			continue
		}
		visited[item.url] = struct{}{}
		response, fetchErr := broker.Fetch(run.Context(), item.url)
		report.RequestsUsed = broker.RequestsUsed()
		if fetchErr != nil {
			stopped := safeStopError(fetchErr)
			report.StopCode = stopped.Error()
			report.CoverageLimited = true
			return report, stopped
		}
		visited[response.FinalURL] = struct{}{}
		check := checkXContentTypeOptions(head, response)
		discovery, surface, observeErr := observeResponseSurface(permit, head, response)
		if observeErr == nil {
			observeErr = permit.Validate()
		}
		if observeErr != nil {
			stopped := safeStopError(observeErr)
			report.StopCode = stopped.Error()
			report.CoverageLimited = true
			return report, stopped
		}
		visit := CrawlVisit{VisitIndex: head, Depth: item.depth,
			StatusCode: check.StatusCode, RuleID: check.RuleID, RuleRevision: check.RuleRevision,
			Outcome: check.Outcome, EvidenceCode: check.EvidenceCode, Discovery: discovery}
		if err := store.AppendScanVisit(run.Context(), runID, storage.ScanVisit{
			VisitIndex: visit.VisitIndex, Depth: visit.Depth, StatusCode: visit.StatusCode,
			RuleID: visit.RuleID, RuleRevision: visit.RuleRevision,
			Outcome: string(visit.Outcome), EvidenceCode: visit.EvidenceCode,
			DiscoveryStatus: visit.Discovery.Status, DiscoveryReason: visit.Discovery.ReasonCode,
			Links: visit.Discovery.Links, Forms: visit.Discovery.Forms,
			OutOfScope: visit.Discovery.OutOfScope, Invalid: visit.Discovery.Invalid,
		}, broker.RequestsUsed()); err != nil {
			stopped := safeStopError(err)
			report.StopCode, report.CoverageLimited = stopped.Error(), true
			return report, stopped
		}
		report.CompletedVisits++
		report.Visits = append(report.Visits, visit)
		if plan.OnProgress != nil {
			plan.OnProgress(report.CompletedVisits)
		}
		if discovery.Status == "incomplete" || discovery.OutOfScope != 0 || discovery.Invalid != 0 {
			report.CoverageLimited = true
		}
		if !plan.FollowLinks {
			report.NotScheduled += len(surface.Links)
			continue
		}
		for _, raw := range surface.Links {
			u, checkErr := permit.CheckOrigin(raw)
			if checkErr != nil {
				if errors.Is(checkErr, scope.ErrOutOfScope) || errors.Is(checkErr, scope.ErrInvalidURL) {
					report.PolicyOmitted++
					report.CoverageLimited = true
					continue
				}
				stopped := safeStopError(checkErr)
				report.StopCode = stopped.Error()
				report.CoverageLimited = true
				return report, stopped
			}
			if err := plan.Policy.Check("GET", u); err != nil {
				report.PolicyOmitted++
				report.CoverageLimited = true
				continue
			}
			key := u.String()
			if _, duplicate := visited[key]; duplicate {
				report.DuplicateOmitted++
				continue
			}
			if _, duplicate := queued[key]; duplicate {
				report.DuplicateOmitted++
				continue
			}
			if item.depth >= plan.MaxDepth {
				report.DepthOmitted++
				report.CoverageLimited = true
				continue
			}
			if len(queue) >= plan.MaxPages {
				report.PageLimitOmitted++
				report.CoverageLimited = true
				continue
			}
			queued[key] = struct{}{}
			queue = append(queue, crawlItem{url: key, depth: item.depth + 1})
		}
	}
	report.QueueDrained = true
	if report.NotScheduled > 0 {
		report.CoverageLimited = true
	}
	completedNormally = true
	return report, nil
}
