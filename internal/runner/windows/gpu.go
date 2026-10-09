package windows

import (
	"context"
	"strings"
	"sync"
	"time"

	"kompeg/internal/capabilities"
	"kompeg/internal/compiler"
	"kompeg/internal/core"
)

func (r *Runner) qualifyVideoEncoders(ctx context.Context, encoders []core.Encoder) []core.VideoEncoder {
	result := make([]core.VideoEncoder, len(capabilities.GPUDefinitions))
	var workers sync.WaitGroup
	// Bound startup cost and avoid occupying all sessions on a multi-GPU host.
	semaphore := make(chan struct{}, 3)
	for index, definition := range capabilities.GPUDefinitions {
		result[index] = definition
		result[index].PixelFormats = []string{}
		if !capabilities.HasEncoder(encoders, definition.Name) {
			result[index].Reason = "当前 FFmpeg 未包含这个 GPU 编码器。"
			continue
		}
		workers.Add(1)
		go func(index int, definition core.VideoEncoder) {
			defer workers.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				result[index].Reason = "GPU 编码测试已取消。"
				return
			}
			trialCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()
			formats := []string{"nv12"}
			if definition.Codec != "h264" {
				formats = append(formats, "p010le")
			}
			for _, format := range formats {
				var failure error
				for _, mode := range []string{"quality", "bitrate"} {
					settings := core.Settings{Codec: definition.Codec, Preset: "medium", Mode: mode, Quality: 26}
					videoArgs, err := compiler.VideoArguments(definition.Name, settings, 1000)
					if err == nil {
						args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-f", "lavfi", "-i", "testsrc2=size=320x240:rate=30", "-frames:v", "3", "-an"}
						args = append(args, videoArgs...)
						// The leading '+' forbids FFmpeg from silently choosing
						// an 8-bit alternative. An explicit scale filter performs
						// conversion, since strict pixel format disables auto filters.
						args = append(args, "-vf", "scale=320:240,format="+format, "-pix_fmt", "+"+format, "-f", "null", "-")
						_, err = capture(trialCtx, r.FFmpeg, args, false)
					}
					if err != nil {
						failure = err
						break
					}
				}
				if failure == nil {
					result[index].PixelFormats = append(result[index].PixelFormats, format)
				} else if format == "nv12" {
					result[index].Reason = gpuFailureReason(failure)
					// No usable basic encode session, so a second depth cannot
					// make this encoder eligible for the default selection.
					break
				} else {
					result[index].Reason = "8-bit 编码实测通过；10-bit 编码未通过。"
				}
			}
			result[index].Available = len(result[index].PixelFormats) > 0
			if result[index].Available && result[index].Reason == "" {
				result[index].Reason = "已通过本机质量模式与码率模式的实际 GPU 编码测试。"
			}
		}(index, definition)
	}
	workers.Wait()
	return append(result, capabilities.SoftwareVideoEncoders(encoders)...)
}

func gpuFailureReason(err error) string {
	if err == context.DeadlineExceeded {
		return "GPU 编码测试超时；检查显卡驱动后重新检测。"
	}
	message := strings.TrimSpace(err.Error())
	if index := strings.Index(message, "exit status"); index >= 0 {
		message = message[index:]
	}
	runes := []rune(message)
	if len(runes) > 320 {
		message = string(runes[:320]) + "…"
	}
	return "本机 GPU 编码实测未通过（硬件或驱动不支持）：" + message
}
