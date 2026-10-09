package capabilities

import (
	"fmt"
	"strings"

	"kompeg/internal/core"
)

// Order is deterministic: prefer a qualified discrete GPU, then Intel QSV,
// then software. An encoder's presence in -encoders never qualifies a GPU.
var GPUDefinitions = []core.VideoEncoder{
	{Name: "h264_nvenc", Codec: "h264", Label: "NVIDIA NVENC", Kind: "gpu"},
	{Name: "hevc_nvenc", Codec: "hevc", Label: "NVIDIA NVENC", Kind: "gpu"},
	{Name: "av1_nvenc", Codec: "av1", Label: "NVIDIA NVENC", Kind: "gpu"},
	{Name: "h264_amf", Codec: "h264", Label: "AMD AMF", Kind: "gpu"},
	{Name: "hevc_amf", Codec: "hevc", Label: "AMD AMF", Kind: "gpu"},
	{Name: "av1_amf", Codec: "av1", Label: "AMD AMF", Kind: "gpu"},
	{Name: "h264_qsv", Codec: "h264", Label: "Intel Quick Sync", Kind: "gpu"},
	{Name: "hevc_qsv", Codec: "hevc", Label: "Intel Quick Sync", Kind: "gpu"},
	{Name: "av1_qsv", Codec: "av1", Label: "Intel Quick Sync", Kind: "gpu"},
}

func SoftwareVideoEncoders(encoders []core.Encoder) []core.VideoEncoder {
	result := make([]core.VideoEncoder, 0, 4)
	for _, definition := range Definitions {
		for _, name := range definition.Candidates {
			if HasEncoder(encoders, name) {
				result = append(result, core.VideoEncoder{Name: name, Codec: definition.ID,
					Label: "CPU · " + name, Kind: "cpu", Available: true, PixelFormats: []string{"yuv420p", "yuv420p10le"}})
			}
		}
	}
	return result
}

func ResolveQualified(encoders []core.VideoEncoder) []core.Codec {
	result := make([]core.Codec, 0, len(Definitions))
	for _, definition := range Definitions {
		codec := core.Codec{ID: definition.ID, Label: definition.Label}
		for _, encoder := range encoders {
			if encoder.Codec == definition.ID && encoder.Available {
				codec.Available, codec.Encoder = true, encoder.Name
				break
			}
		}
		result = append(result, codec)
	}
	return result
}

// SelectVideoEncoder preserves legacy projects with an empty selector as CPU.
// New projects explicitly choose auto, so the changed default is reviewable.
func SelectVideoEncoder(caps core.Capabilities, codec, requested string, tenBit bool) (core.VideoEncoder, string, string, error) {
	encoders := caps.VideoEncoders
	if len(encoders) == 0 {
		encoders = SoftwareVideoEncoders(caps.Encoders)
	}
	if requested == "" {
		requested = "cpu"
	}
	formatFor := func(encoder core.VideoEncoder) string {
		if encoder.Kind == "gpu" {
			if tenBit {
				return "p010le"
			}
			return "nv12"
		}
		if tenBit {
			return "yuv420p10le"
		}
		return "yuv420p"
	}
	for _, encoder := range encoders {
		if encoder.Codec != codec || !encoder.Available || (requested == "cpu" && encoder.Kind != "cpu") ||
			(requested != "auto" && requested != "cpu" && requested != encoder.Name) {
			continue
		}
		format := formatFor(encoder)
		if encoder.Kind == "gpu" && !hasFormat(encoder.PixelFormats, format) {
			continue
		}
		reason := ""
		if requested == "auto" && encoder.Kind == "cpu" {
			reason = "当前编码和位深没有通过实测的 GPU 编码器，自动使用 CPU。"
		}
		return encoder, format, reason, nil
	}
	if requested != "auto" && requested != "cpu" {
		for _, encoder := range encoders {
			if encoder.Name == requested && encoder.Codec == codec {
				reason := encoder.Reason
				if encoder.Available {
					reason = "该 GPU 未通过当前视频位深的编码测试"
				}
				return core.VideoEncoder{}, "", "", fmt.Errorf("GPU 编码器 %s 不可用：%s", encoder.Label, reason)
			}
		}
	}
	return core.VideoEncoder{}, "", "", fmt.Errorf("no supported %s encoder available for %q in this FFmpeg build", requested, codec)
}

func hasFormat(formats []string, format string) bool {
	for _, available := range formats {
		if strings.EqualFold(available, format) {
			return true
		}
	}
	return false
}
