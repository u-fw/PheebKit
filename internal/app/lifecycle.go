package app

import "fmt"

// CanClose is called by native hooks, including Alt+F4. The backend is authoritative.
//
//wails:ignore
func (s *Service) CanClose() bool {
	s.lifecycleMu.Lock()
	if s.exiting {
		finished, accepted := s.shutdownFinished, s.historyLossAccepted
		s.lifecycleMu.Unlock()
		if !finished {
			return false
		}
		if s.jobs.PersistenceError() != "" && !accepted {
			s.requestHistoryRetry()
			return false
		}
		return true
	}
	count := 0
	if s.healthBusy.Load() {
		count++
	}
	for _, job := range s.jobs.List() {
		if activeJob(job.State) {
			count++
		}
	}
	downloading := s.setupBusy.Load()
	if count > 0 || downloading || s.unsavedEdits {
		request := CloseRequest{Jobs: count, Downloading: downloading, Unsaved: s.unsavedEdits}
		s.lifecycleMu.Unlock()
		if s.app != nil {
			s.app.Emit("app:close-requested", request)
		}
		return false
	}
	s.exiting = true
	s.lifecycleMu.Unlock()
	// Native close hooks may run on the window thread. Perform disk I/O and
	// event draining off it; Quit re-enters the hook after a successful flush.
	go func() {
		if err := s.Close(); err != nil {
			s.requestHistoryRetry()
			return
		}
		if s.app != nil {
			s.app.Quit()
		}
	}()
	return false
}

type CloseRequest struct {
	Jobs              int  `json:"jobs"`
	Downloading       bool `json:"downloading"`
	Unsaved           bool `json:"unsaved"`
	ShutdownStarted   bool `json:"shutdownStarted"`
	HistorySaveFailed bool `json:"historySaveFailed"`
}

func (s *Service) SetUnsavedEdits(unsaved bool, revision uint64) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if revision > s.draftRevision {
		s.unsavedEdits = unsaved
		s.draftRevision = revision
	}
}

func (s *Service) ConfirmQuit(allowHistoryLoss bool) error {
	if s.app != nil {
		s.app.Emit("app:close-requested", CloseRequest{ShutdownStarted: true})
	}
	if err := s.Close(); err != nil && !allowHistoryLoss {
		s.requestHistoryRetry()
		return fmt.Errorf("任务历史保存失败，请重试或明确选择不保存历史后退出：%w", err)
	}
	s.lifecycleMu.Lock()
	s.historyLossAccepted = allowHistoryLoss
	s.lifecycleMu.Unlock()
	if s.app != nil {
		s.app.Quit()
	}
	return nil
}

func (s *Service) requestHistoryRetry() {
	if s.app != nil {
		s.app.Emit("app:close-requested", CloseRequest{ShutdownStarted: true, HistorySaveFailed: true})
	}
}
