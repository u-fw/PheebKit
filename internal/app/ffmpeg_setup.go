package app

import (
	"context"
	"debug/pe"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"kompeg/internal/ffmpegsetup"
)

type FFmpegSetupStatus struct {
	ffmpegsetup.Progress
	Active  bool   `json:"active"`
	Error   string `json:"error"`
	Version string `json:"version"`
}

func (s *Service) GetFFmpegSetupStatus() FFmpegSetupStatus {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	return s.setupStatus
}

func (s *Service) StartFFmpegSetup(source string) (FFmpegSetupStatus, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return FFmpegSetupStatus{}, errors.New("自动下载目前仅支持 Windows x64；其他平台请选择适配的本机程序。")
	}
	packages, err := ffmpegsetup.Catalog(source)
	if err != nil {
		return FFmpegSetupStatus{}, err
	}
	return s.startFFmpegSetup(packages, false)
}

func (s *Service) StartCustomFFmpegSetup(link, sha256 string, trustBuild bool) (FFmpegSetupStatus, error) {
	pkg, err := ffmpegsetup.CustomArchive(link, sha256, trustBuild)
	if err != nil {
		return FFmpegSetupStatus{}, err
	}
	return s.startFFmpegSetup([]ffmpegsetup.Package{pkg}, true)
}

func (s *Service) startFFmpegSetup(packages []ffmpegsetup.Package, custom bool) (FFmpegSetupStatus, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return FFmpegSetupStatus{}, errors.New("自动下载目前仅支持 Windows x64；其他平台请选择适配的本机程序。")
	}
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	s.setupBusy.Store(true)
	defer func() { s.setupBusy.Store(s.setupStatus.Active) }()
	s.lifecycleMu.Lock()
	closing := s.exiting
	s.lifecycleMu.Unlock()
	if closing {
		return s.setupStatus, errors.New("应用正在退出，无法开始环境补全")
	}
	if s.setupStatus.Active {
		return s.setupStatus, errors.New("已有下载进行中。")
	}
	s.configMu.RLock()
	pending := s.hasPendingJobs()
	s.configMu.RUnlock()
	if pending {
		return s.setupStatus, errors.New("请等待所有任务结束或取消任务后，再下载配置 FFmpeg。")
	}
	if variable := explicitFFmpegOverride(); variable != "" {
		return s.setupStatus, errors.New("环境变量 " + variable + " 优先于下载配置。请先清除或修正该变量，并重启 kompeg。")
	}
	// Revalidate at the native entry point: a stale UI or another caller must
	// not download over a working environment. NewService has not run these
	// binaries yet, so the stored Ready flag alone cannot decide eligibility.
	s.configMu.RLock()
	checkCtx, checkCancel := context.WithTimeout(s.ctx, 8*time.Second)
	environment := validateEnvironment(checkCtx, s.environment)
	checkCancel()
	s.configMu.RUnlock()
	if environment.Ready {
		return s.setupStatus, errors.New("已找到可用的 FFmpeg 和 ffprobe，无需重复下载。需要切换版本时，请选择本机 FFmpeg 路径。")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return s.setupStatus, err
	}
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	s.setupCancel = cancel
	s.setupStatus = FFmpegSetupStatus{Active: true, Progress: ffmpegsetup.Progress{Stage: "selecting"}}
	go s.runFFmpegSetup(ctx, cancel, filepath.Join(cache, "kompeg", "tools"), packages, custom)
	return s.setupStatus, nil
}

func (s *Service) CancelFFmpegSetup() error {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if !s.setupStatus.Active {
		return nil
	}
	if s.setupStatus.Stage == "configuring" {
		return errors.New("正在验证并保存配置，请稍候。")
	}
	s.setupStatus.Stage = "cancelling"
	s.setupCancel()
	return nil
}

func (s *Service) runFFmpegSetup(ctx context.Context, cancel context.CancelFunc, root string, packages []ffmpegsetup.Package, custom bool) {
	defer cancel()
	report := func(progress ffmpegsetup.Progress) {
		s.setupMu.Lock()
		defer s.setupMu.Unlock()
		if s.setupStatus.Stage != "cancelling" {
			s.setupStatus.Progress = progress
		}
	}
	installer := ffmpegsetup.New(report)
	if custom {
		installer = ffmpegsetup.NewCustom(report)
	}
	result, err := installer.Install(ctx, root, packages)
	if err == nil {
		s.setupMu.Lock()
		err = ctx.Err()
		if err == nil {
			s.setupStatus.Progress = ffmpegsetup.Progress{Stage: "configuring", Package: result.Label}
			s.setupStatus.Version = result.Version
		}
		s.setupMu.Unlock()
		if err == nil {
			err = validateDownloadedArchitecture(result.FFmpeg, result.FFprobe)
		}
		if err == nil {
			var environment Environment
			environment, err = s.ConfigureFFmpeg(result.FFmpeg, result.FFprobe)
			if err == nil && result.Version == "" {
				fields := strings.Fields(environment.FFmpeg.Version)
				if len(fields) >= 3 {
					result.Version = fields[2]
				}
			}
		}
		if err != nil {
			os.RemoveAll(result.Directory)
		}
	}
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	s.setupStatus.Active = false
	s.setupBusy.Store(false)
	s.setupCancel = nil
	if errors.Is(err, context.Canceled) {
		s.setupStatus.Stage = "cancelled"
		s.setupStatus.Error = ""
	} else if err != nil {
		s.setupStatus.Stage = "failed"
		s.setupStatus.Error = err.Error()
	} else {
		s.setupStatus.Stage = "completed"
		s.setupStatus.Version = result.Version
	}
}

func validateDownloadedArchitecture(paths ...string) error {
	for _, path := range paths {
		file, err := pe.Open(path)
		if err != nil {
			return errors.New("下载包中的 " + filepath.Base(path) + " 不是有效的 Windows 可执行程序。")
		}
		machine := file.Machine
		file.Close()
		if machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			return errors.New("下载包中的 " + filepath.Base(path) + " 不是 Windows x64 构建，请选择 amd64 / x86_64 版本。")
		}
	}
	return nil
}
