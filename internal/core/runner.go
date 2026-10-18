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
	JobID                  string             `json:"jobID"`
	ProjectID              string             `json:"projectID"`
	Input                  Source             `json:"input"`
	SourceFingerprint      string             `json:"sourceFingerprint,omitempty"`
	Output                 Source             `json:"output"`
	Args                   []string           `json:"args" ts_type:"string[] | null"`
	Preview                string             `json:"preview"`
	DurationSeconds        float64            `json:"durationSeconds"`
	InputSizeBytes         int64              `json:"inputSizeBytes"`
	Encoder                string             `json:"encoder"`
	EncoderKind            string             `json:"encoderKind"`
	EncoderLabel           string             `json:"encoderLabel"`
	EncoderReason          string             `json:"encoderReason"`
	Trim                   *TimeRange         `json:"trim,omitempty" ts_type:"TimeRange | null"`
	SubtitleAsset          *SubtitleAsset     `json:"subtitleAsset,omitempty" ts_type:"SubtitleAsset | null"`
	FullDecodeVerification bool               `json:"fullDecodeVerification,omitempty"`
	ExpectedOutput         *OutputExpectation `json:"expectedOutput,omitempty" ts_type:"OutputExpectation | null"`
}

type SubtitleAsset struct {
	Source      Source `json:"source"`
	Fingerprint string `json:"fingerprint"`
	StreamIndex int    `json:"streamIndex"`
	Format      string `json:"format"`
}

type CatalogEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Flags       string `json:"flags"`
}
type FFmpegInspector interface {
	Catalog(context.Context, string) ([]CatalogEntry, error)
	OptionHelp(context.Context, string, string) (string, error)
}

// OutputExpectation is compiler intent, checked before publishing a result.
type OutputExpectation struct {
	AudioSampleRate     int    `json:"audioSampleRate,omitempty"`
	AudioChannels       int    `json:"audioChannels,omitempty"`
	AudioSourceChannels []int  `json:"audioSourceChannels,omitempty" ts_type:"number[] | null"`
	VideoStreams        int    `json:"videoStreams"`
	AudioStreams        int    `json:"audioStreams"`
	SubtitleStreams     int    `json:"subtitleStreams"`
	VideoCodec          string `json:"videoCodec,omitempty"`
}

// Ordinary jobs check metadata/structure; recovery additionally requires a full decode.
type OutputVerification struct {
	FullDecodePassed bool     `json:"fullDecodePassed,omitempty"`
	DurationSeconds  float64  `json:"durationSeconds"`
	VideoStreams     int      `json:"videoStreams"`
	AudioStreams     int      `json:"audioStreams"`
	SubtitleStreams  int      `json:"subtitleStreams"`
	Warnings         []string `json:"warnings" ts_type:"string[] | null"`
}

type Capabilities struct {
	Version            string         `json:"version"`
	BuildConfiguration string         `json:"buildConfiguration"`
	Encoders           []Encoder      `json:"encoders" ts_type:"Encoder[] | null"`
	Codecs             []Codec        `json:"codecs" ts_type:"Codec[] | null"`
	Hardware           []string       `json:"hardware" ts_type:"string[] | null"`
	Platform           string         `json:"platform"`
	VideoEncoders      []VideoEncoder `json:"videoEncoders" ts_type:"VideoEncoder[] | null"`
	GPUs               []GPU          `json:"gpus" ts_type:"GPU[] | null"`
	Filters            []string       `json:"filters,omitempty" ts_type:"string[] | null"`
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
	PixelFormats []string `json:"pixelFormats" ts_type:"string[] | null"`
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
	InputSizeBytes  int64               `json:"inputSizeBytes"`
	OutputSizeBytes int64               `json:"outputSizeBytes"`
	SavedPercent    float64             `json:"savedPercent"`
	ElapsedSeconds  float64             `json:"elapsedSeconds"`
	AverageSpeed    float64             `json:"averageSpeed"`
	Output          Source              `json:"output"`
	Verification    *OutputVerification `json:"verification,omitempty" ts_type:"OutputVerification | null"`
}

type Job struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"projectID"`
	Input             Source     `json:"input"`
	SourceFingerprint string     `json:"sourceFingerprint,omitempty"`
	Output            Source     `json:"output"`
	State             string     `json:"state"`
	QueuedAt          time.Time  `json:"queuedAt" ts_type:"string"`
	StartedAt         time.Time  `json:"startedAt" ts_type:"string"`
	CompletedAt       time.Time  `json:"completedAt" ts_type:"string"`
	Progress          Progress   `json:"progress"`
	Result            *Result    `json:"result,omitempty" ts_type:"Result | null"`
	Error             string     `json:"error"`
	Logs              []string   `json:"logs" ts_type:"string[] | null"`
	Trim              *TimeRange `json:"trim,omitempty" ts_type:"TimeRange | null"`
	Project           *Project   `json:"project,omitempty" ts_type:"Project | null"`
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
