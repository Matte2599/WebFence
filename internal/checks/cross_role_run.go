package checks

import (
	"context"
	"errors"
	"net/netip"
	"unicode/utf8"

	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/session"
	"github.com/Matte2599/WebFence/internal/storage"
	"github.com/Matte2599/WebFence/internal/transport"
)

var ErrCrossRoleRunPlan = errors.New("check_cross_role_run_invalid_plan")

// CrossRoleRunPlan binds both test identities and the declared resource to a
// single loopback-only managed run. Secrets are referenced by ID; neither
// credentials nor response bodies are written to the project store.
type CrossRoleRunPlan struct {
	ProjectID string
	Origin    string
	Grant     transport.Grant
	Policy    scope.RequestPolicy
	Limits    transport.Limits
	Resolver  transport.Resolver
	Routes    transport.SessionRoutes
	Owner     session.Account
	Other     session.Account
	Check     CrossRolePlan
}

// RunCrossRole preflights all declarations before creating the broker. Login
// failures return inconclusive; they never turn into a negative access-control
// result. The run, broker and both ephemeral sessions are closed on every path.
func RunCrossRole(ctx context.Context, store *storage.Store, secrets session.SecretSource,
	plan CrossRoleRunPlan) (CrossRoleResult, error) {
	if ctx == nil || store == nil || secrets == nil || plan.ProjectID == "" || plan.Origin == "" ||
		plan.Grant.Origin != plan.Origin || len(plan.Grant.Addresses) == 0 || len(plan.Grant.Addresses) > 8 ||
		!plan.Policy.Valid() || !plan.Routes.LoginConfirmed ||
		!plan.Check.ResourceConfirmed || !plan.Check.OtherForbiddenConfirmed ||
		len(plan.Check.PrivateBody) == 0 || len(plan.Check.PrivateBody) > 1024 ||
		!utf8.ValidString(plan.Check.PrivateBody) || !session.ValidAccount(plan.Owner) ||
		!session.ValidAccount(plan.Other) || plan.Owner.ID == plan.Other.ID ||
		plan.Owner.Username == plan.Other.Username || plan.Owner.SecretID == plan.Other.SecretID ||
		plan.Owner.ExpectedBody == plan.Other.ExpectedBody {
		return CrossRoleResult{}, ErrCrossRoleRunPlan
	}
	for _, ip := range plan.Grant.Addresses {
		if !ip.IsValid() || !ip.IsLoopback() || ip.Is4In6() {
			return CrossRoleResult{}, ErrCrossRoleRunPlan
		}
	}
	run, err := store.BeginRun(ctx, plan.ProjectID)
	if err != nil {
		return CrossRoleResult{}, err
	}
	defer run.Close()
	permit := run.Scope()
	if _, err := permit.CheckOrigin(plan.Origin); err != nil {
		return CrossRoleResult{}, err
	}
	declaredOrigin, err := scope.New([]string{plan.Origin})
	if err != nil {
		return CrossRoleResult{}, ErrCrossRoleRunPlan
	}
	if _, err := declaredOrigin.Check(plan.Check.ResourceURL); err != nil {
		return CrossRoleResult{}, err
	}
	resource, err := permit.CheckOrigin(plan.Check.ResourceURL)
	if err != nil {
		return CrossRoleResult{}, err
	}
	if err := plan.Policy.Check("GET", resource); err != nil {
		return CrossRoleResult{}, err
	}
	broker, err := transport.NewAuthorizedLabWithSession(run.Context(), permit,
		[]transport.Grant{{Origin: plan.Origin, Addresses: append([]netip.Addr(nil), plan.Grant.Addresses...)}},
		plan.Limits, plan.Resolver, plan.Policy, plan.Routes)
	if err != nil {
		return CrossRoleResult{}, err
	}
	defer broker.Close()
	manager, err := session.NewManager(broker, secrets)
	if err != nil {
		return CrossRoleResult{}, err
	}
	owner, err := manager.Login(run.Context(), plan.Owner)
	if err != nil {
		return loginFailure(run.Context(), "owner_login_unverified"), contextFailure(run.Context())
	}
	defer owner.Close()
	other, err := manager.Login(run.Context(), plan.Other)
	if err != nil {
		return loginFailure(run.Context(), "other_login_unverified"), contextFailure(run.Context())
	}
	defer other.Close()
	checked, err := CheckCrossRole(run.Context(), broker, owner, other, plan.Check)
	if run.Context().Err() != nil {
		return result(Inconclusive, "run_interrupted"), contextFailure(run.Context())
	}
	return checked, err
}

func loginFailure(ctx context.Context, code string) CrossRoleResult {
	if ctx.Err() != nil {
		return result(Inconclusive, "run_interrupted")
	}
	return result(Inconclusive, code)
}

func contextFailure(ctx context.Context) error {
	if ctx.Err() == nil {
		return nil
	}
	return context.Cause(ctx)
}
