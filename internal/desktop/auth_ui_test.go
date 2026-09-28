package desktop

import (
	"errors"
	"testing"
	"time"

	"github.com/Matte2599/WebFence/internal/checks"
	"github.com/Matte2599/WebFence/internal/scanner"
	"github.com/Matte2599/WebFence/internal/transport"
)

func TestAuthPublicModeRequiresConfirmationAndTighterLimits(t *testing.T) {
	origin := "https://public.fixture.test:443"
	crawl := scanner.CrawlPlan{Mode: scanner.CrawlPinnedPublic,
		Grants: []transport.Grant{{Origin: origin}},
		Limits: transport.Limits{MaxRequests: 128, MaxConcurrent: 1, MaxRedirects: 3,
			MaxBodyBytes: 1 << 20, RunTimeout: 5 * time.Minute, MinRequestInterval: 250 * time.Millisecond}}
	for _, change := range []func(*scanner.CrawlPlan){
		func(p *scanner.CrawlPlan) { p.Limits.MaxRequests = 129 },
		func(p *scanner.CrawlPlan) { p.Limits.RunTimeout = 16 * time.Minute },
		func(p *scanner.CrawlPlan) { p.Grants[0].Origin = "http://public.fixture.test:80" },
		func(p *scanner.CrawlPlan) { p.Mode = "unknown" },
	} {
		invalid := crawl
		invalid.Grants = append([]transport.Grant(nil), crawl.Grants...)
		change(&invalid)
		if _, _, err := authModeAndLimits(invalid, true, invalid.Grants[0].Origin); !errors.Is(err, transport.ErrConfig) {
			t.Fatalf("unsafe public mode accepted: %v", err)
		}
	}
	if _, _, err := authModeAndLimits(crawl, false, origin); !errors.Is(err, transport.ErrConfig) {
		t.Fatalf("missing confirmation accepted: %v", err)
	}
	if _, _, err := authModeAndLimits(crawl, true, "https://other.fixture.test:443"); !errors.Is(err, transport.ErrConfig) {
		t.Fatalf("changed visible origin accepted: %v", err)
	}
	mode, limits, err := authModeAndLimits(crawl, true, origin)
	if err != nil || mode != checks.CrossRoleRunPinnedPublic || limits.MaxRequests != 128 ||
		limits.MinRequestInterval != 500*time.Millisecond || limits.MaxBodyBytes != 64<<10 || limits.MaxRedirects != 0 {
		t.Fatalf("public auth limits: mode=%s limits=%+v err=%v", mode, limits, err)
	}
	crawl.Mode = scanner.CrawlLoopback
	mode, limits, err = authModeAndLimits(crawl, false, origin)
	if err != nil || mode != checks.CrossRoleRunLoopback || limits.MinRequestInterval != 250*time.Millisecond {
		t.Fatalf("loopback settings changed: mode=%s limits=%+v err=%v", mode, limits, err)
	}
}
