package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	wailsassets "github.com/wailsapp/wails/v2/pkg/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"pheebkit/internal/app"
)

type testRuntimeAssets struct{}

func (testRuntimeAssets) DesktopIPC() []byte {
	return []byte("window.WailsInvoke = () => {};")
}

func (testRuntimeAssets) WebsocketIPC() []byte {
	return nil
}

func (testRuntimeAssets) RuntimeDesktopJS() []byte {
	return []byte("window.runtime = {};")
}

func TestWailsAssetInjectionPreservesSecurityHeaders(t *testing.T) {
	service := app.NewService()
	defer service.Close()
	handler, err := wailsassets.NewAssetServer("", assetserver.Options{
		Assets: fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<!doctype html><html><head></head><body><div id=root></div></body></html>")},
		},
		Middleware: service.AssetHandler,
	}, false, nil, testRuntimeAssets{})
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `src="/wails/runtime.js"`) {
		t.Fatalf("Wails index processing failed: %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Security-Policy") == "" ||
		response.Header().Get("X-Frame-Options") != "DENY" ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("security headers were lost during Wails asset injection: %v", response.Header())
	}
}
