//go:build !windows

package taskbar

type noopController struct{}

func New() Controller { return noopController{} }

func (noopController) Set(Snapshot) {}

func (noopController) Close() {}
