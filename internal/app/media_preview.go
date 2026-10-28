package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/ffmpegpolicy"
	"pheebkit/internal/processguard"
	"pheebkit/internal/sourcefile"
)

const previewTokenLimit = 64

type previewGrant struct {
	media      core.Media
	references uint64
	lastUsed   uint64
}

func (s *Service) importedMedia(uri string) (core.Media, error) {
	media, found := s.cachedMedia(uri)
	if !found || media.Source.Kind != core.SourceLocalFile {
		return core.Media{}, errors.New("请先导入本地媒体")
	}
	if err := sourcefile.Verify(uri, media.SourceFingerprint); err != nil {
		return core.Media{}, err
	}
	return media, nil
}

func (s *Service) GetMediaPreviewURL(uri string) (string, error) {
	media, err := s.importedMedia(uri)
	if err != nil {
		return "", err
	}
	s.previewMu.Lock()
	defer s.previewMu.Unlock()
	if s.previewTokens == nil {
		s.previewTokens = make(map[string]*previewGrant)
	}
	// Reopening comparisons must not exhaust session capabilities and revoke an
	// existing player's URL for the same unchanged imported source.
	for key, grant := range s.previewTokens {
		if grant != nil &&
			grant.media.Source.URI == media.Source.URI &&
			grant.media.SourceFingerprint == media.SourceFingerprint {
			grant.references++
			s.previewSequence++
			grant.lastUsed = s.previewSequence
			return "/__media/" + key, nil
		}
	}
	// URLs are session capabilities; bound tokens and avoid exposing filesystem paths.
	if len(s.previewTokens) >= previewTokenLimit {
		var oldestKey string
		oldestUse := ^uint64(0)
		for key, grant := range s.previewTokens {
			if grant != nil && grant.lastUsed < oldestUse {
				oldestKey = key
				oldestUse = grant.lastUsed
			}
		}
		if oldestKey != "" {
			delete(s.previewTokens, oldestKey)
		}
	}
	token := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	key := hex.EncodeToString(token)
	s.previewSequence++
	s.previewTokens[key] = &previewGrant{
		media:      media,
		references: 1,
		lastUsed:   s.previewSequence,
	}
	return "/__media/" + key, nil
}

func (s *Service) ReleaseMediaPreviewURL(url string) {
	key, ok := previewTokenFromURL(url)
	if !ok {
		return
	}
	s.previewMu.Lock()
	defer s.previewMu.Unlock()
	grant := s.previewTokens[key]
	if grant == nil {
		return
	}
	if grant.references > 1 {
		grant.references--
		return
	}
	delete(s.previewTokens, key)
}

func previewTokenFromURL(url string) (string, bool) {
	const prefix = "/__media/"
	if !strings.HasPrefix(url, prefix) {
		return "", false
	}
	key := strings.TrimPrefix(url, prefix)
	if len(key) != 48 {
		return "", false
	}
	if _, err := hex.DecodeString(key); err != nil {
		return "", false
	}
	return key, true
}

//wails:ignore
func (s *Service) AssetHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyAssetSecurityHeaders(w.Header())
		if !strings.HasPrefix(r.URL.Path, "/__media/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimPrefix(r.URL.Path, "/__media/")
		s.previewMu.Lock()
		grant := s.previewTokens[token]
		if grant != nil {
			s.previewSequence++
			grant.lastUsed = s.previewSequence
		}
		s.previewMu.Unlock()
		if grant == nil {
			http.NotFound(w, r)
			return
		}
		media := grant.media
		if err := sourcefile.Verify(media.Source.URI, media.SourceFingerprint); err != nil {
			http.Error(w, "媒体已变化，请重新导入", http.StatusConflict)
			return
		}
		file, err := os.Open(media.Source.URI)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		// The pinned Windows WebView response writer buffers each response.
		// Never feed a multi-gigabyte source to it as an unbounded GET.
		if r.Method == http.MethodGet {
			rangeHeader, err := boundedMediaRange(r.Header.Get("Range"), info.Size())
			if err != nil {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", info.Size()))
				http.Error(w, "请使用分段读取预览视频", http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if rangeHeader != "" {
				r = r.Clone(r.Context())
				r.Header.Set("Range", rangeHeader)
				r.Header.Del("If-Range")
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		switch strings.ToLower(filepath.Ext(media.Source.URI)) {
		case ".mp4", ".m4v", ".mov":
			w.Header().Set("Content-Type", "video/mp4")
		case ".webm":
			w.Header().Set("Content-Type", "video/webm")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		http.ServeContent(w, r, filepath.Base(media.Source.URI), info.ModTime(), file)
	})
}

const assetContentSecurityPolicy = "default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; media-src 'self' blob:; font-src 'self'; connect-src 'self'; frame-src 'none'; worker-src 'self' blob:"

func applyAssetSecurityHeaders(header http.Header) {
	header.Set("Content-Security-Policy", assetContentSecurityPolicy)
	header.Set("Cross-Origin-Resource-Policy", "same-origin")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), serial=(), hid=()")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
}

const previewChunkBytes int64 = 2 * 1024 * 1024

func boundedMediaRange(header string, size int64) (string, error) {
	if header == "" {
		if size > previewChunkBytes {
			return "", errors.New("unbounded video response")
		}
		return "", nil
	}
	if !strings.HasPrefix(header, "bytes=") || strings.Contains(header, ",") {
		return "", errors.New("unsupported range")
	}
	parts := strings.Split(strings.TrimPrefix(header, "bytes="), "-")
	if len(parts) != 2 || size <= 0 {
		return "", errors.New("invalid range")
	}
	start, end := int64(0), size-1
	if parts[0] == "" {
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return "", errors.New("invalid suffix")
		}
		start = max(0, size-suffix)
	} else {
		var err error
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || start < 0 || start >= size {
			return "", errors.New("invalid start")
		}
		if parts[1] != "" {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil || end < start {
				return "", errors.New("invalid end")
			}
			end = min(end, size-1)
		}
	}
	end = min(end, start+min(previewChunkBytes-1, size-1-start))
	return fmt.Sprintf("bytes=%d-%d", start, end), nil
}

func (s *Service) GetVideoFrame(uri string, seconds float64) (Thumbnail, error) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	media, err := s.importedMedia(uri)
	if err != nil {
		return Thumbnail{}, err
	}
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 || media.DurationSeconds <= 0 || seconds > media.DurationSeconds {
		return Thumbnail{}, errors.New("预览时间超出视频时长")
	}
	var video *core.Stream
	for _, stream := range media.Streams {
		if stream.CodecType == "video" && !stream.AttachedPicture && !stream.TimedThumbnails {
			copy := stream
			video = &copy
			break
		}
	}
	if video == nil {
		return Thumbnail{}, errors.New("没有可预览的视频轨")
	}
	frameStep := 0.04
	if video.FrameRate > 0 {
		frameStep = 1 / video.FrameRate
	}
	seconds = min(seconds, max(0, media.DurationSeconds-frameStep))
	ctx, cancel := context.WithTimeout(s.ctx, 12*time.Second)
	defer cancel()
	release, err := acquireThumbnailSlot(ctx, thumbnailGate)
	if err != nil {
		return Thumbnail{}, err
	}
	defer release()
	args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-ss", strconv.FormatFloat(seconds, 'f', 6, 64)}
	args = append(args, ffmpegpolicy.LocalInput(uri)...)
	args = append(args, "-map", "0:"+strconv.Itoa(video.Index), "-frames:v", "1", "-an", "-sn", "-vf", "scale=512:512:force_original_aspect_ratio=decrease", "-c:v", "mjpeg", "-q:v", "3", "-f", "image2pipe", "pipe:1")
	command := exec.CommandContext(ctx, previewBinary(s.runner.snapshot()), args...)
	var output, stderr limitedBuffer
	output.limit = thumbnailLimit
	stderr.limit = 32 * 1024
	command.Stdout = &output
	command.Stderr = &stderr
	if err := processguard.Run(command); err != nil {
		return Thumbnail{}, fmt.Errorf("位置画面读取失败：%w：%s", err, stderr.String())
	}
	if err := sourcefile.Verify(uri, media.SourceFingerprint); err != nil {
		return Thumbnail{}, err
	}
	return thumbnailData(output.Bytes(), "video-position")
}

func (s *Service) OpenJobOutput(id string, folder bool) error {
	job, found := s.jobs.Get(id)
	if !found || job.State != core.JobCompleted {
		return errors.New("只能打开已完成任务的输出")
	}
	extension := strings.ToLower(filepath.Ext(job.Output.URI))
	allowed := map[string]bool{".mp4": true, ".mkv": true, ".webm": true, ".m4a": true, ".mka": true, ".opus": true, ".mp3": true, ".flac": true, ".wav": true, ".srt": true, ".ass": true, ".vtt": true}
	if !allowed[extension] {
		return errors.New("输出类型不允许打开")
	}
	info, err := os.Stat(job.Output.URI)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("输出文件已移动或删除")
	}
	return openOutput(job.Output.URI, folder)
}
