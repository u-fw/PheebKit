package compiler

import (
	"fmt"
	"math"

	"pheebkit/internal/core"
)

func outputDuration(trim *core.TimeRange, total float64) (float64, error) {
	if trim == nil {
		return total, nil
	}
	finite := func(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
	if !finite(total) || total <= 0 {
		return 0, fmt.Errorf("trimming requires a known positive source duration")
	}
	if !finite(trim.StartSeconds) || !finite(trim.EndSeconds) || trim.StartSeconds < 0 || trim.EndSeconds <= trim.StartSeconds || trim.EndSeconds > total {
		return 0, fmt.Errorf("trim range must satisfy 0 <= start < end <= source duration (%.3f seconds)", total)
	}
	return trim.EndSeconds - trim.StartSeconds, nil
}
