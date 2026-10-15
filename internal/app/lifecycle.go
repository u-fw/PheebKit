package app

// CanClose is called by native hooks, including Alt+F4. The backend is authoritative.
//
//wails:ignore
func (s *Service) CanClose() bool {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.exiting {
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
		if s.app != nil {
			s.app.Event.Emit("app:close-requested", CloseRequest{Jobs: count, Downloading: downloading, Unsaved: s.unsavedEdits})
		}
		return false
	}
	s.exiting = true
	return true
}

type CloseRequest struct {
	Jobs        int  `json:"jobs"`
	Downloading bool `json:"downloading"`
	Unsaved     bool `json:"unsaved"`
}

func (s *Service) SetUnsavedEdits(unsaved bool, revision uint64) {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if revision > s.draftRevision {
		s.unsavedEdits = unsaved
		s.draftRevision = revision
	}
}

func (s *Service) ConfirmQuit() {
	s.lifecycleMu.Lock()
	s.exiting = true
	s.lifecycleMu.Unlock()
	s.Close()
	if s.app != nil {
		s.app.Quit()
	}
}
