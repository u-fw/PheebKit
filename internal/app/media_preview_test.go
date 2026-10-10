package app

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kompeg/internal/core"
	"kompeg/internal/sourcefile"
)

func TestMediaPreviewRangeAndAccessBoundary(t *testing.T) {
	service := NewService()
	defer service.Close()
	path := filepath.Join(t.TempDir(), "媒体.mp4")
	os.WriteFile(path, []byte("0123456789"), 0600)
	if _, err := service.GetMediaPreviewURL(path); err == nil {
		t.Fatal("unimported path accepted")
	}
	fingerprint, err := sourcefile.Fingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	service.media[path] = core.Media{Source: source(path), SourceFingerprint: fingerprint}
	url, err := service.GetMediaPreviewURL(path)
	if err != nil {
		t.Fatal(err)
	}
	handler := service.AssetHandler(http.NotFoundHandler())
	request := httptest.NewRequest("GET", url, nil)
	request.Header.Set("Range", "bytes=2-5")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 206 || response.Body.String() != "2345" {
		t.Fatalf("range failed: %d %s", response.Code, response.Body.String())
	}
	for _, route := range []string{"/__media/../../secret", "/__media/unknown"} {
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", route, nil))
		if response.Code != 404 {
			t.Fatal("unknown path disclosed")
		}
	}
	os.WriteFile(path, []byte("replaced"), 0600)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", url, nil))
	if response.Code != 409 {
		t.Fatal("changed source served")
	}
	if err := service.OpenJobOutput("unknown", false); err == nil {
		t.Fatal("unknown output accepted")
	}
	service.jobs.Restore([]core.Job{
		{ID: "unsafe", State: core.JobCompleted, Output: source(path + ".exe")},
		{ID: "missing", State: core.JobCompleted, Output: source(path + "-missing.mp4")},
	}, nil)
	for _, id := range []string{"unsafe", "missing"} {
		if err := service.OpenJobOutput(id, false); err == nil {
			t.Fatal("unsafe/missing output accepted")
		}
	}
}

func TestRealVideoPositionFrames(t *testing.T) {
	if os.Getenv("KOMPEG_INTEGRATION") != "1" {
		t.Skip("set KOMPEG_INTEGRATION=1")
	}
	service := NewService()
	defer service.Close()
	environment := discoverEnvironment(binaryConfig{})
	path := filepath.Join(t.TempDir(), "时间位置.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, environment.FFmpeg.Path, "-hide_banner", "-loglevel", "error", "-nostdin", "-f", "lavfi", "-i", "color=c=red:s=160x90:r=10:d=1", "-f", "lavfi", "-i", "color=c=blue:s=160x90:r=10:d=1", "-filter_complex", "[0:v][1:v]concat=n=2:v=1:a=0[v]", "-map", "[v]", "-c:v", "libx264", "-pix_fmt", "yuv420p", path)
	hidePlatformProcess(command)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v %s", err, output)
	}
	media, err := service.Probe(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, position := range []float64{0.4, 1.4, media.DurationSeconds} {
		frame, err := service.GetVideoFrame(path, position)
		if err != nil {
			t.Fatal(err)
		}
		assertThumbnailColour(t, frame, position < 1)
	}
	for _, invalid := range []float64{-1, 3, math.NaN(), math.Inf(1)} {
		if _, err := service.GetVideoFrame(path, invalid); err == nil {
			t.Fatal("invalid position accepted")
		}
	}
	project := core.Project{SchemaVersion: 1, ID: "finished", Inputs: []core.Source{media.Source}, Output: source(path + ".old.mp4"), Settings: core.DefaultSettings()}
	project.Settings.Trim = &core.TimeRange{StartSeconds: .3, EndSeconds: 1.4}
	service.jobs.Restore([]core.Job{{ID: "reuse", State: core.JobCompleted, Input: media.Source, Output: project.Output, Project: &project}}, nil)
	reused, err := service.ReuseJob("reuse")
	if err != nil {
		t.Fatal(err)
	}
	if reused.Project.Output.URI != "" || reused.Media.Source.URI != path || reused.Project.Settings.Trim.StartSeconds != .3 {
		t.Fatal("reuse lost intent or retained old output")
	}
	reused.Project.Settings.Trim.StartSeconds = .5
	stored, _ := service.jobs.Get("reuse")
	if stored.Project.Settings.Trim.StartSeconds != .3 {
		t.Fatal("reused caller mutated saved intent")
	}
}
