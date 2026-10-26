package app

import "pheebkit/internal/core"

func (s *Service) publishJobFeedback(job core.Job) {
	if s.jobFeedback == nil {
		return
	}
	update := s.jobFeedback.observe(job, s.Locale())
	if s.app == nil {
		return
	}
	s.app.SetTaskbarProgress(update.taskbar)
	if update.notification != nil {
		s.app.NotifyJob(*update.notification)
	}
}
