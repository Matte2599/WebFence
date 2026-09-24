package storage

import (
	"context"
	"sync"

	"github.com/Matte2599/WebFence/internal/project"
)

// ManagedRun binds a current, immutable scope to cancellation by this Store.
// Scope carries the lifecycle for authorized brokers; Context also cancels
// other run work. Call Close when the run ends. A bare Project.BeginRun is not
// registered for revocation.
type ManagedRun struct {
	store        *Store
	scope        project.RunScope
	ctx          context.Context
	cancel       context.CancelCauseFunc
	stopDeadline context.CancelFunc
	once         sync.Once
}

func (r *ManagedRun) Scope() project.RunScope {
	if r == nil {
		return project.RunScope{}
	}
	return r.scope
}

func (r *ManagedRun) Context() context.Context {
	if r == nil {
		return nil
	}
	return r.ctx
}

// Close releases the registration and cancels any work using Context.
func (r *ManagedRun) Close() {
	if r == nil || r.store == nil {
		return
	}
	r.once.Do(func() {
		r.cancel(context.Canceled)
		r.stopDeadline()
		r.store.mu.Lock()
		defer r.store.mu.Unlock()
		if runs := r.store.runs[r.scope.ProjectID()]; runs != nil {
			delete(runs, r)
			if len(runs) == 0 {
				delete(r.store.runs, r.scope.ProjectID())
			}
		}
	})
}

// BeginRun loads the current revision and registers it with the Store before
// returning. Revision, deletion and Store.Close cancel every registered run
// for that project. This registry covers one Store instance in one process.
func (s *Store) BeginRun(ctx context.Context, id string) (*ManagedRun, error) {
	if s == nil || s.db == nil || ctx == nil {
		return nil, ErrUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrUnavailable
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, err := s.LoadProject(ctx, id)
	if err != nil {
		return nil, err
	}
	scope, err := p.BeginRun()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	baseCtx, cancel := context.WithCancelCause(ctx)
	runCtx, stopDeadline := context.WithDeadlineCause(baseCtx, scope.ExpiresAt(), project.ErrAuthorizationExpired)
	scope, err = scope.BindLifecycle(runCtx)
	if err != nil {
		cancel(err)
		stopDeadline()
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		cancel(err)
		stopDeadline()
		return nil, err
	}
	r := &ManagedRun{store: s, scope: scope, ctx: runCtx, cancel: cancel, stopDeadline: stopDeadline}
	if s.runs[id] == nil {
		s.runs[id] = make(map[*ManagedRun]struct{})
	}
	s.runs[id][r] = struct{}{}
	context.AfterFunc(runCtx, r.Close)
	return r, nil
}

// revokeRuns is called only while holding s.mu. Cancellation is deliberately
// issued before a revision or delete transaction commits, so commit failure
// can only stop work unnecessarily, never continue work under a changed grant.
func (s *Store) revokeRuns(id string) {
	for run := range s.runs[id] {
		run.cancel(project.ErrAuthorizationRevoked)
	}
	delete(s.runs, id)
}
