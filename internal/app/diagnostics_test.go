package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kompeg/internal/core"
)

func TestDiagnosticsNeverSerializeUntrustedFields(t *testing.T) {
	s, projects := batchService(t)
	s.ensureWorkspace()
	secret := `C:\Users\PrivateUser\private-film-token123.mp4 https://host.invalid/?password=token123`
	s.environment = Environment{Ready: true, FFmpeg: BinaryStatus{Path: secret, Version: "ffmpeg version 9.0.2 " + secret, Source: "PATH", Available: true}, FFprobe: BinaryStatus{Path: secret, Version: "ffprobe version 9.0.2 " + secret, Source: secret, Available: true}, Message: secret}
	s.caps = &core.Capabilities{GPUs: []core.GPU{{Name: secret, Vendor: "NVIDIA"}}, VideoEncoders: []core.VideoEncoder{{Name: "hevc_nvenc", Codec: "hevc", Kind: "gpu", Available: false, Reason: secret}, {Name: secret, Reason: secret}}}
	p := projects[0]
	p.ID = secret
	p.Settings.Codec = "hevc"
	p.Settings.RawArguments = []string{"-metadata", secret}
	s.jobs.Restore([]core.Job{{ID: secret, Input: core.Source{URI: secret, DisplayName: secret}, Output: core.Source{URI: secret}, Project: &p, State: core.JobFailed, Error: "verify output structure: " + secret, Logs: []string{secret}}}, nil)
	preview, err := s.GetDiagnosticPreview(DiagnosticOptions{IncludeRecentJobs: true, JobLimit: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"PrivateUser", "private-film", "token123", "https://host.invalid", "-metadata"} {
		if strings.Contains(preview.JSON, value) {
			t.Fatalf("diagnostic leaked %q: %s", value, preview.JSON)
		}
	}
	var report diagnosticReport
	if err := json.Unmarshal([]byte(preview.JSON), &report); err != nil {
		t.Fatal(err)
	}
	if report.FFmpeg.Release != "9.0.2" || report.FFprobe.Source != "other" || len(report.Encoders) != 1 || len(report.RecentJobs) != 1 || report.RecentJobs[0].Failure != "output-invalid" || report.GPUVendors[0] != "nvidia" {
		t.Fatalf("useful sanitized context missing: %+v", report)
	}
	data, err := s.diagnosticData(preview.Token)
	if err != nil || string(data) != preview.JSON {
		t.Fatal("saved snapshot differs from preview", err)
	}
	defaultView, err := s.GetDiagnosticPreview(DiagnosticOptions{})
	if err != nil || strings.Contains(defaultView.JSON, "recentJobs") {
		t.Fatal("job detail unexpectedly included by default", err)
	}
	if _, err := s.GetDiagnosticPreview(DiagnosticOptions{JobLimit: 51}); err == nil {
		t.Fatal("unbounded summary accepted")
	}
}

func TestDiagnosticsTokensExpireAndAreBounded(t *testing.T) {
	s, _ := batchService(t)
	first, err := s.GetDiagnosticPreview(DiagnosticOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := s.GetDiagnosticPreview(DiagnosticOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.diagnostics) != 4 {
		t.Fatal("diagnostic cache exceeded bound")
	}
	if _, err := s.diagnosticData(first.Token); err == nil {
		t.Fatal("oldest preview was not evicted")
	}
	for token, snapshot := range s.diagnostics {
		snapshot.created = time.Now().Add(-11 * time.Minute)
		s.diagnostics[token] = snapshot
		if _, err := s.diagnosticData(token); err == nil {
			t.Fatal("expired report exported")
		}
		break
	}
}

func TestDiagnosticSaveProtectsOwnedFilesAndPreservesExactPreview(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.json")
	target := filepath.Join(dir, "report.json")
	if err := os.WriteFile(source, []byte("private media/config"), 0600); err != nil {
		t.Fatal(err)
	}
	data := []byte("{\"schemaVersion\":1}\n")
	if err := writeDiagnosticAt(source, data, []string{source}); err == nil {
		t.Fatal("diagnostic replaced protected file")
	}
	alias := filepath.Join(dir, "alias.json")
	if err := os.Link(source, alias); err == nil {
		if err := writeDiagnosticAt(alias, data, []string{source}); err == nil {
			t.Fatal("diagnostic accepted protected alias")
		}
	}
	if err := writeDiagnosticAt(target, data, []string{source}); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(saved, data) {
		t.Fatal("saved report differs from reviewed bytes", err)
	}
	if err := writeDiagnosticAt(filepath.Join(dir, "bad.txt"), data, nil); err == nil {
		t.Fatal("non-JSON report allowed")
	}
	if err := writeDiagnosticAt(target, bytes.Repeat([]byte("x"), 512*1024+1), nil); err == nil {
		t.Fatal("oversized report allowed")
	}
}

func TestDiagnosticsDoNotWaitForGPUScan(t *testing.T) {
	s, _ := batchService(t)
	s.ensureWorkspace()
	s.capsMu.Lock()
	defer s.capsMu.Unlock()
	preview, err := s.GetDiagnosticPreview(DiagnosticOptions{})
	if err != nil || !strings.Contains(preview.JSON, `"capabilitiesStatus": "scan-in-progress"`) {
		t.Fatal("report blocked or claimed scan completion", err)
	}
}
