package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"kompeg/internal/core"
	"kompeg/internal/jobs"
)

type QueueStatus jobs.QueueStatus

func (s *Service) QueueStatus() QueueStatus { return QueueStatus(s.jobs.QueueStatus()) }

func (s *Service) emitQueueStatus() {
	if s.app != nil {
		s.app.Event.Emit("queue:updated", s.QueueStatus())
	}
}

func (s *Service) SetQueuePaused(paused bool) (QueueStatus, error) {
	status, err := s.jobs.SetQueuePaused(paused)
	if err == nil {
		s.emitQueueStatus()
	}
	return QueueStatus(status), err
}

func (s *Service) MoveQueued(id string, direction int) (QueueStatus, error) {
	status, err := s.jobs.MoveQueued(id, direction)
	if err == nil {
		s.emitQueueStatus()
	}
	return QueueStatus(status), err
}

// StartBatch compiles every project before the manager reserves the whole batch.
// A validation or capacity failure never queues only a subset of the selection.
func (s *Service) StartBatch(projects []core.Project) ([]core.Job, error) {
	if len(projects) == 0 || len(projects) > 32 {
		return nil, errors.New("批量任务需要 1–32 个文件")
	}
	s.ensureWorkspace()
	s.lifecycleMu.Lock()
	exiting := s.exiting
	s.lifecycleMu.Unlock()
	if exiting {
		return nil, errors.New("应用正在退出，无法创建任务")
	}
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	plans := make([]core.ExecutionPlan, len(projects))
	for i, project := range projects {
		plan, err := s.preview(project)
		if err != nil {
			return nil, fmt.Errorf("第 %d 个文件：%w", i+1, err)
		}
		plans[i] = plan
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.exiting {
		return nil, errors.New("应用正在退出，无法创建任务")
	}
	queued, err := s.jobs.EnqueueBatch(plans, projects)
	if err == nil {
		s.emitQueueStatus()
	}
	return queued, err
}

func (s *Service) ChooseMediaPaths() ([]string, error) {
	if s.app == nil {
		return nil, errors.New("native file dialog is unavailable")
	}
	return s.app.Dialog.OpenFile().SetTitle("选择媒体文件（可多选）").AddFilter("视频与音频", "*.mp4;*.mkv;*.mov;*.webm;*.avi;*.m4v;*.ts;*.mp3;*.flac;*.wav;*.m4a;*.ogg;*.opus;*.aac").PromptForMultipleSelection()
}

func (s *Service) ChooseOutputDirectory(suggested string) (string, error) {
	if s.app == nil {
		return "", errors.New("native directory dialog is unavailable")
	}
	return s.app.Dialog.OpenFile().SetTitle("选择批量输出文件夹").CanChooseFiles(false).CanChooseDirectories(true).CanCreateDirectories(true).SetDirectory(filepath.Clean(suggested)).PromptForSingleSelection()
}
