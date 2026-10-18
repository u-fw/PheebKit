package app

import (
	"fmt"
	"pheebkit/internal/core"
	"testing"
)

func TestMediaCacheBoundsAndPinsCurrentCollection(t *testing.T) {
	s := &Service{}
	uris := make([]string, 32)
	for i := range uris {
		uris[i] = fmt.Sprintf("current-%d", i)
		s.cacheMedia(core.Media{Source: core.Source{URI: uris[i]}})
	}
	if err := s.SetActiveMedia(uris, 2); err != nil {
		t.Fatal(err)
	}
	if err := s.SetActiveMedia(nil, 1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		s.cacheMedia(core.Media{Source: core.Source{URI: fmt.Sprintf("sample-%d", i)}})
	}
	if len(s.media) != mediaCacheLimit || len(s.mediaUsed) != mediaCacheLimit {
		t.Fatal("media metadata grew without bound")
	}
	for _, uri := range uris {
		if _, ok := s.cachedMedia(uri); !ok {
			t.Fatal("visible batch was evicted")
		}
	}
	if _, ok := s.cachedMedia("sample-0"); ok {
		t.Fatal("old unpinned sample retained")
	}
	if err := s.SetActiveMedia([]string{"unknown"}, 3); err == nil {
		t.Fatal("unknown source pinned")
	}
	if len(s.mediaPinned) != 32 {
		t.Fatal("invalid pin update discarded current collection")
	}
}
