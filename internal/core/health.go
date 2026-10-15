package core

import "context"

// Recovery removes or replaces conservative failed decode windows, never source data.
type Recovery struct {
	Mode              string      `json:"mode"`
	SourceFingerprint string      `json:"sourceFingerprint"`
	DamagedRanges     []TimeRange `json:"damagedRanges"`
}

type MediaHealth struct {
	ID                string      `json:"id"`
	URI               string      `json:"uri"`
	SourceFingerprint string      `json:"sourceFingerprint"`
	State             string      `json:"state"`
	Percent           float64     `json:"percent"`
	DurationSeconds   float64     `json:"durationSeconds"`
	WindowSeconds     float64     `json:"windowSeconds"`
	DamagedRanges     []TimeRange `json:"damagedRanges"`
	Error             string      `json:"error"`
}

type MediaHealthScanner interface {
	ScanHealth(context.Context, Media, int, func(MediaHealth)) (MediaHealth, error)
}
