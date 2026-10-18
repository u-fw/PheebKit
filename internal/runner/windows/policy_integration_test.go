package windows

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"pheebkit/internal/core"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIntegrationLocalInputNeverFollowsHTTPPlaylist(t *testing.T) {
	if os.Getenv("PHEEBKIT_INTEGRATION") != "1" {
		t.Skip("requires installed FFprobe")
	}
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1); http.Error(w, "unexpected", 404) }))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "local.m3u8")
	body := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\n" + server.URL + "/segment.ts\n#EXT-X-ENDLIST\n"
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := New(os.Getenv("PHEEBKIT_FFMPEG"), os.Getenv("PHEEBKIT_FFPROBE")).Probe(ctx, core.Source{URI: path, Kind: core.SourceLocalFile})
	if err == nil || !strings.Contains(err.Error(), "whitelist 'file'") {
		t.Fatal("input protocol policy not enforced", err)
	}
	if hits.Load() != 0 {
		t.Fatal("probing a local playlist contacted HTTP")
	}
}
