package app

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"pheebkit/internal/compiler"
	"pheebkit/internal/core"
	"pheebkit/internal/localstore"
)

type historyEntry struct {
	Job core.Job `json:"job"`
	PID int      `json:"pid"`
}
type historyFile struct {
	Version int            `json:"version"`
	Entries []historyEntry `json:"entries" ts_type:"historyEntry[] | null"`
}
type WorkspacePreferences struct {
	Version             int            `json:"version"`
	LastSettings        *core.Settings `json:"lastSettings,omitempty" ts_type:"core.Settings | null"`
	Presets             []core.Preset  `json:"presets" ts_type:"core.Preset[] | null"`
	LastOutputDirectory string         `json:"lastOutputDirectory,omitempty"`
}
type WorkspaceStatus struct {
	Error       string               `json:"error"`
	Preferences WorkspacePreferences `json:"preferences"`
}

func workspaceFile(name string) (string, error) {
	config, err := configFile()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(config), name), nil
}
func loadHistory(path string) ([]core.Job, error) {
	data := historyFile{Version: 1}
	err := localstore.Update(path, &data, nil)
	if err != nil {
		return nil, err
	}
	if data.Version != 1 {
		return nil, errors.New("任务历史版本不受支持")
	}
	jobs := []core.Job{}
	for _, entry := range data.Entries {
		job := entry.Job
		if activeJob(job.State) {
			if localstore.ProcessAlive(entry.PID) {
				continue
			}
			job.State = core.JobInterrupted
			job.Error = "上次处理未正常结束，请检查输出后重新执行。"
			job.CompletedAt = time.Now()
		}
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].QueuedAt.Before(jobs[j].QueuedAt) })
	if len(jobs) > 200 {
		jobs = jobs[len(jobs)-200:]
	}
	return jobs, nil
}
func saveHistory(path string, jobs []core.Job) error {
	data := historyFile{Version: 1}
	return localstore.Update(path, &data, func() error {
		if data.Version != 1 {
			return errors.New("任务历史版本不受支持")
		}
		merged := map[string]historyEntry{}
		for _, entry := range data.Entries {
			if activeJob(entry.Job.State) && !localstore.ProcessAlive(entry.PID) {
				entry.Job.State = core.JobInterrupted
				entry.Job.Error = "上次处理未正常结束，请检查输出后重新执行。"
				entry.Job.CompletedAt = time.Now()
			}
			merged[entry.Job.ID] = entry
		}
		for _, job := range jobs {
			old, found := merged[job.ID]
			if found && old.Job.CompletedAt.After(job.CompletedAt) {
				continue
			}
			merged[job.ID] = historyEntry{Job: job, PID: os.Getpid()}
		}
		data.Entries = nil
		for _, entry := range merged {
			data.Entries = append(data.Entries, entry)
		}
		sort.Slice(data.Entries, func(i, j int) bool { return data.Entries[i].Job.QueuedAt.Before(data.Entries[j].Job.QueuedAt) })
		for len(data.Entries) > 200 {
			removed := false
			for i, entry := range data.Entries {
				if !activeJob(entry.Job.State) {
					data.Entries = append(data.Entries[:i], data.Entries[i+1:]...)
					removed = true
					break
				}
			}
			if !removed {
				break
			}
		}
		return nil
	})
}
func activeJob(state string) bool {
	return state == core.JobQueued || state == core.JobPreparing || state == core.JobRunning || state == core.JobCancelling
}

func (s *Service) GetWorkspace() WorkspaceStatus {
	s.ensureWorkspace()
	status := WorkspaceStatus{Preferences: WorkspacePreferences{Version: 1, Presets: []core.Preset{}}}
	path, err := workspaceFile("preferences.json")
	if err == nil {
		err = localstore.Update(path, &status.Preferences, nil)
	}
	if err == nil && status.Preferences.Version != 1 {
		err = errors.New("常用设置版本不受支持")
	}
	if directory := status.Preferences.LastOutputDirectory; directory != "" {
		if info, statErr := os.Stat(directory); statErr != nil || !info.IsDir() {
			status.Preferences.LastOutputDirectory = ""
		}
	}
	if err != nil {
		status.Error = err.Error()
		status.Preferences = WorkspacePreferences{Version: 1, Presets: []core.Preset{}}
	}
	if historyError := s.jobs.PersistenceError(); historyError != "" {
		status.Error += " 任务历史保存失败：" + historyError
	}
	s.workspaceMu.Lock()
	defer s.workspaceMu.Unlock()
	if s.workspaceError != "" {
		status.Error += " " + s.workspaceError
	}
	return status
}

func (s *Service) DeletePreset(id string) (WorkspacePreferences, error) {
	path, err := workspaceFile("preferences.json")
	if err != nil {
		return WorkspacePreferences{}, err
	}
	data := WorkspacePreferences{Version: 1, Presets: []core.Preset{}}
	err = localstore.Update(path, &data, func() error {
		if data.Version != 1 {
			return errors.New("常用设置版本不受支持")
		}
		for i, preset := range data.Presets {
			if preset.ID == id {
				data.Presets = append(data.Presets[:i], data.Presets[i+1:]...)
				return nil
			}
		}
		return errors.New("常用配置不存在")
	})
	return data, err
}

func (s *Service) ensureWorkspace() {
	s.workspaceOnce.Do(func() {
		path, err := workspaceFile("history.json")
		var history []core.Job
		if err == nil {
			history, err = loadHistory(path)
		}
		if err != nil {
			s.workspaceMu.Lock()
			s.workspaceError = "历史记录读取失败（原文件已保留）：" + err.Error()
			s.workspaceMu.Unlock()
			return
		}
		s.jobs.Restore(history, func(jobs []core.Job) error { return saveHistory(path, jobs) })
	})
}

// Saves encoding preferences only: source/output paths and trim are media-specific.
func (s *Service) SaveSettings(settings core.Settings, name string) (WorkspacePreferences, error) {
	settings = portableSettings(settings)
	if len(name) > 160 || len(settings.RawArguments) > 64 {
		return WorkspacePreferences{}, errors.New("设置名称或高级参数过长")
	}
	if err := compiler.ValidateSettings(settings); err != nil {
		return WorkspacePreferences{}, err
	}
	for _, argument := range settings.RawArguments {
		if len(argument) > 4096 {
			return WorkspacePreferences{}, errors.New("单项高级参数过长")
		}
	}
	path, err := workspaceFile("preferences.json")
	if err != nil {
		return WorkspacePreferences{}, err
	}
	data := WorkspacePreferences{Version: 1, Presets: []core.Preset{}}
	err = localstore.Update(path, &data, func() error {
		if data.Version != 1 {
			return errors.New("常用设置版本不受支持")
		}
		data.LastSettings = &settings
		if name != "" {
			for i := range data.Presets {
				if data.Presets[i].Name == name {
					data.Presets[i].Settings = settings
					return nil
				}
			}
			if len(data.Presets) >= 20 {
				return errors.New("最多保存 20 个常用配置")
			}
			data.Presets = append(data.Presets, core.Preset{SchemaVersion: 1, ID: name, Name: name, Settings: settings})
		}
		return nil
	})
	return data, err
}

type JobReuse struct {
	Project   core.Project `json:"project"`
	Media     core.Media   `json:"media"`
	Notice    string       `json:"notice,omitempty"`
	NeedsSave bool         `json:"needsSave,omitempty"`
}

func (s *Service) ReuseJob(id string) (JobReuse, error) {
	job, found := s.jobs.Get(id)
	if !found || job.Project == nil {
		return JobReuse{}, errors.New("该任务没有可恢复的设置")
	}
	if activeJob(job.State) {
		return JobReuse{}, errors.New("请等待任务结束后再复用设置")
	}
	project := *job.Project
	media, err := s.Probe(job.Input.URI)
	if err != nil {
		return JobReuse{}, err
	}
	project.ID = "reuse-" + time.Now().Format("20060102150405.000000000")
	// A new output is required; the UI suggests a distinct filename.
	project.Output = core.Source{}
	return JobReuse{Project: project, Media: media}, nil
}
