package app

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"kompeg/internal/compiler"
	"kompeg/internal/core"
	"kompeg/internal/jobs"
)

// Service is the Wails boundary; the compiler and job engine know nothing of UI.
type Service struct {
	app     *application.App
	runner  core.Runner
	jobs    *jobs.Manager
	ctx     context.Context
	cancel  context.CancelFunc
	mediaMu sync.RWMutex
	media   map[string]core.Media
	capsMu  sync.Mutex
	caps    *core.Capabilities
}

func NewService() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{ctx: ctx, cancel: cancel, runner: newPlatformRunner(), media: make(map[string]core.Media)}
	s.jobs = jobs.New(s.runner, func(job core.Job) {
		if s.app != nil {
			s.app.Event.Emit("job:updated", job)
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
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	m, err := s.runner.Probe(ctx, source(uri))
	if err != nil {
		return core.Media{}, err
	}
	s.mediaMu.Lock()
	s.media[m.Source.URI] = m
	s.mediaMu.Unlock()
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
	return s.app.Dialog.SaveFile().SetDirectory(filepath.Dir(suggested)).SetFilename(filepath.Base(suggested)).AddFilter("视频输出", "*.mp4;*.mkv;*.webm").PromptForSingleSelection()
}

func (s *Service) Capabilities() (core.Capabilities, error) {
	s.capsMu.Lock()
	defer s.capsMu.Unlock()
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
	if len(p.Inputs) != 1 {
		return core.ExecutionPlan{}, errors.New("the spike requires exactly one input")
	}
	s.mediaMu.RLock()
	m, found := s.media[p.Inputs[0].URI]
	s.mediaMu.RUnlock()
	if !found {
		var err error
		m, err = s.Probe(p.Inputs[0].URI)
		if err != nil {
			return core.ExecutionPlan{}, err
		}
	}
	caps, err := s.Capabilities()
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	plan, err := compiler.Compile(p, m, caps)
	if err != nil {
		return core.ExecutionPlan{}, err
	}
	plan.Preview = compiler.Preview(previewBinary(s.runner), plan.Args)
	return plan, nil
}

func (s *Service) Start(p core.Project) (core.Job, error) {
	plan, err := s.Preview(p)
	if err != nil {
		return core.Job{}, err
	}
	return s.jobs.Enqueue(plan)
}

func (s *Service) Cancel(id string) error { return s.jobs.Cancel(id) }
func (s *Service) Jobs() []core.Job       { return s.jobs.List() }
