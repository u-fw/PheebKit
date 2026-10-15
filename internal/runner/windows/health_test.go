package windows

import "testing"

func TestHealthDoesNotDeleteOnOperationalFailures(t *testing.T) {
	for _, text := range []string{"Unknown decoder 'h264'", "Decoder not found", "unsupported codec", "Permission denied", "Error submitting packet: Function not implemented", "No such file or directory", "Invalid argument", "Cannot allocate memory", "Error opening C:/corrupt-video.mp4: Invalid argument"} {
		if mediaDecodeFailure(text) {
			t.Fatalf("operational error became deletion advice: %s", text)
		}
	}
	for _, text := range []string{"corrupt decoded frame", "Error submitting packet to decoder: Invalid data found when processing input", "error while decoding MB 0 0", "Invalid NAL unit size"} {
		if !mediaDecodeFailure(text) {
			t.Fatalf("missed known corruption: %s", text)
		}
	}
}
