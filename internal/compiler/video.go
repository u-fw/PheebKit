package compiler

import (
	"fmt"
	"strconv"
	"strings"

	"kompeg/internal/core"
)

// VideoArguments is shared with the runner's real GPU smoke tests, so a device
// is only offered after the actual preset and rate-control policy can encode.
func VideoArguments(encoder string, settings core.Settings, bitrate int) ([]string, error) {
	args := []string{"-c:v", encoder}
	if policy, ok := encoderPolicies[encoder]; ok {
		preset, ok := policy.Presets[settings.Preset]
		if !ok {
			return nil, fmt.Errorf("preset %q is unsupported for encoder %s", settings.Preset, encoder)
		}
		args = append(args, policy.Option, preset)
	} else {
		preset := ""
		switch {
		case strings.HasSuffix(encoder, "_nvenc"):
			preset = map[string]string{"fast": "p3", "medium": "p5", "slow": "p7"}[settings.Preset]
		case strings.HasSuffix(encoder, "_qsv"):
			preset = map[string]string{"fast": "fast", "medium": "medium", "slow": "slow"}[settings.Preset]
		case strings.HasSuffix(encoder, "_amf"):
			preset = map[string]string{"fast": "speed", "medium": "balanced", "slow": "quality"}[settings.Preset]
		default:
			return nil, fmt.Errorf("unsupported video encoder %q", encoder)
		}
		if preset == "" {
			return nil, fmt.Errorf("GPU 编码器 %s 支持 fast、medium、slow 预设", encoder)
		}
		if strings.HasSuffix(encoder, "_amf") {
			args = append(args, "-quality", preset)
		} else {
			args = append(args, "-preset", preset)
		}
	}
	if settings.Mode != "quality" {
		if strings.HasSuffix(encoder, "_nvenc") {
			args = append(args, "-rc", "vbr")
		} else if strings.HasSuffix(encoder, "_amf") {
			args = append(args, "-rc", "vbr_peak", "-maxrate", strconv.Itoa(bitrate*2)+"k")
		}
		return append(args, "-b:v", strconv.Itoa(bitrate)+"k"), nil
	}
	maxQuality := 51
	if settings.Codec == "av1" && !strings.HasSuffix(encoder, "_qsv") {
		maxQuality = 63
	}
	minQuality := 0
	if strings.HasSuffix(encoder, "_qsv") || strings.HasSuffix(encoder, "_nvenc") {
		// NVENC CQ 0 means automatic rather than best quality. Keep the UI's
		// "lower is better" promise by rejecting that ambiguous value.
		minQuality = 1
	}
	if settings.Quality < minQuality || settings.Quality > maxQuality {
		return nil, fmt.Errorf("%s quality must be between %d and %d", encoder, minQuality, maxQuality)
	}
	quality := strconv.Itoa(settings.Quality)
	switch {
	case strings.HasSuffix(encoder, "_nvenc"):
		return append(args, "-rc", "vbr", "-cq", quality, "-b:v", "0"), nil
	case strings.HasSuffix(encoder, "_qsv"):
		return append(args, "-global_quality", quality), nil
	case strings.HasSuffix(encoder, "_amf"):
		if settings.Codec == "av1" {
			// AV1 uses a 0..255 QP scale; expose the same 0..63 UI range as
			// other AV1 encoders, mapped proportionally rather than as CRF.
			quality = strconv.Itoa(int(float64(settings.Quality) * 255 / 63))
		}
		args = append(args, "-rc", "cqp", "-qp_i", quality, "-qp_p", quality)
		if settings.Codec == "h264" || settings.Codec == "av1" {
			args = append(args, "-qp_b", quality)
		}
		return args, nil
	default:
		args = append(args, "-crf", quality)
		if encoder == "libaom-av1" {
			args = append(args, "-b:v", "0")
		}
		return args, nil
	}
}

func validateEncoderRaw(args []string, encoder string) error {
	owners := map[string]string{"-x264-params": "libx264", "-x265-params": "libx265", "-svtav1-params": "libsvtav1"}
	for index := 0; index < len(args); index += 2 {
		if owner, found := owners[args[index]]; found && owner != encoder {
			return fmt.Errorf("高级参数 %s 仅适用于 %s；请选择对应编码的 CPU 编码器，或移除该参数。", args[index], owner)
		}
	}
	return nil
}
