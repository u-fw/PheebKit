// Package android defines the native execution seam for the Technical Spike.
package android

import (
	"context"

	"pheebkit/internal/core"
)

type Runner struct{}

var _ core.Runner = (*Runner)(nil)

func New() *Runner { return &Runner{} }

func (*Runner) Probe(context.Context, core.Source) (core.Media, error) {
	return core.Media{}, core.ErrUnsupported
}

func (*Runner) Capabilities(context.Context) (core.Capabilities, error) {
	return core.Capabilities{Platform: "android", Encoders: []core.Encoder{}, Codecs: []core.Codec{}, Hardware: []string{}}, nil
}

func (*Runner) Start(context.Context, core.ExecutionPlan, core.Callbacks) (core.Result, error) {
	return core.Result{}, core.ErrUnsupported
}

func (*Runner) Cancel(string) error { return core.ErrUnsupported }
