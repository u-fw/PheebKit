package app

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"time"

	"pheebkit/internal/ffmpegpolicy"
	"pheebkit/internal/processguard"
	"pheebkit/internal/sourcefile"
)

const waveformRate = 4000
const waveformBins = 512
const waveformMaxSeconds = 120

var waveformGate = make(chan struct{}, 1)

type AudioWaveform struct {
	StartSeconds    float64   `json:"startSeconds"`
	DurationSeconds float64   `json:"durationSeconds"`
	StreamIndex     int       `json:"streamIndex"`
	Peaks           []float64 `json:"peaks"`
}

// GetAudioWaveform decodes a bounded, explicitly requested source window. It
// does not cache entire tracks or send source bytes through renderer IPC.
func (s *Service) GetAudioWaveform(uri string, start, duration float64, streamIndex int) (AudioWaveform, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	media, err := s.importedMedia(uri)
	if err != nil {
		return AudioWaveform{}, err
	}
	if math.IsNaN(start) || math.IsInf(start, 0) || math.IsNaN(duration) || math.IsInf(duration, 0) || start < 0 || duration <= 0 || duration > waveformMaxSeconds || media.DurationSeconds <= 0 || start >= media.DurationSeconds || duration > media.DurationSeconds-start+1e-6 {
		return AudioWaveform{}, errors.New("波形范围必须位于源媒体内，且每次不超过 120 秒。")
	}
	found := false
	for _, stream := range media.Streams {
		if stream.Index == streamIndex && stream.CodecType == "audio" {
			found = true
			break
		}
	}
	if !found {
		return AudioWaveform{}, errors.New("请选择源媒体中的有效音轨。")
	}
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	defer cancel()
	release, err := acquireThumbnailSlot(ctx, waveformGate)
	if err != nil {
		return AudioWaveform{}, err
	}
	defer release()
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-threads", "1", "-ss", strconv.FormatFloat(start, 'f', 6, 64)}
	args = append(args, ffmpegpolicy.LocalInput(uri)...)
	args = append(args, "-map", "0:"+strconv.Itoa(streamIndex), "-t", strconv.FormatFloat(duration, 'f', 6, 64), "-vn", "-sn", "-dn", "-ac", "1", "-ar", strconv.Itoa(waveformRate), "-c:a", "pcm_s16le", "-f", "s16le", "pipe:1")
	command := exec.CommandContext(ctx, previewBinary(s.runner.snapshot()), args...)
	var output, stderr limitedBuffer
	output.limit = waveformMaxSeconds*waveformRate*2 + 32
	stderr.limit = 32 * 1024
	command.Stdout = &output
	command.Stderr = &stderr
	if err := processguard.Run(command); err != nil {
		return AudioWaveform{}, fmt.Errorf("音频波形读取失败：%w：%s", err, stderr.String())
	}
	if err := sourcefile.Verify(uri, media.SourceFingerprint); err != nil {
		return AudioWaveform{}, err
	}
	peaks, err := waveformPeaks(output.Bytes())
	if err != nil {
		return AudioWaveform{}, err
	}
	return AudioWaveform{StartSeconds: start, DurationSeconds: math.Min(duration, float64(output.Len()/2)/waveformRate), StreamIndex: streamIndex, Peaks: peaks}, nil
}

func waveformPeaks(data []byte) ([]float64, error) {
	if len(data) < 2 || len(data)%2 != 0 || len(data) > waveformMaxSeconds*waveformRate*2+32 {
		return nil, errors.New("音轨没有可用的波形数据。")
	}
	samples := len(data) / 2
	count := min(samples, waveformBins)
	peaks := make([]float64, count)
	for i := range samples {
		value := math.Abs(float64(int16(binary.LittleEndian.Uint16(data[i*2:])))) / 32768
		index := i * count / samples
		peaks[index] = max(peaks[index], value)
	}
	return peaks, nil
}
