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

	"kompeg/internal/core"
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
}

type binaryConfig struct {
	FFmpeg  string `json:"ffmpeg"`
	FFprobe string `json:"ffprobe"`
}

func configFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kompeg", "ffmpeg.json"), nil
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
	for _, variable := range []string{"KOMPEG_" + strings.ToUpper(name), strings.ToUpper(name) + "_PATH"} {
		if value := strings.TrimSpace(getenv(variable)); value != "" {
			return BinaryStatus{Path: binaryPath(value, name), Source: variable}
		}
	}
	if strings.TrimSpace(saved) != "" {
		return BinaryStatus{Path: binaryPath(saved, name), Source: "config"}
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
	return Environment{Platform: runtime.GOOS, Configurable: runtime.GOOS == "windows",
		FFmpeg:  selectBinary("ffmpeg", config.FFmpeg, os.Getenv),
		FFprobe: selectBinary("ffprobe", config.FFprobe, os.Getenv)}
}

func checkBinary(ctx context.Context, binary BinaryStatus, name string) (BinaryStatus, error) {
	cmd := exec.CommandContext(ctx, binary.Path, "-version")
	hidePlatformProcess(cmd)
	var stdout, stderr limitedBuffer
	stdout.limit, stderr.limit = 64*1024, 16*1024
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return binary, fmt.Errorf("%s 无法运行（来源：%s，路径：%s）：%w", name, binary.Source, binary.Path, err)
	}
	first := strings.TrimSpace(strings.SplitN(stdout.String(), "\n", 2)[0])
	if !strings.HasPrefix(first, name+" version ") {
		return binary, fmt.Errorf("所选文件不是有效的 %s 程序：%s", name, binary.Path)
	}
	binary.Version, binary.Available = first, true
	return binary, nil
}

func validateEnvironment(ctx context.Context, environment Environment) Environment {
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
		environment.Message = strings.Join(failures, "\n") + "\n请在设置中选择 FFmpeg 所在目录或 ffmpeg.exe，kompeg 会同时验证 ffprobe。"
		for _, binary := range []BinaryStatus{environment.FFmpeg, environment.FFprobe} {
			if !binary.Available && binary.Source != "PATH" && binary.Source != "config" {
				environment.Message += "\n环境变量 " + binary.Source + " 优先于应用设置；请修正该变量后重新启动 kompeg。"
			}
		}
	}
	return environment
}

func (s *Service) GetEnvironment() Environment {
	if runtime.GOOS != "windows" {
		return Environment{Platform: runtime.GOOS, Message: "当前平台尚未集成原生 FFmpeg。"}
	}
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	config, err := readBinaryConfig()
	if err != nil {
		return Environment{Platform: runtime.GOOS, Configurable: true, Message: "无法读取 FFmpeg 配置：" + err.Error()}
	}
	ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
	defer cancel()
	return validateEnvironment(ctx, discoverEnvironment(config))
}

func (s *Service) ConfigureFFmpeg(ffmpegPath, ffprobePath string) (Environment, error) {
	if runtime.GOOS != "windows" {
		return Environment{}, core.ErrUnsupported
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	for _, job := range s.jobs.List() {
		if job.State == core.JobQueued || job.State == core.JobPreparing || job.State == core.JobRunning || job.State == core.JobCancelling {
			return Environment{}, errors.New("请等待所有任务结束或取消任务后，再更改 FFmpeg 路径。")
		}
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
	ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
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
	s.runner.replace(newConfiguredRunner(selected.FFmpeg.Path, selected.FFprobe.Path))
	s.capsMu.Lock()
	s.caps = nil
	s.capsMu.Unlock()
	if s.app != nil {
		s.app.Event.Emit("environment:updated", selected)
	}
	return selected, nil
}

func (s *Service) ChooseFFmpeg() (*Environment, error) {
	if s.app == nil {
		return nil, errors.New("本机文件选择器不可用。")
	}
	path, err := s.app.Dialog.OpenFile().SetTitle("选择 ffmpeg.exe（同目录需有 ffprobe.exe）").AddFilter("FFmpeg", "ffmpeg.exe").PromptForSingleSelection()
	if err != nil || path == "" {
		return nil, err
	}
	environment, err := s.ConfigureFFmpeg(path, "")
	return &environment, err
}
