// Package capabilities resolves codec families against a scanned FFmpeg build.
package capabilities

import (
	"strings"

	"ffmpegstudio/internal/core"
)

type Definition struct {
	ID         string
	Label      string
	Candidates []string
}

var Definitions = []Definition{
	{ID: "h264", Label: "H.264", Candidates: []string{"libx264"}},
	{ID: "hevc", Label: "H.265 / HEVC", Candidates: []string{"libx265"}},
	{ID: "av1", Label: "AV1", Candidates: []string{"libsvtav1", "libaom-av1"}},
}

func ParseEncoders(text string) []core.Encoder {
	result := make([]core.Encoder, 0)
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || len(fields[0]) != 6 || fields[1] == "=" {
			continue
		}
		mediaType := ""
		switch fields[0][0] {
		case 'V':
			mediaType = "video"
		case 'A':
			mediaType = "audio"
		case 'S':
			mediaType = "subtitle"
		default:
			continue
		}
		result = append(result, core.Encoder{Name: fields[1], Description: strings.Join(fields[2:], " "), MediaType: mediaType})
	}
	return result
}

func Resolve(encoders []core.Encoder) []core.Codec {
	result := make([]core.Codec, 0, len(Definitions))
	for _, definition := range Definitions {
		codec := core.Codec{ID: definition.ID, Label: definition.Label}
		for _, candidate := range definition.Candidates {
			if HasEncoder(encoders, candidate) {
				codec.Encoder = candidate
				codec.Available = true
				break
			}
		}
		result = append(result, codec)
	}
	return result
}

func HasEncoder(encoders []core.Encoder, name string) bool {
	for _, encoder := range encoders {
		if encoder.Name == name {
			return true
		}
	}
	return false
}
