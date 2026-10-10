package core

import (
	"context"
	"errors"
	"time"
)

var ErrUnsupported = errors.New("platform FFmpeg adapter is not integrated in this technical spike")

type Runner interface {
	Probe(context.Context, Source) (Media, error)
	Capabilities(context.Context) (Capabilities, error)
	Start(context.Context, ExecutionPlan, Callbacks) (Result, error)
	Cancel(jobID string) error
}

type ExecutionPlan struct {
	JobID             string     `json:"jobID"`
	ProjectID         string     `json:"projectID"`
	Input             Source     `json:"input"`
	SourceFingerprint string     `json:"sourceFingerprint,omitempty"`
	Output            Source     `json:"output"`
	Args              []string   `json:"args"`
	Preview           string     `json:"preview"`
	DurationSeconds   float64    `json:"durationSeconds"`
	InputSizeBytes    int64      `json:"inputSizeBytes"`
	Encoder           string     `json:"encoder"`
	EncoderKind       string     `json:"encoderKind"`
	EncoderLabel      string     `json:"encoderLabel"`
	EncoderReason     string     `json:"encoderReason"`
	Trim              *TimeRange `json:"trim,omitempty"`
}

type Capabilities struct {
	Version            string         `json:"version"`
	BuildConfiguration string         `json:"buildConfiguration"`
	Encoders           []Encoder      `json:"encoders"`
	Codecs             []Codec        `json:"codecs"`
	Hardware           []string       `json:"hardware"`
	Platform           string         `json:"platform"`
	VideoEncoders      []VideoEncoder `json:"videoEncoders"`
	GPUs               []GPU          `json:"gpus"`
}

// VideoEncoder reports runtime-qualified hardware encoders, rather than merely
// the wrappers included in the FFmpeg build. PixelFormats contains tested input
// formats for GPU encoders; CPU entries retain the established software policy.
type VideoEncoder struct {
	Name         string   `json:"name"`
	Codec        string   `json:"codec"`
	Label        string   `json:"label"`
	Kind         string   `json:"kind"`
	Available    bool     `json:"available"`
	Reason       string   `json:"reason"`
	PixelFormats []string `json:"pixelFormats"`
}

type GPU struct {
	Name   string `json:"name"`
	Vendor string `json:"vendor"`
}

type Encoder struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MediaType   string `json:"mediaType"`
}

// Codec is a UI-facing family resolved from the actual encoder scan.
type Codec struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Encoder   string `json:"encoder"`
	Available bool   `json:"available"`
}

type Progress struct {
	Percent            float64 `json:"percent"`
	FPS                float64 `json:"fps"`
	Speed              float64 `json:"speed"`
	CurrentTimeSeconds float64 `json:"currentTimeSeconds"`
	TotalTimeSeconds   float64 `json:"totalTimeSeconds"`
	OutputSizeBytes    int64   `json:"outputSizeBytes"`
	Frame              int64   `json:"frame"`
	Finished           bool    `json:"finished"`
}

type Callbacks struct {
	Progress func(Progress)
	Log      func(string)
}

type Result struct {
	InputSizeBytes  int64   `json:"inputSizeBytes"`
	OutputSizeBytes int64   `json:"outputSizeBytes"`
	SavedPercent    float64 `json:"savedPercent"`
	ElapsedSeconds  float64 `json:"elapsedSeconds"`
	AverageSpeed    float64 `json:"averageSpeed"`
	Output          Source  `json:"output"`
}

type Job struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectID"`
	Input       Source     `json:"input"`
	Output      Source     `json:"output"`
	State       string     `json:"state"`
	QueuedAt    time.Time  `json:"queuedAt"`
	StartedAt   time.Time  `json:"startedAt"`
	CompletedAt time.Time  `json:"completedAt"`
	Progress    Progress   `json:"progress"`
	Result      *Result    `json:"result,omitempty"`
	Error       string     `json:"error"`
	Logs        []string   `json:"logs"`
	Trim        *TimeRange `json:"trim,omitempty"`
	Project     *Project   `json:"project,omitempty"`
}

const (
	JobQueued      = "queued"
	JobPreparing   = "preparing"
	JobRunning     = "running"
	JobCancelling  = "cancelling"
	JobCancelled   = "cancelled"
	JobCompleted   = "completed"
	JobFailed      = "failed"
	JobInterrupted = "interrupted"
)
