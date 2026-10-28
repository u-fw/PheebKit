package app

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/sourcefile"
)

func TestMediaPreviewTokensUseLruAndExplicitRelease(t *testing.T) {
	service := NewService()
	defer service.Close()
	directory := t.TempDir()
	handler := service.AssetHandler(http.NotFoundHandler())
	urls := make([]string, previewTokenLimit+1)
	paths := make([]string, len(urls))
	for index := range urls {
		path := filepath.Join(directory, fmt.Sprintf("preview-%02d.mp4", index))
		if err := os.WriteFile(path, []byte("0123456789"), 0600); err != nil {
			t.Fatal(err)
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
		urls[index] = url
		paths[index] = path
	}
	if len(service.previewTokens) != previewTokenLimit {
		t.Fatalf("preview token cache has %d entries", len(service.previewTokens))
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, urls[0], nil))
	if response.Code != http.StatusNotFound {
		t.Fatal("least recently used preview token was not evicted")
	}

	leased, err := service.GetMediaPreviewURL(paths[len(paths)-1])
	if err != nil || leased != urls[len(urls)-1] {
		t.Fatalf("same source did not acquire another reference: %s %v", leased, err)
	}
	service.ReleaseMediaPreviewURL(leased)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodHead, leased, nil))
	if response.Code != http.StatusOK {
		t.Fatal("releasing one shared reference revoked an active preview")
	}
	service.ReleaseMediaPreviewURL(leased)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, leased, nil))
	if response.Code != http.StatusNotFound {
		t.Fatal("released preview token remained valid")
	}
}

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
	for range 70 {
		reopened, err := service.GetMediaPreviewURL(path)
		if err != nil || reopened != url {
			t.Fatalf("reopening unchanged media revoked the active preview: %s %v", reopened, err)
		}
	}
	handler := service.AssetHandler(http.NotFoundHandler())
	request := httptest.NewRequest("GET", url, nil)
	request.Header.Set("Range", "bytes=2-5")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 206 || response.Body.String() != "2345" {
		t.Fatalf("range failed: %d %s", response.Code, response.Body.String())
	}
	assertAssetSecurityHeaders(t, response.Header())
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

func TestAssetHandlerAppliesSecurityHeadersToStaticAndMediaResponses(t *testing.T) {
	service := NewService()
	defer service.Close()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><html></html>"))
	})
	handler := service.AssetHandler(next)

	for _, test := range []struct {
		path string
		code int
	}{
		{path: "/", code: http.StatusOK},
		{path: "/__media/unknown", code: http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.code {
			t.Fatalf("%s returned %d, want %d", test.path, response.Code, test.code)
		}
		assertAssetSecurityHeaders(t, response.Header())
	}
}

func assertAssetSecurityHeaders(t *testing.T, header http.Header) {
	t.Helper()
	for name, want := range map[string]string{
		"Content-Security-Policy":      assetContentSecurityPolicy,
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), microphone=(), geolocation=(), payment=(), usb=(), serial=(), hid=()",
		"Referrer-Policy":              "no-referrer",
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
	} {
		if got := header.Get(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestVideoRangesStayBoundedForWindowsResponseBuffer(t *testing.T) {
	if _, err := boundedMediaRange("", 10*previewChunkBytes); err == nil {
		t.Fatal("unbounded large GET allowed")
	}
	for _, header := range []string{"bytes=0-", "bytes=0-99999999999", "bytes=-99999999999"} {
		got, err := boundedMediaRange(header, 10*previewChunkBytes)
		if err != nil || got != "bytes=0-2097151" {
			t.Fatalf("unbounded range %q: %s %v", header, got, err)
		}
	}
	for _, header := range []string{"bytes=1-2,4-6", "bytes=-0", "bytes=9-2", "bytes=99999999999999999999-", "bytes=99999999-"} {
		if _, err := boundedMediaRange(header, 10*previewChunkBytes); err == nil {
			t.Fatalf("invalid range accepted: %q", header)
		}
	}
	if got, err := boundedMediaRange("bytes=-10", 100); err != nil || got != "bytes=90-99" {
		t.Fatal("suffix range changed", got, err)
	}
	service := NewService()
	defer service.Close()
	path := filepath.Join(t.TempDir(), "large.mp4")
	if err := os.WriteFile(path, bytes.Repeat([]byte{1}, int(previewChunkBytes+32)), 0600); err != nil {
		t.Fatal(err)
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
	request.Header.Set("Range", "bytes=0-")
	request.Header.Set("If-Range", "an-old-etag")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 206 || int64(response.Body.Len()) != previewChunkBytes {
		t.Fatal("large response or If-Range bypassed bound", response.Code, response.Body.Len())
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", url, nil))
	if response.Code != 416 {
		t.Fatal("unbounded large GET accepted")
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("HEAD", url, nil))
	if response.Code != 200 || response.Body.Len() != 0 {
		t.Fatal("HEAD read the video body")
	}
}

func TestRealVideoPositionFrames(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("set PHEEBKIT_INTEGRATION=1")
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
