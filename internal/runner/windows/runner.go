// Package windows is the process adapter. Core business logic never starts a shell.
package windows

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kompeg/internal/capabilities"
	"kompeg/internal/core"
	"kompeg/internal/ffmpegpolicy"
	"kompeg/internal/probe"
	"kompeg/internal/sourcefile"
)

type Runner struct {
	FFmpeg  string
	FFprobe string
	mu      sync.Mutex
	active  map[string]context.CancelFunc
}

var _ core.Runner = (*Runner)(nil)

func New(ffmpegPath, ffprobePath string) *Runner {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg.exe"
	}
	if ffprobePath == "" {
		ffprobePath = "ffprobe.exe"
	}
	return &Runner{FFmpeg: resolveBinary(ffmpegPath), FFprobe: resolveBinary(ffprobePath), active: make(map[string]context.CancelFunc)}
}

func (r *Runner) Probe(ctx context.Context, source core.Source) (core.Media, error) {
	if err := localFile(source, true); err != nil {
		return core.Media{}, err
	}
	fingerprint, err := sourcefile.Fingerprint(source.URI)
	if err != nil {
		return core.Media{}, err
	}
	args := append([]string{"-v", "error", "-show_format", "-show_streams", "-show_chapters", "-show_programs", "-of", "json"}, ffmpegpolicy.LocalInput(source.URI)...)
	data, err := capture(ctx, r.FFprobe, args, false)
	if err != nil {
		return core.Media{}, fmt.Errorf("probe media: %w", err)
	}
	media, err := probe.Parse(data, source)
	if err != nil {
		return core.Media{}, err
	}
	if err := sourcefile.Verify(source.URI, fingerprint); err != nil {
		return core.Media{}, err
	}
	media.SourceFingerprint = fingerprint
	// Filesystem size is authoritative when container metadata omits it.
	if stat, err := os.Stat(source.URI); err == nil {
		media.SizeBytes = stat.Size()
	}
	return media, nil
}

func (r *Runner) Capabilities(ctx context.Context) (core.Capabilities, error) {
	version, err := capture(ctx, r.FFmpeg, []string{"-version"}, false)
	if err != nil {
		return core.Capabilities{}, fmt.Errorf("scan FFmpeg version: %w", err)
	}
	build, err := capture(ctx, r.FFmpeg, []string{"-hide_banner", "-buildconf"}, true)
	if err != nil {
		return core.Capabilities{}, fmt.Errorf("scan FFmpeg configuration: %w", err)
	}
	encoders, err := capture(ctx, r.FFmpeg, []string{"-hide_banner", "-encoders"}, false)
	if err != nil {
		return core.Capabilities{}, fmt.Errorf("scan FFmpeg encoders: %w", err)
	}
	hwaccels, err := capture(ctx, r.FFmpeg, []string{"-hide_banner", "-hwaccels"}, false)
	if err != nil {
		return core.Capabilities{}, fmt.Errorf("scan FFmpeg hardware backends: %w", err)
	}
	parsed := capabilities.ParseEncoders(string(encoders))
	qualified := r.qualifyVideoEncoders(ctx, parsed)
	if err := ctx.Err(); err != nil {
		return core.Capabilities{}, fmt.Errorf("scan GPU encoders: %w", err)
	}
	hardware := make([]string, 0)
	for _, line := range strings.Split(string(hwaccels), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, ":") {
			hardware = append(hardware, line)
		}
	}
	return core.Capabilities{Version: strings.SplitN(strings.TrimSpace(string(version)), "\n", 2)[0],
		BuildConfiguration: strings.TrimSpace(string(build)), Encoders: parsed,
		Codecs: capabilities.ResolveQualified(qualified), Hardware: hardware, Platform: "windows",
		VideoEncoders: qualified, GPUs: detectGPUs()}, nil
}

func (r *Runner) Start(parent context.Context, plan core.ExecutionPlan, callbacks core.Callbacks) (result core.Result, resultErr error) {
	if plan.JobID == "" {
		return core.Result{}, fmt.Errorf("execution requires a job ID")
	}
	if err := validatePlan(plan); err != nil {
		return core.Result{}, err
	}
	if err := sourcefile.Verify(plan.Input.URI, plan.SourceFingerprint); err != nil {
		return core.Result{}, err
	}
	outputParent := filepath.Dir(plan.Output.URI)
	scratchDir, err := os.MkdirTemp(outputParent, ".kompeg-*")
	if err != nil {
		return core.Result{}, fmt.Errorf("create output scratch directory: %w", err)
	}
	defer func() {
		if err := cleanupScratch(scratchDir, outputParent); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("clean output scratch directory: %w", err))
		}
	}()
	scratchOutput := filepath.Join(scratchDir, "output"+filepath.Ext(plan.Output.URI))
	args := append([]string(nil), plan.Args...)
	args[len(args)-1] = scratchOutput
	ctx, cancel := context.WithCancel(parent)
	r.mu.Lock()
	if _, exists := r.active[plan.JobID]; exists {
		r.mu.Unlock()
		cancel()
		return core.Result{}, fmt.Errorf("job %s is already running", plan.JobID)
	}
	r.active[plan.JobID] = cancel
	r.mu.Unlock()
	defer func() {
		cancel()
		r.mu.Lock()
		delete(r.active, plan.JobID)
		r.mu.Unlock()
	}()

	cmd := exec.CommandContext(ctx, r.FFmpeg, args...)
	hideProcess(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return core.Result{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return core.Result{}, err
	}
	started := time.Now()
	if err := cmd.Start(); err != nil {
		return core.Result{}, fmt.Errorf("start FFmpeg: %w", err)
	}
	var readers sync.WaitGroup
	readers.Add(2)
	var progressErr, logErr error
	var tail string
	go func() {
		defer readers.Done()
		progressErr = readProgress(stdout, plan.DurationSeconds, callbacks.Progress)
	}()
	go func() { defer readers.Done(); tail, logErr = readLogs(stderr, callbacks.Log) }()
	readers.Wait()
	err = cmd.Wait()
	if ctx.Err() != nil {
		return core.Result{}, ctx.Err()
	}
	if err != nil {
		return core.Result{}, fmt.Errorf("FFmpeg failed: %w\n%s", err, tail)
	}
	if progressErr != nil || logErr != nil {
		return core.Result{}, fmt.Errorf("read FFmpeg output: progress=%v log=%v", progressErr, logErr)
	}
	stat, err := os.Stat(scratchOutput)
	if err != nil {
		return core.Result{}, fmt.Errorf("inspect encoded output: %w", err)
	}
	if !stat.Mode().IsRegular() || stat.Size() == 0 {
		return core.Result{}, fmt.Errorf("encoder did not create a nonempty regular output file")
	}
	if ctx.Err() != nil {
		return core.Result{}, ctx.Err()
	}
	// A source edited during encoding must not be published as the analyzed
	// video, even if FFmpeg happened to finish reading the older open handle.
	if err := sourcefile.Verify(plan.Input.URI, plan.SourceFingerprint); err != nil {
		return core.Result{}, err
	}
	// Publish only a completed file. The platform primitive never replaces an
	// output that appeared while encoding, even if another process created it.
	if err := publishFile(scratchOutput, plan.Output.URI); err != nil {
		return core.Result{}, fmt.Errorf("publish encoded output without overwrite: %w", err)
	}
	elapsed := time.Since(started).Seconds()
	result = core.Result{InputSizeBytes: plan.InputSizeBytes, OutputSizeBytes: stat.Size(),
		ElapsedSeconds: elapsed, Output: plan.Output}
	if plan.InputSizeBytes > 0 {
		result.SavedPercent = (1 - float64(stat.Size())/float64(plan.InputSizeBytes)) * 100
	}
	if elapsed > 0 {
		result.AverageSpeed = plan.DurationSeconds / elapsed
	}
	return result, nil
}

func cleanupScratch(path, parent string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	absParent, err := filepath.Abs(parent)
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Dir(absPath), filepath.Clean(absParent)) || !strings.HasPrefix(filepath.Base(absPath), ".kompeg-") {
		return fmt.Errorf("scratch directory is outside the selected output directory")
	}
	return os.RemoveAll(absPath)
}

func (r *Runner) Cancel(jobID string) error {
	r.mu.Lock()
	cancel, exists := r.active[jobID]
	r.mu.Unlock()
	if !exists {
		return fmt.Errorf("job %s is not running", jobID)
	}
	cancel()
	return nil
}

func localFile(source core.Source, mustExist bool) error {
	if source.Kind != core.SourceLocalFile || !filepath.IsAbs(source.URI) || strings.ContainsRune(source.URI, 0) {
		return fmt.Errorf("Windows runner requires an absolute local-file source")
	}
	if mustExist {
		stat, err := os.Stat(source.URI)
		if err != nil {
			return fmt.Errorf("inspect input: %w", err)
		}
		if !stat.Mode().IsRegular() {
			return fmt.Errorf("input is not a regular file")
		}
	}
	return nil
}

func validatePlan(plan core.ExecutionPlan) error {
	if err := localFile(plan.Input, true); err != nil {
		return err
	}
	if err := localFile(plan.Output, false); err != nil {
		return err
	}
	if strings.EqualFold(filepath.Clean(plan.Input.URI), filepath.Clean(plan.Output.URI)) {
		return fmt.Errorf("cannot overwrite input")
	}
	if _, err := os.Stat(plan.Output.URI); err == nil {
		return fmt.Errorf("output already exists")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect output: %w", err)
	}
	// Plans come from the compiler; assert the adapter's process invariants too.
	if len(plan.Args) == 0 || plan.Args[len(plan.Args)-1] != plan.Output.URI {
		return fmt.Errorf("plan output does not match output source")
	}
	hasNoOverwrite, hasInput, hasProgress := false, false, false
	for i, arg := range plan.Args {
		if arg == "-y" {
			return fmt.Errorf("overwrite option is forbidden")
		}
		hasNoOverwrite = hasNoOverwrite || arg == "-n"
		if arg == "-i" && i+1 < len(plan.Args) {
			if hasInput || plan.Args[i+1] != plan.Input.URI {
				return fmt.Errorf("plan contains an unexpected input")
			}
			hasInput = true
		}
		if arg == "-progress" && i+1 < len(plan.Args) && plan.Args[i+1] == "pipe:1" {
			hasProgress = true
		}
	}
	if !hasNoOverwrite || !hasInput || !hasProgress {
		return fmt.Errorf("plan is missing safe input, overwrite or progress settings")
	}
	return nil
}
