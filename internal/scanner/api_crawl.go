package scanner

import (
	"context"
	"errors"

	"github.com/Matte2599/WebFence/internal/apiimport"
	"github.com/Matte2599/WebFence/internal/storage"
)

var ErrAPISelection = errors.New("scanner_invalid_api_selection")

// APICrawlPlan contains an operator-selected, bounded set of static GET paths.
// The document is untrusted input and never supplies an origin, credentials,
// methods, IP grants, or a request policy. SelectedPaths must exactly match
// candidates in the current offline inventory.
type APICrawlPlan struct {
	Document      []byte
	Origin        string
	SelectedPaths []string
	Crawl         CrawlPlan
}

// RunAPICrawl imports the document without networking, validates every
// explicitly selected route, then delegates all traffic to RunCrawl. The
// latter starts a fresh managed run and rechecks authorization, scope, route
// policy, grants and budget before the first request. No discovered links are
// followed in this API-only workflow.
func RunAPICrawl(ctx context.Context, store *storage.Store, plan APICrawlPlan) (CrawlReport, error) {
	if ctx == nil || store == nil || plan.Crawl.ProjectID == "" || plan.Origin == "" ||
		len(plan.Document) == 0 || len(plan.Document) > apiimport.MaxDocumentBytes ||
		len(plan.SelectedPaths) == 0 || len(plan.SelectedPaths) > MaxHeaderLabSeeds ||
		len(plan.Crawl.SeedURLs) != 0 || plan.Crawl.FollowLinks || plan.Crawl.MaxDepth != 0 ||
		!plan.Crawl.Policy.Valid() {
		return CrawlReport{}, ErrAPISelection
	}
	run, err := store.BeginRun(ctx, plan.Crawl.ProjectID)
	if err != nil {
		return CrawlReport{}, err
	}
	inventory, err := apiimport.Import(plan.Document, plan.Origin, run.Scope(), plan.Crawl.Policy)
	run.Close()
	if err != nil {
		return CrawlReport{}, err
	}
	candidates := make(map[string]string)
	for _, route := range inventory.Routes {
		if route.Method == "GET" && route.Disposition == apiimport.Candidate {
			candidates[route.Path] = route.CandidateURL
		}
	}
	seeds := make([]string, 0, len(plan.SelectedPaths))
	seen := make(map[string]struct{}, len(plan.SelectedPaths))
	for _, path := range plan.SelectedPaths {
		url, ok := candidates[path]
		if !ok {
			return CrawlReport{}, ErrAPISelection
		}
		if _, duplicate := seen[path]; duplicate {
			return CrawlReport{}, ErrAPISelection
		}
		seen[path] = struct{}{}
		seeds = append(seeds, url)
	}
	plan.Crawl.SeedURLs = seeds
	plan.Crawl.MaxPages = len(seeds)
	// A redirect to another allowed path is still a different operation than
	// the one selected by the operator. Stop at the response instead.
	plan.Crawl.Limits.MaxRedirects = 0
	return RunCrawl(ctx, store, plan.Crawl)
}
