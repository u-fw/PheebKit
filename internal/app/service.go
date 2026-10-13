package app

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"kompeg/internal/compiler"
	"kompeg/internal/core"
	"kompeg/internal/jobs"
	"kompeg/internal/sourcefile"
)

// Service is the Wails boundary; the compiler and job engine know nothing of UI.
type Service struct {
	app                *application.App
	runner             *runnerProxy
	jobs               *jobs.Manager
	ctx                context.Context
	cancel             context.CancelFunc
	mediaMu            sync.RWMutex
	media              map[string]core.Media
	mediaUsed          map[string]uint64
	mediaPinned        map[string]bool
	mediaSequence      uint64
	mediaRevision      uint64
	capsMu             sync.Mutex
	caps               *core.Capabilities
	configMu           sync.RWMutex
	environment        Environment
	configuration      binaryConfig
	configurationError string
	setupMu            sync.Mutex
	setupStatus        FFmpegSetupStatus
	setupCancel        context.CancelFunc
	setupBusy          atomic.Bool
	workspaceOnce      sync.Once
	workspaceMu        sync.Mutex
	workspaceError     string
	lifecycleMu        sync.Mutex
	exiting            bool
	unsavedEdits       bool
	draftRevision      uint64
	previewMu          sync.Mutex
	previewTokens      map[string]core.Media
}

func NewService() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	config, err := readBinaryConfig()
	environment := discoverEnvironment(config)
	s := &Service{ctx: ctx, cancel: cancel, runner: &runnerProxy{current: newPlatformRunner(environment)}, media: make(map[string]core.Media), environment: environment, configuration: config}
	if err != nil {
		s.configurationError = "无法读取 FFmpeg 配置：" + err.Error()
		log.Print(s.configurationError)
	}
	s.jobs = jobs.New(s.runner, func(job core.Job) {
		if s.app != nil {
			s.app.Event.Emit("job:updated", job)
			s.emitQueueStatus()
		}
	})
	return s
}

//wails:ignore
func (s *Service) Attach(a *application.App) { s.app = a }

//wails:ignore
func (s *Service) Close() { s.cancel(); s.jobs.Close() }

func (s *Service) GetPlatform() string {
	log.Printf("bridge GetPlatform: %s", runtime.GOOS)
	if s.app != nil {
		s.app.Event.Emit("bridge:ready", runtime.GOOS)
	}
	return runtime.GOOS
}

func source(uri string) core.Source {
	kind := core.SourceLocalFile
	if strings.HasPrefix(uri, "content://") {
		kind = core.SourceContentURI
	}
	return core.Source{URI: uri, DisplayName: filepath.Base(uri), Kind: kind}
}

func (s *Service) Probe(uri string) (core.Media, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.probe(uri)
}

func (s *Service) probe(uri string) (core.Media, error) {
	input := source(uri)
	fingerprint := ""
	if input.Kind == core.SourceLocalFile {
		var err error
		fingerprint, err = sourcefile.Fingerprint(uri)
		if err != nil {
			return core.Media{}, err
		}
	}
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	m, err := s.runner.Probe(ctx, input)
	if err != nil {
		return core.Media{}, err
	}
	if err := sourcefile.Verify(uri, fingerprint); err != nil {
		return core.Media{}, err
	}
	m.SourceFingerprint = fingerprint
	s.cacheMedia(m)
	if s.app != nil {
		s.app.Event.Emit("media:probed", m)
	}
	return m, nil
}

func (s *Service) ChooseMedia() (*core.Media, error) {
	if s.app == nil {
		return nil, errors.New("native file dialog is unavailable")
	}
	path, err := s.app.Dialog.OpenFile().SetTitle("选择媒体文件").AddFilter("视频与音频", "*.mp4;*.mkv;*.mov;*.webm;*.avi;*.m4v;*.ts;*.mp3;*.flac;*.wav;*.m4a;*.ogg").PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, err
	}
	m, err := s.Probe(path)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Service) ChooseOutput(suggested string) (string, error) {
	if s.app == nil {
		return "", errors.New("native file dialog is unavailable")
	}
	return s.app.Dialog.SaveFile().SetDirectory(filepath.Dir(suggested)).SetFilename(filepath.Base(suggested)).AddFilter("媒体输出", "*.mp4;*.mkv;*.webm;*.m4a;*.mka;*.mp3;*.flac;*.wav;*.opus;*.srt;*.vtt;*.ass").PromptForSingleSelection()
}

func (s *Service) Capabilities() (core.Capabilities, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.scanCapabilities(false)
}

// RefreshCapabilities reruns real encoder/device qualification after a rescan.
func (s *Service) RefreshCapabilities() (core.Capabilities, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if err := s.reloadEnvironment(); err != nil {
		return core.Capabilities{}, err
	}
	return s.scanCapabilities(true)
}

func (s *Service) scanCapabilities(refresh bool) (core.Capabilities, error) {
	s.capsMu.Lock()
	defer s.capsMu.Unlock()
	if refresh {
		s.caps = nil
	}
	if s.caps != nil {
		return *s.caps, nil
	}
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	caps, err := s.runner.Capabilities(ctx)
	if err != nil {
		return core.Capabilities{}, err
	}
	s.caps = &caps
	return caps, nil
}

func (s *Service) Preview(p core.Project) (core.ExecutionPlan, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.preview(p)
}

func (s *Service) preview(p core.Project) (core.ExecutionPlan, error) {
	if len(p.Inputs) != 1 {
		return core.ExecutionPlan{}, errors.New("the spike requires exactly one input")
	}
	m, found := s.cachedMedia(p.Inputs[0].URI)
	if !found {
		var err error
		m, err = s.probe(p.Inputs[0].URI)
		if err != nil {
			return core.ExecutionPlan{}, err
		}
	}
	if err := sourcefile.Verify(m.Source.URI, m.SourceFingerprint); err != nil {
		return core.ExecutionPlan{}, err
	}
	caps, err := s.scanCapabilities(false)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	if err := sourcefile.Verify(m.Source.URI, m.SourceFingerprint); err != nil {
		return core.ExecutionPlan{}, err
	}
	plan, err := compiler.Compile(p, m, caps)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	plan.Preview = compiler.Preview(previewBinary(s.runner.snapshot()), plan.Args)
	return plan, nil
}

func (s *Service) Start(p core.Project) (core.Job, error) {
	s.ensureWorkspace()
	s.lifecycleMu.Lock()
	if s.exiting {
		s.lifecycleMu.Unlock()
		return core.Job{}, errors.New("应用正在退出，无法创建任务")
	}
	s.lifecycleMu.Unlock()
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	plan, err := s.preview(p)
	if err != nil {
		return core.Job{}, err
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.exiting {
		return core.Job{}, errors.New("应用正在退出，无法创建任务")
	}
	job, err := s.jobs.EnqueueProject(plan, p)
	if err == nil {
		s.emitQueueStatus()
	}
	return job, err
}

func (s *Service) Cancel(id string) error {
	err := s.jobs.Cancel(id)
	if err == nil {
		s.emitQueueStatus()
	}
	return err
}
func (s *Service) Jobs() []core.Job { s.ensureWorkspace(); return s.jobs.List() }
