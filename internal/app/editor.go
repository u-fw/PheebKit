package app

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"kompeg/internal/core"
	"kompeg/internal/localstore"
)

type EditorPreferences struct {
	Autosave        bool `json:"autosave"`
	IntervalSeconds int  `json:"intervalSeconds"`
	DraftLimit      int  `json:"draftLimit"`
	RetentionDays   int  `json:"retentionDays"`
	RecentLimit     int  `json:"recentLimit"`
	UndoLimit       int  `json:"undoLimit"`
}
type RecoveryDraft struct {
	ID        string       `json:"id"`
	PID       int          `json:"pid"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Project   core.Project `json:"project"`
}
type RecentProject struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type editorFile struct {
	Version     int               `json:"version"`
	Preferences EditorPreferences `json:"preferences"`
	Drafts      []RecoveryDraft   `json:"drafts"`
	Recent      []RecentProject   `json:"recent"`
}
type DraftSummary struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updatedAt"`
	Active    bool      `json:"active"`
}
type ProjectLibrary struct {
	Preferences EditorPreferences `json:"preferences"`
	Drafts      []DraftSummary    `json:"drafts"`
	Recent      []RecentProject   `json:"recent"`
	Error       string            `json:"error"`
}

func defaultEditorPreferences() EditorPreferences {
	return EditorPreferences{Autosave: true, IntervalSeconds: 15, DraftLimit: 5, RetentionDays: 14, RecentLimit: 10, UndoLimit: 50}
}
func validateEditorPreferences(p EditorPreferences) error {
	if p.IntervalSeconds < 5 || p.IntervalSeconds > 300 || p.DraftLimit < 1 || p.DraftLimit > 20 || p.RetentionDays < 1 || p.RetentionDays > 90 || p.RecentLimit < 0 || p.RecentLimit > 50 || p.UndoLimit < 10 || p.UndoLimit > 200 {
		return errors.New("草稿间隔需为 5–300 秒，草稿数 1–20，保留天数 1–90，最近项目数 0–50，撤销步数 10–200")
	}
	return nil
}
func readEditor() (editorFile, string, error) {
	data := editorFile{Version: 1, Preferences: defaultEditorPreferences()}
	path, err := workspaceFile("editor.json")
	if err == nil {
		err = localstore.Update(path, &data, nil)
	}
	if err == nil {
		err = validateEditorFile(data)
	}
	return data, path, err
}
func validateEditorFile(data editorFile) error {
	if data.Version != 1 || len(data.Drafts) > 32 || len(data.Recent) > 50 {
		return errors.New("编辑记录版本或数量无效，原文件已保留")
	}
	return validateEditorPreferences(data.Preferences)
}
func pruneEditor(data *editorFile, current string, now time.Time) {
	sort.SliceStable(data.Drafts, func(i, j int) bool { return data.Drafts[i].UpdatedAt.After(data.Drafts[j].UpdatedAt) })
	kept := make([]RecoveryDraft, 0, len(data.Drafts))
	inactive := 0
	for _, draft := range data.Drafts {
		if draft.ID == current || localstore.ProcessAlive(draft.PID) {
			kept = append(kept, draft)
		} else if inactive < data.Preferences.DraftLimit && now.Sub(draft.UpdatedAt) <= time.Duration(data.Preferences.RetentionDays)*24*time.Hour {
			kept = append(kept, draft)
			inactive++
		}
	}
	data.Drafts = kept
	sort.SliceStable(data.Recent, func(i, j int) bool { return data.Recent[i].UpdatedAt.After(data.Recent[j].UpdatedAt) })
	if len(data.Recent) > data.Preferences.RecentLimit {
		data.Recent = data.Recent[:data.Preferences.RecentLimit]
	}
}

func (s *Service) GetProjectLibrary() ProjectLibrary {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	data, _, err := readEditor()
	value := ProjectLibrary{Preferences: data.Preferences, Drafts: []DraftSummary{}, Recent: []RecentProject{}, Error: s.editorError}
	if err != nil {
		value.Preferences = defaultEditorPreferences()
		value.Error = "编辑记录读取失败（原文件已保留）：" + err.Error()
		return value
	}
	pruneEditor(&data, s.editorSession, time.Now())
	value.Recent = data.Recent
	for _, draft := range data.Drafts {
		if draft.ID == s.editorSession {
			continue
		}
		input := ""
		if len(draft.Project.Inputs) == 1 {
			input = draft.Project.Inputs[0].URI
		}
		value.Drafts = append(value.Drafts, DraftSummary{ID: draft.ID, Source: input, UpdatedAt: draft.UpdatedAt, Active: localstore.ProcessAlive(draft.PID)})
	}
	return value
}

func (s *Service) SaveEditorPreferences(p EditorPreferences) error {
	if err := validateEditorPreferences(p); err != nil {
		return err
	}
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	data, path, err := readEditor()
	if err != nil {
		return err
	}
	err = localstore.Update(path, &data, func() error {
		if err := validateEditorFile(data); err != nil {
			return err
		}
		data.Preferences = p
		pruneEditor(&data, s.editorSession, time.Now())
		return nil
	})
	if err == nil {
		s.editorError = ""
	}
	return err
}

func (s *Service) SaveRecoveryDraft(p core.Project, revision uint64) error {
	if err := validateProjectShape(p); err != nil {
		return err
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if len(encoded) > 1024*1024 {
		return errors.New("自动草稿超过 1 MiB；请精简设置后重试")
	}
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	if revision <= s.recoveryRevision {
		return nil
	}
	data, path, err := readEditor()
	if err != nil {
		return err
	}
	if s.editorSession == "" {
		s.editorSession = "draft-" + rand.Text()
	}
	err = localstore.Update(path, &data, func() error {
		if err := validateEditorFile(data); err != nil {
			return err
		}
		if !data.Preferences.Autosave {
			return errors.New("自动草稿已关闭")
		}
		found := false
		for i := range data.Drafts {
			if data.Drafts[i].ID == s.editorSession {
				data.Drafts[i].Project = core.CloneProject(p)
				data.Drafts[i].UpdatedAt = time.Now()
				found = true
			}
		}
		if !found {
			data.Drafts = append(data.Drafts, RecoveryDraft{ID: s.editorSession, PID: os.Getpid(), UpdatedAt: time.Now(), Project: core.CloneProject(p)})
		}
		pruneEditor(&data, s.editorSession, time.Now())
		if len(data.Drafts) > 32 {
			return errors.New("其他窗口的活动草稿过多，请关闭不用的窗口")
		}
		return nil
	})
	if err == nil {
		s.recoveryRevision = revision
	}
	return err
}

func (s *Service) ClearRecoveryDraft(revision uint64) error {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	if revision <= s.recoveryRevision {
		return nil
	}
	data, path, err := readEditor()
	if err != nil {
		return err
	}
	err = localstore.Update(path, &data, func() error {
		if err := validateEditorFile(data); err != nil {
			return err
		}
		for i, draft := range data.Drafts {
			if draft.ID == s.editorSession {
				data.Drafts = append(data.Drafts[:i], data.Drafts[i+1:]...)
				break
			}
		}
		pruneEditor(&data, s.editorSession, time.Now())
		return nil
	})
	if err == nil {
		s.recoveryRevision = revision
	}
	return err
}

func (s *Service) RemoveRecoveryDraft(id string) error {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	data, path, err := readEditor()
	if err != nil {
		return err
	}
	return localstore.Update(path, &data, func() error {
		if err := validateEditorFile(data); err != nil {
			return err
		}
		for i, draft := range data.Drafts {
			if draft.ID == id {
				if localstore.ProcessAlive(draft.PID) {
					return errors.New("不能删除活动窗口的草稿")
				}
				data.Drafts = append(data.Drafts[:i], data.Drafts[i+1:]...)
				return nil
			}
		}
		return errors.New("草稿已不存在，请刷新列表")
	})
}

func (s *Service) RestoreRecoveryDraft(id string) (*JobReuse, error) {
	s.editorMu.Lock()
	data, _, err := readEditor()
	s.editorMu.Unlock()
	if err != nil {
		return nil, err
	}
	for _, draft := range data.Drafts {
		if draft.ID != id {
			continue
		}
		if localstore.ProcessAlive(draft.PID) {
			return nil, errors.New("该草稿仍由活动窗口编辑")
		}
		value, err := s.openStoredProject(core.CloneProject(draft.Project))
		if value != nil {
			value.NeedsSave = true
			value.Notice = "已恢复自动草稿，请检查设置并保存项目；原草稿仍保留，任务不会自动开始。"
		}
		return value, err
	}
	return nil, errors.New("草稿已不存在，请刷新列表")
}

func (s *Service) rememberProject(path string) {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	data, storage, err := readEditor()
	if err == nil {
		err = localstore.Update(storage, &data, func() error {
			if err := validateEditorFile(data); err != nil {
				return err
			}
			if data.Preferences.RecentLimit == 0 {
				data.Recent = nil
				return nil
			}
			for i := range data.Recent {
				if pathsAlias(data.Recent[i].Path, path) {
					data.Recent[i].Path = path
					data.Recent[i].UpdatedAt = time.Now()
					pruneEditor(&data, s.editorSession, time.Now())
					return nil
				}
			}
			data.Recent = append(data.Recent, RecentProject{ID: "recent-" + rand.Text(), Path: path, UpdatedAt: time.Now()})
			pruneEditor(&data, s.editorSession, time.Now())
			return nil
		})
	}
	s.editorError = ""
	if err != nil {
		s.editorError = "项目已打开/保存，但最近项目记录失败：" + err.Error()
	}
}

func (s *Service) OpenRecentProject(id string) (*JobReuse, error) {
	s.editorMu.Lock()
	data, _, err := readEditor()
	s.editorMu.Unlock()
	if err != nil {
		return nil, err
	}
	for _, recent := range data.Recent {
		if recent.ID == id {
			return s.openProjectFile(recent.Path)
		}
	}
	return nil, fmt.Errorf("最近项目已不存在，请刷新列表")
}

func (s *Service) ClearRecentProjects() error {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	data, path, err := readEditor()
	if err != nil {
		return err
	}
	return localstore.Update(path, &data, func() error {
		if err := validateEditorFile(data); err != nil {
			return err
		}
		data.Recent = nil
		return nil
	})
}
