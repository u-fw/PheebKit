package app

import (
	"context"
	"pheebkit/internal/core"
	"sync"
)

// runnerProxy keeps the queue's runner identity stable while idle configuration
// changes replace the platform adapter. Queued/running jobs block replacement.
type runnerProxy struct {
	mu      sync.RWMutex
	current core.Runner
}

func (r *runnerProxy) snapshot() core.Runner    { r.mu.RLock(); defer r.mu.RUnlock(); return r.current }
func (r *runnerProxy) replace(next core.Runner) { r.mu.Lock(); defer r.mu.Unlock(); r.current = next }
func (r *runnerProxy) Probe(ctx context.Context, source core.Source) (core.Media, error) {
	return r.snapshot().Probe(ctx, source)
}
func (r *runnerProxy) Capabilities(ctx context.Context) (core.Capabilities, error) {
	return r.snapshot().Capabilities(ctx)
}
func (r *runnerProxy) Start(ctx context.Context, plan core.ExecutionPlan, callbacks core.Callbacks) (core.Result, error) {
	return r.snapshot().Start(ctx, plan, callbacks)
}
func (r *runnerProxy) Cancel(id string) error { return r.snapshot().Cancel(id) }
