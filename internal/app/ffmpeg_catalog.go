package app

import (
	"context"
	"errors"
	"time"

	"kompeg/internal/core"
)

func (s *Service) GetFFmpegCatalog(category string) ([]core.CatalogEntry, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	inspector, ok := s.runner.snapshot().(core.FFmpegInspector)
	if !ok {
		return nil, core.ErrUnsupported
	}
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()
	return inspector.Catalog(ctx, category)
}
func (s *Service) GetFFmpegOptionHelp(category, name string) (string, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	inspector, ok := s.runner.snapshot().(core.FFmpegInspector)
	if !ok {
		return "", core.ErrUnsupported
	}
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()
	return inspector.OptionHelp(ctx, category, name)
}
func (s *Service) ChooseSubtitleFile() (string, error) {
	if s.app == nil {
		return "", errors.New("native file dialog is unavailable")
	}
	return s.app.OpenFile(FileDialog{FilterName: "文字字幕 SRT / ASS / SSA / WebVTT", Pattern: "*.srt;*.ass;*.ssa;*.vtt"})
}
