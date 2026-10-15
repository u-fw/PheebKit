package windows

import (
	"context"
	"errors"
	"testing"
)

func TestCatalogParsingSeparatesEntriesFromLegends(t *testing.T) {
	for _, test := range []struct{ category, raw, want string }{
		{"filters", "Filters:\n T.. = Timeline support\n .. subtitles V->V Render text\n TS scale V->V Scale\n", "scale,subtitles"},
		{"encoders", "Encoders:\n V....D libx264 AVC\n A..... aac AAC\n", "aac,libx264"},
		{"demuxers", "Formats:\n D.. = Demuxing supported\n D mov,mp4 QuickTime / MOV\n", "mov,mp4"},
		{"protocols", "Supported file protocols:\nInput:\n file\nOutput:\n file\n https\n", "file,https"},
	} {
		values := parseCatalog(test.category, test.raw)
		got := ""
		for i, v := range values {
			if i > 0 {
				got += ","
			}
			got += v.Name
		}
		if got != test.want {
			t.Fatalf("%s: %q", test.category, got)
		}
	}
}

func TestCatalogWaitRespectsCancelledContext(t *testing.T) {
	r := New("not-executed.exe", "not-executed.exe")
	r.inspectMu.Lock()
	defer r.inspectMu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.Catalog(ctx, "filters"); !errors.Is(err, context.Canceled) {
		t.Fatal("inspection ignored cancellation", err)
	}
	if _, err := r.OptionHelp(ctx, "filters", "unsafe;input=x"); err == nil {
		t.Fatal("unsafe name accepted")
	}
}
