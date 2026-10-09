package capabilities

import (
	"testing"

	"kompeg/internal/core"
)

func TestAutomaticSelectionRequiresRuntimeQualificationAndDepth(t *testing.T) {
	caps := core.Capabilities{Encoders: []core.Encoder{{Name: "hevc_nvenc"}, {Name: "libx265"}}}
	selected, format, reason, err := SelectVideoEncoder(caps, "hevc", "auto", false)
	if err != nil || selected.Name != "libx265" || format != "yuv420p" || reason == "" {
		t.Fatalf("an untested GPU wrapper was trusted: %+v %s %s %v", selected, format, reason, err)
	}
	caps.VideoEncoders = []core.VideoEncoder{
		{Name: "hevc_nvenc", Codec: "hevc", Kind: "gpu", Available: true, PixelFormats: []string{"nv12"}},
		{Name: "libx265", Codec: "hevc", Kind: "cpu", Available: true},
	}
	selected, format, _, err = SelectVideoEncoder(caps, "hevc", "auto", false)
	if err != nil || selected.Name != "hevc_nvenc" || format != "nv12" {
		t.Fatalf("qualified GPU was not preferred: %+v %s %v", selected, format, err)
	}
	selected, format, reason, err = SelectVideoEncoder(caps, "hevc", "auto", true)
	if err != nil || selected.Name != "libx265" || format != "yuv420p10le" || reason == "" {
		t.Fatalf("10-bit input was silently downgraded on GPU: %+v %s %s %v", selected, format, reason, err)
	}
	if _, _, _, err := SelectVideoEncoder(caps, "hevc", "hevc_nvenc", true); err == nil {
		t.Fatal("explicit unqualified depth was accepted")
	}
	caps.VideoEncoders[0].PixelFormats = append(caps.VideoEncoders[0].PixelFormats, "p010le")
	selected, format, _, err = SelectVideoEncoder(caps, "hevc", "auto", true)
	if err != nil || selected.Name != "hevc_nvenc" || format != "p010le" {
		t.Fatalf("qualified 10-bit GPU was not selected: %+v %s %v", selected, format, err)
	}
}

func TestExplicitAndLegacyCPUSelectionIsPreserved(t *testing.T) {
	caps := core.Capabilities{VideoEncoders: []core.VideoEncoder{
		{Name: "hevc_nvenc", Codec: "hevc", Kind: "gpu", Available: true, PixelFormats: []string{"nv12"}},
		{Name: "libx265", Codec: "hevc", Kind: "cpu", Available: true},
	}}
	for _, requested := range []string{"", "cpu"} {
		selected, _, _, err := SelectVideoEncoder(caps, "hevc", requested, false)
		if err != nil || selected.Name != "libx265" {
			t.Fatalf("%q selection changed to GPU: %+v %v", requested, selected, err)
		}
	}
	if _, _, _, err := SelectVideoEncoder(caps, "h264", "hevc_nvenc", false); err == nil {
		t.Fatal("encoder for another codec was accepted")
	}
}
