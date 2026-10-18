package probe

import (
	"math"
	"testing"

	"pheebkit/internal/core"
)

func TestParsePreservesTrackMetadataAndRationalFrameRate(t *testing.T) {
	data := []byte(`{"format":{"format_name":"matroska,webm","duration":"6151.5","size":"102400","bit_rate":"120000"},"streams":[{"index":0,"codec_type":"video","codec_name":"hevc","width":3840,"height":2160,"avg_frame_rate":"60000/1001","pix_fmt":"yuv420p10le","color_transfer":"smpte2084","color_primaries":"bt2020","color_space":"bt2020nc"},{"index":1,"codec_type":"audio","codec_name":"aac","tags":{"language":"jpn","title":"Japanese"}},{"index":2,"codec_type":"subtitle","codec_name":"subrip","tags":{"language":"eng"}}]}`)
	media, err := Parse(data, core.Source{URI: "movie 中文.mkv", DisplayName: "movie 中文.mkv", Kind: core.SourceLocalFile})
	if err != nil {
		t.Fatal(err)
	}
	if media.Name != "movie 中文.mkv" || len(media.Streams) != 3 || media.SizeBytes != 102400 {
		t.Fatalf("unexpected media: %+v", media)
	}
	video := media.Streams[0]
	if math.Abs(video.FrameRate-59.94005994) > 0.00001 || video.BitDepth != 10 || video.HDR != "PQ / HDR10" {
		t.Fatalf("lost video precision/metadata: %+v", video)
	}
	if media.Streams[1].Language != "jpn" || media.Streams[1].Title != "Japanese" || media.Streams[2].CodecType != "subtitle" {
		t.Fatalf("lost track metadata: %+v", media.Streams)
	}
}

func TestDurationFallbackAndInvalidJSON(t *testing.T) {
	media, err := Parse([]byte(`{"streams":[{"codec_type":"video","duration":"1.0","avg_frame_rate":"0/0"},{"codec_type":"audio","duration":"2.5"}]}`), core.Source{})
	if err != nil || media.DurationSeconds != 2.5 || media.Streams[0].FrameRate != 0 {
		t.Fatalf("fallback = %+v, %v", media, err)
	}
	for _, raw := range []string{`invalid`, `{}`, `{"streams":[]}`} {
		if _, err := Parse([]byte(raw), core.Source{}); err == nil {
			t.Fatalf("accepted invalid probe document: %s", raw)
		}
	}
}

func TestProbePreservesNonMovieVideoDispositions(t *testing.T) {
	media, err := Parse([]byte(`{"streams":[{"index":0,"codec_type":"video","codec_name":"mjpeg","disposition":{"attached_pic":1}},{"index":1,"codec_type":"video","codec_name":"mjpeg","disposition":{"timed_thumbnails":1}},{"index":2,"codec_type":"video","codec_name":"h264"}]}`), core.Source{})
	if err != nil {
		t.Fatal(err)
	}
	if !media.Streams[0].AttachedPicture || !media.Streams[1].TimedThumbnails || media.Streams[2].AttachedPicture || media.Streams[2].TimedThumbnails {
		t.Fatalf("lost stream dispositions: %+v", media.Streams)
	}
}

func TestDisplayRotationAndMP4TrackName(t *testing.T) {
	media, err := Parse([]byte(`{"streams":[{"index":0,"codec_type":"video","width":1920,"height":1080,"tags":{"rotate":"180"},"side_data_list":[{"side_data_type":"Display Matrix","rotation":-90}]},{"index":1,"codec_type":"audio","tags":{"handler_name":"Main audio"}},{"index":2,"codec_type":"subtitle","tags":{"title":"preferred","handler_name":"fallback"}}]}`), core.Source{})
	if err != nil || media.Streams[0].DisplayRotation != 270 || media.Streams[0].Width != 1920 || media.Streams[1].Title != "Main audio" || media.Streams[2].Title != "preferred" {
		t.Fatalf("lost normalized rotation or MP4 track name: %+v %v", media, err)
	}
	for _, test := range []struct {
		input string
		want  float64
	}{{"-450", 270}, {"720", 0}, {"89.999999", 90}, {"45", 45}, {"NaN", 0}, {"Inf", 0}, {"invalid", 0}} {
		media, err := Parse([]byte(`{"streams":[{"codec_type":"video","tags":{"rotate":"`+test.input+`"}}]}`), core.Source{})
		if err != nil || media.Streams[0].DisplayRotation != test.want {
			t.Fatalf("rotation %q => %+v %v", test.input, media, err)
		}
	}
}
