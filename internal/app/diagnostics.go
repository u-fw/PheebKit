package app

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"kompeg/internal/buildinfo"
	"kompeg/internal/core"
)

type DiagnosticOptions struct {
	IncludeRecentJobs bool `json:"includeRecentJobs"`
	JobLimit          int  `json:"jobLimit"`
}
type DiagnosticPreview struct {
	Token string `json:"token"`
	JSON  string `json:"json"`
}
type diagnosticSnapshot struct {
	data    []byte
	created time.Time
	order   uint64
}
type diagnosticBinary struct {
	Available bool   `json:"available"`
	Source    string `json:"source"`
	Release   string `json:"release"`
}
type diagnosticEncoder struct {
	Name      string `json:"name"`
	Codec     string `json:"codec"`
	Kind      string `json:"kind"`
	Available bool   `json:"available"`
}
type diagnosticJob struct {
	State          string `json:"state"`
	Operation      string `json:"operation"`
	Codec          string `json:"codec"`
	Failure        string `json:"failure"`
	OutputChecked  bool   `json:"outputChecked"`
	ReviewWarnings int    `json:"reviewWarnings"`
}
type diagnosticReport struct {
	SchemaVersion       int                 `json:"schemaVersion"`
	GeneratedAt         time.Time           `json:"generatedAt"`
	ApplicationVersion  string              `json:"applicationVersion"`
	Revision            string              `json:"revision"`
	ModifiedBuild       bool                `json:"modifiedBuild"`
	WailsVersion        string              `json:"wailsVersion"`
	GoVersion           string              `json:"goVersion"`
	OS                  string              `json:"os"`
	OSVersion           string              `json:"osVersion"`
	Architecture        string              `json:"architecture"`
	LogicalCPUs         int                 `json:"logicalCPUs"`
	FFmpeg              diagnosticBinary    `json:"ffmpeg"`
	FFprobe             diagnosticBinary    `json:"ffprobe"`
	EnvironmentReady    bool                `json:"environmentReady"`
	CapabilitiesScanned bool                `json:"capabilitiesScanned"`
	CapabilitiesStatus  string              `json:"capabilitiesStatus"`
	EditorRecordStatus  string              `json:"editorRecordStatus"`
	WorkspaceReadError  bool                `json:"workspaceReadError"`
	GPUVendors          []string            `json:"gpuVendors"`
	Encoders            []diagnosticEncoder `json:"encoders"`
	QueueStates         map[string]int      `json:"queueStates"`
	HistorySaveError    bool                `json:"historySaveError"`
	RecentJobs          []diagnosticJob     `json:"recentJobs,omitempty"`
	PrivacyPolicy       string              `json:"privacyPolicy"`
}

var ffmpegRelease = regexp.MustCompile(`^(?:ffmpeg|ffprobe) version n?([0-9]{1,3}\.[0-9]{1,3}(?:\.[0-9]{1,3})?)\b`)
var revisionPattern = regexp.MustCompile(`^[a-f0-9]{7,64}$`)
var diagnosticModuleVersion = regexp.MustCompile(`^v[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}(?:-(?:alpha|beta|rc)\.[0-9]{1,4})?$`)

func diagnosticChoice(value string, allowed ...string) string {
	for _, option := range allowed {
		if value == option {
			return value
		}
	}
	return "other"
}
func diagnosticBinaryView(binary BinaryStatus) diagnosticBinary {
	release := "unknown"
	if match := ffmpegRelease.FindStringSubmatch(binary.Version); len(match) == 2 {
		release = match[1]
	}
	return diagnosticBinary{Available: binary.Available, Release: release, Source: diagnosticChoice(binary.Source, "PATH", "config", "KOMPEG_FFMPEG", "KOMPEG_FFPROBE", "FFMPEG_PATH", "FFPROBE_PATH")}
}
func diagnosticFailure(raw string) string {
	if raw == "" {
		return "none"
	}
	text := strings.ToLower(raw)
	for _, category := range []struct {
		name  string
		words []string
	}{
		{"output-invalid", []string{"verify output structure"}},
		{"source-changed", []string{"fingerprint", "source changed", "已变化", "re-import"}},
		{"output-exists", []string{"already exists", "file exists", "已存在"}},
		{"permission-denied", []string{"permission denied", "access is denied", "拒绝访问"}},
		{"disk-full", []string{"no space", "disk full", "空间不足"}},
		{"encoder", []string{"encoder", "nvenc", "qsv", "amf"}},
	} {
		for _, word := range category.words {
			if strings.Contains(text, word) {
				return category.name
			}
		}
	}
	return "other"
}

func (s *Service) GetDiagnosticPreview(options DiagnosticOptions) (DiagnosticPreview, error) {
	if options.JobLimit == 0 {
		options.JobLimit = 20
	}
	if options.JobLimit < 1 || options.JobLimit > 50 {
		return DiagnosticPreview{}, errors.New("诊断任务摘要数量需为 1–50")
	}
	s.ensureWorkspace()
	s.configMu.RLock()
	environment := s.environment
	s.configMu.RUnlock()
	report := diagnosticReport{SchemaVersion: 1, GeneratedAt: time.Now().UTC(), ApplicationVersion: buildinfo.Version, Revision: "unknown", GoVersion: runtime.Version(), OS: runtime.GOOS, OSVersion: diagnosticOSVersion(), Architecture: runtime.GOARCH, LogicalCPUs: runtime.NumCPU(), FFmpeg: diagnosticBinaryView(environment.FFmpeg), FFprobe: diagnosticBinaryView(environment.FFprobe), EnvironmentReady: environment.Ready, GPUVendors: []string{}, Encoders: []diagnosticEncoder{}, QueueStates: map[string]int{}, PrivacyPolicy: "structured-allowlist-v1: no paths, filenames, usernames, argv, raw logs, raw errors, URLs, environment values, media metadata, custom parameters or project documents"}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && revisionPattern.MatchString(setting.Value) {
				report.Revision = setting.Value
			}
			if setting.Key == "vcs.modified" {
				report.ModifiedBuild = setting.Value == "true"
			}
		}
		for _, dependency := range info.Deps {
			if dependency.Path == "github.com/wailsapp/wails/v3" && diagnosticModuleVersion.MatchString(dependency.Version) {
				report.WailsVersion = dependency.Version
			}
		}
	}
	if report.WailsVersion == "" {
		report.WailsVersion = "unknown"
	}
	report.CapabilitiesStatus = "scan-in-progress"
	if s.capsMu.TryLock() {
		report.CapabilitiesStatus = "not-scanned"
		if s.caps != nil {
			report.CapabilitiesStatus = "cached"
			report.CapabilitiesScanned = true
			vendors := map[string]bool{}
			for _, gpu := range s.caps.GPUs {
				vendors[diagnosticChoice(strings.ToLower(gpu.Vendor), "nvidia", "amd", "intel")] = true
			}
			for vendor := range vendors {
				report.GPUVendors = append(report.GPUVendors, vendor)
			}
			sort.Strings(report.GPUVendors)
			for _, encoder := range s.caps.VideoEncoders {
				name := diagnosticChoice(encoder.Name, "libx264", "libx265", "libsvtav1", "libaom-av1", "h264_nvenc", "hevc_nvenc", "av1_nvenc", "h264_qsv", "hevc_qsv", "av1_qsv", "h264_amf", "hevc_amf", "av1_amf")
				if name == "other" || len(report.Encoders) >= 32 {
					continue
				}
				report.Encoders = append(report.Encoders, diagnosticEncoder{Name: name, Codec: diagnosticChoice(encoder.Codec, "h264", "hevc", "av1"), Kind: diagnosticChoice(encoder.Kind, "cpu", "gpu"), Available: encoder.Available})
			}
		}
		s.capsMu.Unlock()
	}
	report.EditorRecordStatus = "busy"
	if s.editorMu.TryLock() {
		_, path, err := readEditor()
		report.EditorRecordStatus = "readable"
		if err != nil {
			report.EditorRecordStatus = "unreadable"
		} else if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			report.EditorRecordStatus = "missing"
		}
		s.editorMu.Unlock()
	}
	s.workspaceMu.Lock()
	report.WorkspaceReadError = s.workspaceError != ""
	s.workspaceMu.Unlock()
	all := s.jobs.List()
	for _, job := range all {
		report.QueueStates[diagnosticChoice(job.State, core.JobQueued, core.JobPreparing, core.JobRunning, core.JobCancelling, core.JobCancelled, core.JobCompleted, core.JobFailed, core.JobInterrupted)]++
	}
	report.HistorySaveError = s.jobs.PersistenceError() != ""
	if options.IncludeRecentJobs {
		sort.SliceStable(all, func(i, j int) bool { return all[i].QueuedAt.After(all[j].QueuedAt) })
		for _, job := range all {
			if len(report.RecentJobs) >= options.JobLimit {
				break
			}
			item := diagnosticJob{State: diagnosticChoice(job.State, core.JobQueued, core.JobPreparing, core.JobRunning, core.JobCancelling, core.JobCancelled, core.JobCompleted, core.JobFailed, core.JobInterrupted), Operation: "unknown", Codec: "unknown", Failure: diagnosticFailure(job.Error)}
			if job.Project != nil {
				operation := job.Project.Settings.Operation
				if operation == "" {
					operation = "compress"
				}
				item.Operation = diagnosticChoice(operation, "compress", "remux", "extract-audio", "extract-subtitle")
				item.Codec = diagnosticChoice(job.Project.Settings.Codec, "h264", "hevc", "av1")
			}
			if job.Result != nil && job.Result.Verification != nil {
				item.OutputChecked = true
				item.ReviewWarnings = len(job.Result.Verification.Warnings)
			}
			report.RecentJobs = append(report.RecentJobs, item)
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return DiagnosticPreview{}, err
	}
	data = append(data, '\n')
	if len(data) > 512*1024 {
		return DiagnosticPreview{}, errors.New("诊断报告超过 512 KiB")
	}
	s.diagnosticsMu.Lock()
	defer s.diagnosticsMu.Unlock()
	if s.diagnostics == nil {
		s.diagnostics = make(map[string]diagnosticSnapshot)
	}
	for token, snapshot := range s.diagnostics {
		if time.Since(snapshot.created) > 10*time.Minute {
			delete(s.diagnostics, token)
		}
	}
	for len(s.diagnostics) >= 4 {
		oldest := ""
		var oldestOrder uint64
		for token, snapshot := range s.diagnostics {
			if oldest == "" || snapshot.order < oldestOrder {
				oldest = token
				oldestOrder = snapshot.order
			}
		}
		delete(s.diagnostics, oldest)
	}
	token := "diagnostic-" + rand.Text()
	s.diagnosticSequence++
	s.diagnostics[token] = diagnosticSnapshot{data: data, created: time.Now(), order: s.diagnosticSequence}
	return DiagnosticPreview{Token: token, JSON: string(data)}, nil
}

func (s *Service) ExportDiagnosticReport(token string) (string, error) {
	if s.app == nil {
		return "", errors.New("native file dialog is unavailable")
	}
	data, err := s.diagnosticData(token)
	if err != nil {
		return "", err
	}
	path, err := s.app.Dialog.SaveFile().SetFilename("PheebKit-diagnostics.json").AddFilter("脱敏诊断报告", "*.json").PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	protected := []string{}
	s.mediaMu.RLock()
	for _, media := range s.media {
		protected = append(protected, media.Source.URI)
	}
	s.mediaMu.RUnlock()
	for _, job := range s.jobs.List() {
		protected = append(protected, job.Input.URI, job.Output.URI)
	}
	for _, name := range []string{"ffmpeg.json", "preferences.json", "history.json", "editor.json"} {
		value, err := workspaceFile(name)
		if err != nil {
			return "", err
		}
		protected = append(protected, value)
	}
	if err := writeDiagnosticAt(path, data, protected); err != nil {
		return "", err
	}
	return path, nil
}
func (s *Service) diagnosticData(token string) ([]byte, error) {
	s.diagnosticsMu.Lock()
	defer s.diagnosticsMu.Unlock()
	snapshot, ok := s.diagnostics[token]
	if !ok || time.Since(snapshot.created) > 10*time.Minute {
		delete(s.diagnostics, token)
		return nil, errors.New("诊断预览已过期，请重新生成")
	}
	return append([]byte(nil), snapshot.data...), nil
}
func writeDiagnosticAt(path string, data []byte, protected []string) error {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".json") || strings.ContainsRune(path, 0) || len(data) > 512*1024 {
		return errors.New("诊断报告需保存为本地 .json 文件，且不超过 512 KiB")
	}
	for _, value := range protected {
		if value != "" && pathsAlias(path, value) {
			return errors.New("诊断报告不能覆盖源媒体、处理结果或应用配置")
		}
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".diagnostics-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
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
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("保存诊断报告：%w", err)
	}
	return nil
}
