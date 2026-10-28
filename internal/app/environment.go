package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"pheebkit/internal/bundledffmpeg"
	"pheebkit/internal/compat"
	"pheebkit/internal/core"
	"pheebkit/internal/processguard"
)

type BinaryStatus struct {
	Path      string `json:"path"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	Available bool   `json:"available"`
}

type Environment struct {
	Platform     string       `json:"platform"`
	Ready        bool         `json:"ready"`
	FFmpeg       BinaryStatus `json:"ffmpeg"`
	FFprobe      BinaryStatus `json:"ffprobe"`
	Message      string       `json:"message"`
	Configurable bool         `json:"configurable"`
	bundleError  string
}

type binaryConfig struct {
	FFmpeg  string `json:"ffmpeg"`
	FFprobe string `json:"ffprobe"`
}

// A newly selected executable may have a slow first launch on a clean system.
// Validation stays bounded while allowing the complete tool pair to start.
const environmentCheckTimeout = 30 * time.Second

func configFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(compat.ConfigDirectory(dir), "ffmpeg.json"), nil
}

func readBinaryConfig() (binaryConfig, error) {
	path, err := configFile()
	if err != nil {
		return binaryConfig{}, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return binaryConfig{}, nil
	}
	if err != nil {
		return binaryConfig{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64*1024))
	decoder.DisallowUnknownFields()
	var config binaryConfig
	if err := decoder.Decode(&config); err != nil {
		return binaryConfig{}, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return binaryConfig{}, errors.New("invalid FFmpeg configuration")
	}
	return config, nil
}

func writeBinaryConfig(config binaryConfig) error {
	path, err := configFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".ffmpeg-*.json")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
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
	return os.Rename(temporary, path)
}

func selectBinary(name, saved string, getenv func(string) string) BinaryStatus {
	return selectBinaryWithBundle(name, saved, getenv, "")
}

func selectBinaryWithBundle(name, saved string, getenv func(string) string, bundledDirectory string) BinaryStatus {
	for _, variable := range compat.BinaryVariables(name) {
		if value := strings.TrimSpace(getenv(variable)); value != "" {
			return BinaryStatus{Path: binaryPath(value, name), Source: variable}
		}
	}
	if strings.TrimSpace(saved) != "" {
		return BinaryStatus{Path: binaryPath(saved, name), Source: "config"}
	}
	if strings.TrimSpace(bundledDirectory) != "" {
		return BinaryStatus{Path: filepath.Join(bundledDirectory, name+".exe"), Source: "bundled"}
	}
	return BinaryStatus{Path: binaryPath(name, name), Source: "PATH"}
}

func binaryPath(value, name string) string {
	value = strings.Trim(strings.TrimSpace(value), "\"")
	if info, err := os.Stat(value); err == nil && info.IsDir() {
		extension := ""
		if runtime.GOOS == "windows" {
			extension = ".exe"
		}
		value = filepath.Join(value, name+extension)
	}
	if resolved, err := exec.LookPath(value); err == nil {
		value = resolved
	}
	return resolvePlatformBinary(value)
}

func discoverEnvironment(config binaryConfig) Environment {
	needsBundled := false
	for _, item := range []struct {
		name, saved string
	}{
		{"ffmpeg", config.FFmpeg},
		{"ffprobe", config.FFprobe},
	} {
		if strings.TrimSpace(item.saved) != "" {
			continue
		}
		explicit := false
		for _, variable := range compat.BinaryVariables(item.name) {
			if strings.TrimSpace(os.Getenv(variable)) != "" {
				explicit = true
				break
			}
		}
		needsBundled = needsBundled || !explicit
	}
	bundledDirectory := ""
	bundleError := ""
	if needsBundled {
		bundledDirectory, bundleError = bundledBinaryDirectory()
	}
	return Environment{
		Platform:     runtime.GOOS,
		Configurable: runtime.GOOS == "windows",
		FFmpeg:       selectBinaryWithBundle("ffmpeg", config.FFmpeg, os.Getenv, bundledDirectory),
		FFprobe:      selectBinaryWithBundle("ffprobe", config.FFprobe, os.Getenv, bundledDirectory),
		bundleError:  bundleError,
	}
}

func bundledBinaryDirectory() (string, string) {
	if executable, err := os.Executable(); err == nil {
		directory := filepath.Join(filepath.Dir(executable), "runtime", "ffmpeg", "bin")
		if executablePairAvailable(directory) {
			return directory, ""
		}
	}
	resolved, err := bundledffmpeg.Resolve()
	if errors.Is(err, bundledffmpeg.ErrUnavailable) {
		return "", ""
	}
	if err != nil {
		return "", "内置 FFmpeg 无法准备：" + err.Error()
	}
	directory := filepath.Join(resolved.Directory, "bin")
	if !executablePairAvailable(directory) {
		return "", "内置 FFmpeg 缺少 ffmpeg.exe 或 ffprobe.exe。"
	}
	return directory, ""
}

func executablePairAvailable(directory string) bool {
	for _, name := range []string{"ffmpeg.exe", "ffprobe.exe"} {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func checkBinary(ctx context.Context, binary BinaryStatus, name string) (BinaryStatus, error) {
	binary.Available, binary.Version = false, ""
	cmd := exec.CommandContext(ctx, binary.Path, "-version")
	var stdout, stderr limitedBuffer
	stdout.limit, stderr.limit = 64*1024, 16*1024
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := processguard.Run(cmd); err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		detail := strings.TrimSpace(stderr.String())
		if len(detail) > 1024 {
			detail = detail[:1024] + "…"
		}
		if detail != "" {
			detail = "\n" + detail
		}
		return binary, fmt.Errorf("%s 无法运行（来源：%s，路径：%s）：%w%s", name, binary.Source, binary.Path, err, detail)
	}
	first := strings.TrimSpace(strings.SplitN(stdout.String(), "\n", 2)[0])
	if !strings.HasPrefix(first, name+" version ") {
		return binary, fmt.Errorf("所选文件不是有效的 %s 程序：%s", name, binary.Path)
	}
	binary.Version, binary.Available = first, true
	return binary, nil
}

func validateEnvironment(ctx context.Context, environment Environment) Environment {
	environment.Message = ""
	var failures []string
	var err error
	environment.FFmpeg, err = checkBinary(ctx, environment.FFmpeg, "ffmpeg")
	if err != nil {
		failures = append(failures, err.Error())
	}
	environment.FFprobe, err = checkBinary(ctx, environment.FFprobe, "ffprobe")
	if err != nil {
		failures = append(failures, err.Error())
	}
	environment.Ready = len(failures) == 0
	if !environment.Ready {
		environment.Message = strings.Join(failures, "\n") + "\n请在设置中一键下载补全，或选择 FFmpeg 所在目录 / ffmpeg.exe，菲比工具箱会同时验证 ffprobe。"
		for _, binary := range []BinaryStatus{environment.FFmpeg, environment.FFprobe} {
			if !binary.Available && binary.Source != "PATH" && binary.Source != "config" && binary.Source != "bundled" {
				environment.Message += "\n环境变量 " + binary.Source + " 优先于应用设置；请修正该变量后重新启动菲比工具箱。"
			}
		}
	}
	if environment.bundleError != "" {
		environment.Message = appendEnvironmentMessage(environment.Message, environment.bundleError)
	}
	return environment
}

// InspectEnvironment performs the same bounded discovery and executable checks
// as the desktop service without opening a window or starting media jobs.
func InspectEnvironment(ctx context.Context) Environment {
	config, err := readBinaryConfig()
	environment := discoverEnvironment(config)
	if err != nil {
		environment.Message = appendEnvironmentMessage(environment.Message, "无法读取共享 FFmpeg 配置："+err.Error())
	}
	return validateEnvironment(ctx, environment)
}

func (s *Service) GetEnvironment() Environment {
	if runtime.GOOS != "windows" {
		return Environment{Platform: runtime.GOOS, Message: "当前平台尚未集成原生 FFmpeg。"}
	}
	s.configMu.RLock()
	selected := s.environment
	configured := s.configuration
	s.configMu.RUnlock()
	ctx, cancel := context.WithTimeout(s.ctx, environmentCheckTimeout)
	defer cancel()
	// Report the binaries owned by this instance, not settings another window
	// may have written while this instance was processing jobs.
	validated := validateEnvironment(ctx, selected)
	config, err := readBinaryConfig()
	external := Environment{}
	if err == nil {
		external = discoverEnvironment(config)
	}
	s.configMu.Lock()
	if configured == s.configuration && sameSelectedEnvironment(selected, s.environment) {
		s.environment = validated
	}
	environment := s.environment
	configuration := s.configuration
	configurationError := s.configurationError
	s.configMu.Unlock()
	if err != nil {
		environment.Message = appendEnvironmentMessage(environment.Message, "无法读取共享 FFmpeg 配置："+err.Error()+"。本窗口继续使用上方路径。")
	} else if config != configuration || !sameSelectedEnvironment(external, environment) {
		environment.Message = appendEnvironmentMessage(environment.Message, "检测到 FFmpeg 配置已在其他窗口或应用外更改。本窗口继续使用上方路径；请等待任务结束，再点击重新检测以同步配置。")
	} else if configurationError != "" {
		environment.Message = appendEnvironmentMessage(environment.Message, configurationError)
	}
	s.configMu.Lock()
	if configured == s.configuration && sameSelectedEnvironment(selected, s.environment) {
		s.environment.Message = environment.Message
	}
	s.configMu.Unlock()
	return environment
}

func appendEnvironmentMessage(message, addition string) string {
	if message == "" {
		return addition
	}
	return message + "\n" + addition
}

func sameSelectedEnvironment(a, b Environment) bool {
	return a.FFmpeg.Path == b.FFmpeg.Path && a.FFmpeg.Source == b.FFmpeg.Source && a.FFprobe.Path == b.FFprobe.Path && a.FFprobe.Source == b.FFprobe.Source
}

func (s *Service) hasPendingJobs() bool {
	if s.healthBusy.Load() {
		return true
	}
	for _, job := range s.jobs.List() {
		if job.State == core.JobQueued || job.State == core.JobPreparing || job.State == core.JobRunning || job.State == core.JobCancelling {
			return true
		}
	}
	return false
}

// reloadEnvironment is called with configMu held exclusively. Job creation
// holds its read lock, so an idle check and runner replacement are one operation.
func (s *Service) reloadEnvironment() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	config, err := readBinaryConfig()
	if err != nil {
		return fmt.Errorf("无法读取 FFmpeg 配置，仍使用本窗口原有路径：%w", err)
	}
	selected := discoverEnvironment(config)
	if config == s.configuration && sameSelectedEnvironment(selected, s.environment) {
		s.configurationError = ""
		return nil
	}
	if s.hasPendingJobs() {
		return errors.New("FFmpeg 配置已更改，请等待所有任务结束或取消任务后，再重新检测并同步路径。")
	}
	ctx, cancel := context.WithTimeout(s.ctx, environmentCheckTimeout)
	defer cancel()
	selected = validateEnvironment(ctx, selected)
	if !selected.Ready {
		return errors.New("无法同步新配置，本窗口继续使用原有路径。\n" + selected.Message)
	}
	s.applyEnvironment(selected, config)
	return nil
}

func (s *Service) applyEnvironment(environment Environment, config binaryConfig) {
	s.runner.replace(newConfiguredRunner(environment.FFmpeg.Path, environment.FFprobe.Path))
	s.environment, s.configuration, s.configurationError = environment, config, ""
	s.capsMu.Lock()
	s.caps = nil
	s.capsMu.Unlock()
	if s.app != nil {
		s.app.Emit("environment:updated", environment)
	}
}

func (s *Service) ConfigureFFmpeg(ffmpegPath, ffprobePath string) (Environment, error) {
	if runtime.GOOS != "windows" {
		return Environment{}, core.ErrUnsupported
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if s.hasPendingJobs() {
		return Environment{}, errors.New("请等待所有任务结束或取消任务后，再更改 FFmpeg 路径。")
	}
	if variable := explicitFFmpegOverride(); variable != "" {
		return Environment{}, fmt.Errorf("环境变量 %s 优先于应用配置。请先清除或修正该变量，并重启菲比工具箱，再切换本机路径。", variable)
	}
	if strings.TrimSpace(ffmpegPath) == "" {
		return Environment{}, errors.New("请选择 ffmpeg.exe 或填写 FFmpeg 所在目录。")
	}
	config := binaryConfig{FFmpeg: binaryPath(ffmpegPath, "ffmpeg"), FFprobe: binaryPath(ffprobePath, "ffprobe")}
	if strings.TrimSpace(ffprobePath) == "" {
		config.FFprobe = binaryPath(filepath.Join(filepath.Dir(config.FFmpeg), "ffprobe.exe"), "ffprobe")
	}
	for _, binary := range []struct{ name, path string }{{"ffmpeg", config.FFmpeg}, {"ffprobe", config.FFprobe}} {
		if !filepath.IsAbs(binary.path) {
			return Environment{}, fmt.Errorf("%s 路径必须是本机绝对路径。", binary.name)
		}
	}
	ctx, cancel := context.WithTimeout(s.ctx, environmentCheckTimeout)
	defer cancel()
	chosen := validateEnvironment(ctx, Environment{Platform: runtime.GOOS, Configurable: true, FFmpeg: BinaryStatus{Path: config.FFmpeg, Source: "config"}, FFprobe: BinaryStatus{Path: config.FFprobe, Source: "config"}})
	if !chosen.Ready {
		return chosen, errors.New(chosen.Message)
	}
	selected := validateEnvironment(ctx, discoverEnvironment(config))
	if !selected.Ready {
		return selected, errors.New(selected.Message)
	}
	if err := writeBinaryConfig(config); err != nil {
		return Environment{}, fmt.Errorf("保存 FFmpeg 配置失败：%w", err)
	}
	s.applyEnvironment(selected, config)
	return selected, nil
}

func explicitFFmpegOverride() string {
	for _, variable := range append(compat.BinaryVariables("ffmpeg"), compat.BinaryVariables("ffprobe")...) {
		if strings.TrimSpace(os.Getenv(variable)) != "" {
			return variable
		}
	}
	return ""
}

func (s *Service) ChooseFFmpeg() (*Environment, error) {
	if s.app == nil {
		return nil, errors.New("本机文件选择器不可用。")
	}
	path, err := s.app.OpenFile(FileDialog{Title: "选择 ffmpeg.exe（同目录需有 ffprobe.exe）", FilterName: "FFmpeg", Pattern: "ffmpeg.exe"})
	if err != nil || path == "" {
		return nil, err
	}
	environment, err := s.ConfigureFFmpeg(path, "")
	return &environment, err
}
