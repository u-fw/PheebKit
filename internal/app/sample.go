package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"path/filepath"
	"strings"

	"pheebkit/internal/compiler"
	"pheebkit/internal/core"
	"pheebkit/internal/sourcefile"
)

type SampleComparison struct {
	Original        core.Media `json:"original"`
	Output          core.Media `json:"output"`
	StartSeconds    float64    `json:"startSeconds"`
	DurationSeconds float64    `json:"durationSeconds"`
}

// StartSample creates a regular cancellable job with a distinct output beside
// the proposed result. The UI declares the bounded range and audio re-encode.
func (s *Service) StartSample(p core.Project, seconds float64) (core.Job, error) {
	if p.Settings.Recovery != nil {
		return core.Job{}, errors.New("恢复任务需要完整复检，不支持样片；请先恢复再试压")
	}
	if p.Settings.Operation != "" && p.Settings.Operation != "compress" {
		return core.Job{}, errors.New("样片试压仅适用于压缩任务")
	}
	if seconds < 1 || seconds > 30 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return core.Job{}, errors.New("样片时长需要在 1–30 秒之间")
	}
	if len(p.Inputs) != 1 || p.Output.URI == "" {
		return core.Job{}, errors.New("请先选择媒体和输出位置")
	}
	media, err := s.importedMedia(p.Inputs[0].URI)
	if err != nil {
		return core.Job{}, err
	}
	if !filepath.IsAbs(p.Output.URI) || p.Output.Kind != core.SourceLocalFile {
		return core.Job{}, errors.New("输出必须是本地绝对路径")
	}
	// Validate the original intent before narrowing it; malformed ranges must
	// not become acceptable by clamping them into a different user selection.
	if err := compiler.ValidateSettings(p.Settings); err != nil {
		return core.Job{}, err
	}
	rate := 0.0
	for _, stream := range media.Streams {
		if stream.CodecType == "video" && !stream.AttachedPicture && !stream.TimedThumbnails {
			rate = stream.FrameRate
			break
		}
	}
	selectedDuration, err := compiler.SelectedDuration(p.Settings, media.DurationSeconds, rate)
	if err != nil {
		return core.Job{}, err
	}
	if err := compiler.ValidateAudioDuration(p.Settings, selectedDuration); err != nil {
		return core.Job{}, err
	}
	start, end := 0.0, media.DurationSeconds
	if len(p.Settings.Segments) > 0 {
		start, end = p.Settings.Segments[0].StartSeconds, p.Settings.Segments[0].EndSeconds
	}
	if trim := p.Settings.Trim; trim != nil {
		if trim.StartSeconds < 0 || trim.EndSeconds > end || trim.StartSeconds >= trim.EndSeconds {
			return core.Job{}, errors.New("剪辑区间无效")
		}
		start, end = trim.StartSeconds, trim.EndSeconds
	}
	if end <= start || end <= 0 || math.IsNaN(end) || math.IsInf(end, 0) {
		return core.Job{}, errors.New("样片需要已知的媒体时长")
	}
	end = math.Min(end, start+seconds)
	p = core.CloneProject(p)
	if p.Settings.AudioProcessing != nil {
		p.Settings.AudioProcessing.FadeInSeconds = 0
		p.Settings.AudioProcessing.FadeOutSeconds = 0
	}
	if len(p.Settings.Segments) > 0 {
		p.Settings.Segments = nil
	}
	p.Settings.Trim = &core.TimeRange{StartSeconds: start, EndSeconds: end}
	if p.Settings.Audio == "copy" {
		p.Settings.Audio = "aac"
	}
	// Target size is a budget for the original selected range, not for 10s.
	// Keep its bitrate intent proportional when narrowing to the sample.
	if p.Settings.Mode == "size" {
		p.Settings.TargetSizeMB *= (end - start) / selectedDuration
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return core.Job{}, err
	}
	suffix := hex.EncodeToString(random[:])
	p.ID = "sample-" + suffix
	ext := filepath.Ext(p.Output.URI)
	p.Output = source(strings.TrimSuffix(p.Output.URI, ext) + "-sample-" + suffix + ext)
	return s.Start(p)
}

func (s *Service) GetSampleComparison(id string) (SampleComparison, error) {
	job, found := s.jobs.Get(id)
	if !found || job.State != core.JobCompleted || job.Project == nil || !strings.HasPrefix(job.ProjectID, "sample-") || job.Trim == nil {
		return SampleComparison{}, errors.New("请选择已完成的样片任务")
	}
	if job.SourceFingerprint == "" {
		return SampleComparison{}, errors.New("旧样片缺少源文件校验信息，请重新试压")
	}
	if err := sourcefile.Verify(job.Input.URI, job.SourceFingerprint); err != nil {
		return SampleComparison{}, err
	}
	original, err := s.Probe(job.Input.URI)
	if err != nil {
		return SampleComparison{}, err
	}
	output, err := s.Probe(job.Output.URI)
	if err != nil {
		return SampleComparison{}, err
	}
	if err := sourcefile.Verify(job.Input.URI, job.SourceFingerprint); err != nil {
		return SampleComparison{}, err
	}
	return SampleComparison{Original: original, Output: output, StartSeconds: job.Trim.StartSeconds, DurationSeconds: job.Trim.EndSeconds - job.Trim.StartSeconds}, nil
}
