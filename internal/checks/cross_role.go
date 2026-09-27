// Package checks contains bounded, positive-evidence checks built on managed
// transports. It does not own credentials or perform independent networking.
package checks

import (
	"context"
	"errors"
	"net/http"
	"unicode/utf8"

	"github.com/Matte2599/WebFence/internal/scope"
	"github.com/Matte2599/WebFence/internal/session"
	"github.com/Matte2599/WebFence/internal/transport"
)

const (
	CrossRoleRuleID       = "AUTH-CROSSROLE-001"
	CrossRoleRuleRevision = 1
)

var ErrCrossRolePlan = errors.New("check_cross_role_invalid_plan")

type Outcome string

const (
	Finding      Outcome = "finding"
	Inconclusive Outcome = "inconclusive"
)

type CrossRolePlan struct {
	// ResourceURL and PrivateBody are operator-declared, ephemeral inputs.
	// The exact body must be specific to the owner's private resource. The
	// operator must also confirm that the other identity is not entitled to it.
	ResourceURL             string
	PrivateBody             string
	ResourceConfirmed       bool
	OtherForbiddenConfirmed bool
}

// CrossRoleResult deliberately excludes URLs, response bodies and identities.
// An inconclusive result is never a claim that access control is correct.
type CrossRoleResult struct {
	RuleID       string
	RuleRevision int
	Outcome      Outcome
	EvidenceCode string
}

func result(outcome Outcome, code string) CrossRoleResult {
	return CrossRoleResult{RuleID: CrossRoleRuleID, RuleRevision: CrossRoleRuleRevision,
		Outcome: outcome, EvidenceCode: code}
}

// CheckCrossRole reports a finding only when a declared private resource has
// one exact owner-specific body for both distinct, verified test identities,
// and that body is absent anonymously. All traffic passes through the same
// pinned session broker. The caller owns the sessions and their closure.
func CheckCrossRole(ctx context.Context, broker *transport.Broker, owner, other *session.Session, plan CrossRolePlan) (CrossRoleResult, error) {
	if ctx == nil || broker == nil || owner == nil || other == nil || owner == other ||
		!owner.BoundTo(broker) || !other.BoundTo(broker) || owner.Identity() == "" ||
		owner.Identity() == other.Identity() || owner.ProjectID() == "" ||
		owner.ProjectID() != other.ProjectID() || owner.Revision() != other.Revision() ||
		!plan.ResourceConfirmed || !plan.OtherForbiddenConfirmed ||
		plan.ResourceURL == "" || len(plan.PrivateBody) == 0 ||
		len(plan.PrivateBody) > 1024 || !utf8.ValidString(plan.PrivateBody) {
		return CrossRoleResult{}, ErrCrossRolePlan
	}
	origin, err := scope.New([]string{broker.SessionOrigin()})
	if err != nil {
		return CrossRoleResult{}, ErrCrossRolePlan
	}
	if _, err := origin.Check(plan.ResourceURL); err != nil {
		return CrossRoleResult{}, ErrCrossRolePlan
	}
	if owner.Verify(ctx) != nil || other.Verify(ctx) != nil {
		return result(Inconclusive, "test_session_unverified"), nil
	}
	anonymous, err := broker.Fetch(ctx, plan.ResourceURL)
	if err != nil || anonymous.StatusCode == 0 || anonymous.StatusCode >= 500 {
		return result(Inconclusive, "anonymous_baseline_unavailable"), nil
	}
	if string(anonymous.Body) == plan.PrivateBody {
		return result(Inconclusive, "private_marker_public"), nil
	}
	owned, err := owner.Fetch(ctx, plan.ResourceURL)
	if err != nil || owned.StatusCode != http.StatusOK || string(owned.Body) != plan.PrivateBody {
		return result(Inconclusive, "owner_resource_unverified"), nil
	}
	otherResponse, err := other.Fetch(ctx, plan.ResourceURL)
	if err != nil || otherResponse.StatusCode != http.StatusOK {
		return result(Inconclusive, "other_resource_unverified"), nil
	}
	if string(otherResponse.Body) != plan.PrivateBody {
		return result(Inconclusive, "private_marker_not_reproduced"), nil
	}
	return result(Finding, "cross_role_private_body_reproduced"), nil
}
