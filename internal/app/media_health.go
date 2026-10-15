package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"kompeg/internal/core"
)

func cloneHealth(r core.MediaHealth) core.MediaHealth {
	r.DamagedRanges = append([]core.TimeRange{}, r.DamagedRanges...)
	return r
}

func (s *Service) StartMediaHealthScan(uri string, windowSeconds int) (core.MediaHealth, error) {
	if windowSeconds < 1 || windowSeconds > 30 {
		return core.MediaHealth{}, errors.New("扫描窗口需要在 1–30 秒之间")
	}
	s.lifecycleMu.Lock()
	if s.exiting {
		s.lifecycleMu.Unlock()
		return core.MediaHealth{}, errors.New("应用正在退出")
	}
	s.lifecycleMu.Unlock()
	s.configMu.RLock()
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.exiting {
		s.configMu.RUnlock()
		return core.MediaHealth{}, errors.New("应用正在退出")
	}
	media, ok := s.cachedMedia(uri)
	if !ok {
		s.configMu.RUnlock()
		return core.MediaHealth{}, errors.New("请先导入媒体")
	}
	scanner, ok := s.runner.snapshot().(core.MediaHealthScanner)
	if !ok {
		s.configMu.RUnlock()
		return core.MediaHealth{}, core.ErrUnsupported
	}
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if s.healthBusy.Load() {
		s.configMu.RUnlock()
		return core.MediaHealth{}, errors.New("已有扫描正在进行，请先取消")
	}
	var token [16]byte
	_, err := rand.Read(token[:])
	id := hex.EncodeToString(token[:])
	if err != nil {
		s.configMu.RUnlock()
		return core.MediaHealth{}, err
	}
	ctx, cancel := context.WithTimeout(s.ctx, 12*time.Hour)
	s.healthCancel = cancel
	s.health = core.MediaHealth{ID: id, URI: uri, SourceFingerprint: media.SourceFingerprint, DurationSeconds: media.DurationSeconds, WindowSeconds: float64(windowSeconds), State: "scanning", DamagedRanges: []core.TimeRange{}}
	s.healthBusy.Store(true)
	s.healthWG.Add(1)
	s.configMu.RUnlock()
	go func() {
		defer s.healthWG.Done()
		defer cancel()
		report, err := scanner.ScanHealth(ctx, media, windowSeconds, func(value core.MediaHealth) {
			s.healthMu.Lock()
			defer s.healthMu.Unlock()
			value.ID = id
			if s.health.State == "cancelling" {
				value.State = "cancelling"
			}
			s.health = cloneHealth(value)
		})
		s.healthMu.Lock()
		defer s.healthMu.Unlock()
		report.ID = id
		if err != nil {
			report.State = "failed"
			report.Error = err.Error()
		}
		if errors.Is(ctx.Err(), context.Canceled) || s.health.State == "cancelling" {
			report.State = "cancelled"
			report.Error = ""
		}
		s.health = cloneHealth(report)
		s.healthCancel = nil
		s.healthBusy.Store(false)
	}()
	return cloneHealth(s.health), nil
}

func (s *Service) GetMediaHealthScan() core.MediaHealth {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	return cloneHealth(s.health)
}

func (s *Service) CancelMediaHealthScan(id string) error {
	s.healthMu.Lock()
	defer s.healthMu.Unlock()
	if id != s.health.ID {
		return errors.New("扫描已过期")
	}
	if s.healthCancel != nil {
		s.health.State = "cancelling"
		s.healthCancel()
	}
	return nil
}
