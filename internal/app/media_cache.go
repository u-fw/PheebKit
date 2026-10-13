package app

import (
	"errors"
	"kompeg/internal/core"
)

const mediaCacheLimit = 128

func (s *Service) touchMediaLocked(uri string) {
	if s.mediaUsed == nil {
		s.mediaUsed = map[string]uint64{}
	}
	s.mediaSequence++
	s.mediaUsed[uri] = s.mediaSequence
}
func (s *Service) cacheMedia(media core.Media) {
	s.mediaMu.Lock()
	defer s.mediaMu.Unlock()
	if s.media == nil {
		s.media = map[string]core.Media{}
	}
	s.media[media.Source.URI] = media
	s.touchMediaLocked(media.Source.URI)
	for len(s.media) > mediaCacheLimit {
		oldest := ""
		var age uint64
		for uri := range s.media {
			if s.mediaPinned[uri] {
				continue
			}
			if oldest == "" || s.mediaUsed[uri] < age {
				oldest, age = uri, s.mediaUsed[uri]
			}
		}
		if oldest == "" {
			break
		}
		delete(s.media, oldest)
		delete(s.mediaUsed, oldest)
	}
}
func (s *Service) cachedMedia(uri string) (core.Media, bool) {
	s.mediaMu.Lock()
	defer s.mediaMu.Unlock()
	media, found := s.media[uri]
	if found {
		s.touchMediaLocked(uri)
	}
	return media, found
}

// Pin the current collection so cache cleanup cannot break a visible batch.
// Historical samples/projects re-probe their source when opened or reused.
func (s *Service) SetActiveMedia(uris []string, revision uint64) error {
	if len(uris) > 32 {
		return errors.New("当前媒体集合最多 32 个文件")
	}
	s.mediaMu.Lock()
	defer s.mediaMu.Unlock()
	if revision <= s.mediaRevision {
		return nil
	}
	pinned := make(map[string]bool, len(uris))
	for _, uri := range uris {
		if _, found := s.media[uri]; !found {
			return errors.New("媒体缓存已过期，请重新导入")
		}
		pinned[uri] = true
	}
	s.mediaPinned = pinned
	s.mediaRevision = revision
	return nil
}
