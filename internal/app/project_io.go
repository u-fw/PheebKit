package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kompeg/internal/compiler"
	"kompeg/internal/core"
	"kompeg/internal/localstore"
	projectfile "kompeg/internal/project"
)

// Portable presets omit settings tied to a particular source. Shared files are
// editable intent; importing one never starts a process or executes saved argv.
func portableSettings(settings core.Settings) core.Settings {
	settings = core.CloneSettings(settings)
	settings.Trim = nil
	if settings.AudioSelection != nil && settings.AudioSelection.Mode == "selected" {
		settings.AudioSelection = nil
	}
	if settings.SubtitleSelection != nil && settings.SubtitleSelection.Mode == "selected" {
		settings.SubtitleSelection = nil
	}
	if settings.Transform != nil {
		settings.Transform.Crop = nil
	}
	return settings
}

func (s *Service) SaveProject(p core.Project) (string, error) {
	if s.app == nil {
		return "", errors.New("native file dialog is unavailable")
	}
	if err := validateProjectShape(p); err != nil {
		return "", err
	}
	path, err := s.app.Dialog.SaveFile().SetFilename("PheebKit.ffproject").AddFilter("PheebKit 项目", "*.ffproject").PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".ffproject") {
		return "", errors.New("请使用 .ffproject 项目扩展名")
	}
	if err := saveProjectAt(path, p); err != nil {
		return "", err
	}
	return path, nil
}

func validateProjectShape(p core.Project) error {
	if p.SchemaVersion != core.ProjectSchemaVersion || len(p.Inputs) != 1 {
		return errors.New("项目需要受支持的版本和一个源文件")
	}
	for _, input := range p.Inputs {
		if input.Kind != core.SourceLocalFile || !filepath.IsAbs(input.URI) || strings.ContainsRune(input.URI, 0) {
			return errors.New("项目只能引用本地绝对文件路径")
		}
	}
	if p.Output.URI != "" && (p.Output.Kind != core.SourceLocalFile || !filepath.IsAbs(p.Output.URI) || strings.ContainsRune(p.Output.URI, 0)) {
		return errors.New("输出必须是本地绝对路径")
	}
	if p.Output.URI != "" && strings.EqualFold(filepath.Clean(p.Inputs[0].URI), filepath.Clean(p.Output.URI)) {
		return errors.New("项目输出不能覆盖源文件")
	}
	return compiler.ValidateSettings(p.Settings)
}

func saveProjectAt(path string, p core.Project) error {
	if err := validateProjectShape(p); err != nil {
		return err
	}
	if pathsAlias(path, p.Inputs[0].URI) || (p.Output.URI != "" && pathsAlias(path, p.Output.URI)) {
		return errors.New("项目文件不能覆盖媒体或处理结果")
	}
	return projectfile.Save(path, core.CloneProject(p))
}

func pathsAlias(a, b string) bool {
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	if aErr == nil && bErr == nil && os.SameFile(aInfo, bInfo) {
		return true
	}
	canonical := func(path string) string {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return filepath.Clean(path)
		}
		if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
			return resolved
		}
		if parent, err := filepath.EvalSymlinks(filepath.Dir(absolute)); err == nil {
			return filepath.Join(parent, filepath.Base(absolute))
		}
		return absolute
	}
	return strings.EqualFold(canonical(a), canonical(b))
}

func (s *Service) LoadProject() (*JobReuse, error) {
	if s.app == nil {
		return nil, errors.New("native file dialog is unavailable")
	}
	path, err := s.app.Dialog.OpenFile().SetTitle("打开处理项目").AddFilter("PheebKit 项目", "*.ffproject").PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, err
	}
	value, err := s.loadProjectAt(path)
	return &value, err
}

func (s *Service) loadProjectAt(path string) (JobReuse, error) {
	p, err := projectfile.Load(path)
	if err != nil {
		return JobReuse{}, err
	}
	if err := validateProjectShape(p); err != nil {
		return JobReuse{}, err
	}
	media, err := s.Probe(p.Inputs[0].URI)
	if err != nil {
		return JobReuse{}, fmt.Errorf("项目源文件无法读取，请恢复原文件位置：%w", err)
	}
	// A saved target may now exist. Require a new choice instead of overwriting.
	if p.Output.URI != "" {
		if _, err := os.Stat(p.Output.URI); err == nil {
			p.Output = core.Source{}
		} else if !os.IsNotExist(err) {
			return JobReuse{}, err
		}
	}
	p.ID = "project-" + time.Now().Format("20060102150405.000000000")
	return JobReuse{Project: p, Media: media}, nil
}

func readPreset(path string) (core.Preset, error) {
	file, err := os.Open(path)
	if err != nil {
		return core.Preset{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1024*1024 {
		return core.Preset{}, errors.New("配置需要是不超过 1 MiB 的普通文件")
	}
	decoder := json.NewDecoder(io.LimitReader(file, 1024*1024+1))
	decoder.DisallowUnknownFields()
	var p core.Preset
	if err := decoder.Decode(&p); err != nil {
		return core.Preset{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return core.Preset{}, errors.New("配置包含多余的数据")
	}
	if p.SchemaVersion != core.ProjectSchemaVersion || strings.TrimSpace(p.Name) == "" || len(p.Name) > 160 {
		return core.Preset{}, errors.New("配置版本或名称无效")
	}
	if err := compiler.ValidateSettings(p.Settings); err != nil {
		return core.Preset{}, err
	}
	p.Settings = portableSettings(p.Settings)
	p.Name = strings.TrimSpace(p.Name)
	p.ID = p.Name
	return p, nil
}

func (s *Service) ImportPreset() (*WorkspacePreferences, error) {
	if s.app == nil {
		return nil, errors.New("native file dialog is unavailable")
	}
	path, err := s.app.Dialog.OpenFile().SetTitle("导入常用配置").AddFilter("PheebKit 配置", "*.pheebpreset").PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, err
	}
	value, err := s.importPresetAt(path)
	return &value, err
}

func (s *Service) importPresetAt(path string) (WorkspacePreferences, error) {
	preset, err := readPreset(path)
	if err != nil {
		return WorkspacePreferences{}, err
	}
	preferencesPath, err := workspaceFile("preferences.json")
	if err != nil {
		return WorkspacePreferences{}, err
	}
	data := WorkspacePreferences{Version: 1, Presets: []core.Preset{}}
	err = localstore.Update(preferencesPath, &data, func() error {
		if data.Version != 1 {
			return errors.New("常用设置版本不受支持")
		}
		if len(data.Presets) >= 20 {
			return errors.New("最多保存 20 个常用配置")
		}
		for _, existing := range data.Presets {
			if existing.Name == preset.Name {
				return errors.New("已存在同名配置；请修改导入文件中的名称，或先删除旧配置")
			}
		}
		data.Presets = append(data.Presets, preset)
		return nil
	})
	return data, err
}

func (s *Service) ExportPreset(id string) (string, error) {
	if s.app == nil {
		return "", errors.New("native file dialog is unavailable")
	}
	status := s.GetWorkspace()
	if status.Error != "" {
		return "", errors.New(status.Error)
	}
	var found *core.Preset
	for _, p := range status.Preferences.Presets {
		if p.ID == id {
			found = &p
			break
		}
	}
	if found == nil {
		return "", errors.New("常用配置不存在")
	}
	path, err := s.app.Dialog.SaveFile().SetFilename("PheebKit.pheebpreset").AddFilter("PheebKit 配置", "*.pheebpreset").PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".pheebpreset") {
		return "", errors.New("请使用 .pheebpreset 配置扩展名")
	}
	if err := exportPresetAt(path, *found); err != nil {
		return "", err
	}
	return path, nil
}

func exportPresetAt(path string, p core.Preset) error {
	p.Settings = portableSettings(p.Settings)
	if p.SchemaVersion != core.ProjectSchemaVersion || strings.TrimSpace(p.Name) == "" || len(p.Name) > 160 {
		return errors.New("配置版本或名称无效")
	}
	if err := compiler.ValidateSettings(p.Settings); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > 1024*1024 {
		return errors.New("配置超过 1 MiB")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".pheebpreset-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if _, err := file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	// Atomic replacement follows the explicit native Save dialog choice.
	return os.Rename(tempPath, path)
}
