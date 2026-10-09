package windows

import (
	"bufio"
	"io"
	"math"
	"strconv"
	"strings"

	"ffmpegstudio/internal/core"
)

func readProgress(reader io.Reader, duration float64, emit func(core.Progress)) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	values := make(map[string]string)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[parts[0]] = parts[1]
		if parts[0] == "progress" {
			if emit != nil {
				emit(parseProgress(values, duration))
			}
			clear(values)
		}
	}
	if err := scanner.Err(); err != nil {
		_, _ = io.Copy(io.Discard, reader)
		return err
	}
	return nil
}

func parseProgress(values map[string]string, duration float64) core.Progress {
	progress := core.Progress{FPS: number(values["fps"]), Speed: number(strings.TrimSuffix(values["speed"], "x")),
		TotalTimeSeconds: duration, Frame: int64(number(values["frame"])),
		OutputSizeBytes: int64(number(values["total_size"])), Finished: values["progress"] == "end"}
	if raw, ok := values["out_time_us"]; ok {
		progress.CurrentTimeSeconds = number(raw) / 1_000_000
	} else if raw, ok := values["out_time_ms"]; ok {
		// FFmpeg's historical out_time_ms field contains microseconds.
		progress.CurrentTimeSeconds = number(raw) / 1_000_000
	} else {
		parts := strings.Split(values["out_time"], ":")
		if len(parts) == 3 {
			progress.CurrentTimeSeconds = number(parts[0])*3600 + number(parts[1])*60 + number(parts[2])
		}
	}
	if duration > 0 {
		progress.Percent = math.Min(100, progress.CurrentTimeSeconds/duration*100)
	}
	// progress=end describes the encoder stream, not successful process exit.
	// Completion is decided only after Wait and output inspection succeed.
	return progress
}

func number(raw string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func readLogs(reader io.Reader, emit func(string)) (string, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	tail := make([]string, 0, 24)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 4096 {
			line = line[:4096] + "…"
		}
		if len(tail) == 24 {
			copy(tail, tail[1:])
			tail = tail[:23]
		}
		tail = append(tail, line)
		if emit != nil {
			emit(line)
		}
	}
	if err := scanner.Err(); err != nil {
		_, _ = io.Copy(io.Discard, reader)
		return strings.Join(tail, "\n"), err
	}
	return strings.Join(tail, "\n"), nil
}
