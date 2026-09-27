package scanner

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Matte2599/WebFence/internal/browser"
	"github.com/Matte2599/WebFence/internal/storage"
)

var ErrObservedSelection = errors.New("scanner_invalid_observed_selection")

// ObservedCrawlPlan replays only operator-selected, successful GET fetch
// paths. Observations are untrusted hints, never authorization or evidence.
// The caller declares the origin separately and explicitly confirms replay.
type ObservedCrawlPlan struct {
	Origin          string
	Observations    []browser.Observation
	SelectedIndices []int
	ReplayConfirmed bool
	Crawl           CrawlPlan
}

// RunObservedCrawl creates a fresh managed run for selected JavaScript fetch
// paths. It never reuses browser credentials, query strings or response bodies.
// RunCrawl revalidates the current project, route policy, IP grant and budget.
func RunObservedCrawl(ctx context.Context, store *storage.Store, plan ObservedCrawlPlan) (CrawlReport, error) {
	if ctx == nil || store == nil || !plan.ReplayConfirmed || plan.Crawl.ProjectID == "" ||
		len(plan.Observations) == 0 || len(plan.Observations) > browser.MaxObservations ||
		len(plan.SelectedIndices) == 0 || len(plan.SelectedIndices) > MaxHeaderLabSeeds ||
		len(plan.Crawl.SeedURLs) != 0 || plan.Crawl.FollowLinks || plan.Crawl.MaxDepth != 0 ||
		!plan.Crawl.Policy.Valid() {
		return CrawlReport{}, ErrObservedSelection
	}
	origin, err := url.Parse(plan.Origin)
	if err != nil || origin == nil || origin.Scheme == "" || origin.Host == "" || origin.User != nil ||
		origin.Path != "" || origin.RawPath != "" || origin.RawQuery != "" || origin.ForceQuery ||
		origin.Fragment != "" || strings.HasSuffix(plan.Origin, "/") {
		return CrawlReport{}, ErrObservedSelection
	}
	seeds := make([]string, 0, len(plan.SelectedIndices))
	seen := make(map[int]struct{}, len(plan.SelectedIndices))
	paths := make(map[string]struct{}, len(plan.SelectedIndices))
	for _, index := range plan.SelectedIndices {
		if index < 0 || index >= len(plan.Observations) {
			return CrawlReport{}, ErrObservedSelection
		}
		if _, duplicate := seen[index]; duplicate {
			return CrawlReport{}, ErrObservedSelection
		}
		seen[index] = struct{}{}
		item := plan.Observations[index]
		if item.Method != http.MethodGet || item.Kind != browser.Fetch || item.StatusCode < 200 ||
			item.StatusCode >= 300 || item.Path != item.FinalPath || len(item.Path) == 0 ||
			len(item.Path) > 1024 || item.Path[0] != '/' || strings.HasPrefix(item.Path, "//") ||
			strings.ContainsAny(item.Path, "?#\\\r\n") {
			return CrawlReport{}, ErrObservedSelection
		}
		if _, duplicate := paths[item.Path]; duplicate {
			return CrawlReport{}, ErrObservedSelection
		}
		paths[item.Path] = struct{}{}
		seeds = append(seeds, plan.Origin+item.Path)
	}
	plan.Crawl.SeedURLs = seeds
	plan.Crawl.MaxPages = len(seeds)
	plan.Crawl.Limits.MaxRedirects = 0
	return RunCrawl(ctx, store, plan.Crawl)
}
