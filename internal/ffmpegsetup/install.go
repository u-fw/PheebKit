package ffmpegsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxDownload = int64(400 << 20)
const maxExpanded = int64(1 << 30)

type Progress struct {
	Stage    string  `json:"stage"`
	Package  string  `json:"package"`
	Source   string  `json:"source"`
	File     string  `json:"file"`
	Received int64   `json:"received"`
	Total    int64   `json:"total"`
	Speed    float64 `json:"speed"`
}

type Result struct{ Directory, FFmpeg, FFprobe, Version, Label string }
type Installer struct {
	Client *http.Client
	Report func(Progress)
}

func fmtInvalidSource(source string) error { return fmt.Errorf("未知下载源：%s", source) }

func trustedURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	switch u.Hostname() {
	case "www.gyan.dev", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com", "registry.npmmirror.com", "cdn.npmmirror.com":
		return true
	}
	return false
}

func New(report func(Progress)) *Installer {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 12 * time.Second
	return &Installer{Report: report, Client: &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !trustedURL(req.URL.String()) {
			return errors.New("下载重定向到未认可的地址")
		}
		return nil
	}}}
}

func (i *Installer) report(p Progress) {
	if i.Report != nil {
		i.Report(p)
	}
}

type progressWriter struct {
	file          io.Writer
	hash          io.Writer
	installer     *Installer
	progress      Progress
	started, last time.Time
	idle          *time.Timer
}

func (w *progressWriter) Write(data []byte) (int, error) {
	if w.idle != nil {
		w.idle.Reset(25 * time.Second)
	}
	n, err := w.file.Write(data)
	if n > 0 {
		_, _ = w.hash.Write(data[:n])
		w.progress.Received += int64(n)
	}
	if time.Since(w.last) >= 200*time.Millisecond {
		w.progress.Speed = float64(w.progress.Received) / time.Since(w.started).Seconds()
		w.installer.report(w.progress)
		w.last = time.Now()
	}
	return n, err
}

func (i *Installer) download(ctx context.Context, raw, target string, artifact Artifact, label string) error {
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	resp, err := i.Client.Do(req)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) {
			return urlError.Err
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxDownload {
		return errors.New("下载文件超出大小限制")
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	digest := sha256.New()
	started := time.Now()
	idle := time.AfterFunc(25*time.Second, cancel)
	defer idle.Stop()
	writer := &progressWriter{file: file, hash: digest, installer: i, started: started, progress: Progress{Stage: "downloading", Package: label, Source: safeDownloadURL(raw), File: artifact.Name, Total: resp.ContentLength}}
	writer.idle = idle
	i.report(writer.progress)
	n, copyErr := io.Copy(writer, io.LimitReader(resp.Body, maxDownload+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n > maxDownload {
		return errors.New("下载文件超出大小限制")
	}
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		return errors.New("下载文件不完整")
	}
	i.report(Progress{Stage: "verifying", Package: label, Source: safeDownloadURL(raw), File: artifact.Name, Received: n, Total: n})
	if hex.EncodeToString(digest.Sum(nil)) != artifact.SHA256 {
		return errors.New("SHA-256 校验失败，未执行下载文件")
	}
	return ctx.Err()
}

func copyBounded(ctx context.Context, destination string, source io.Reader, limit int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	n, err := io.Copy(file, &contextReader{ctx: ctx, reader: io.LimitReader(source, limit+1)})
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n > limit {
		return errors.New("解压文件超出大小限制")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func findBinaries(directory string) (string, string, error) {
	var ffmpeg string
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(entry.Name(), "ffmpeg.exe") {
			return nil
		}
		if ffmpeg != "" {
			return errors.New("压缩包包含多套 FFmpeg")
		}
		ffmpeg = path
		return nil
	})
	if err != nil {
		return "", "", err
	}
	if ffmpeg == "" {
		return "", "", errors.New("下载包缺少 ffmpeg.exe")
	}
	var ffprobe string
	entries, err := os.ReadDir(filepath.Dir(ffmpeg))
	if err != nil {
		return "", "", err
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), "ffprobe.exe") && entry.Type().IsRegular() {
			ffprobe = filepath.Join(filepath.Dir(ffmpeg), entry.Name())
		}
	}
	if ffprobe == "" {
		return "", "", errors.New("下载包缺少同目录 ffprobe.exe")
	}
	return ffmpeg, ffprobe, nil
}

// Install only publishes a uniquely named private directory. The caller must
// validate the binaries before switching configuration. Existing installs are
// never overwritten, including those owned by another running window.
func (i *Installer) Install(ctx context.Context, root string, packages []Package) (Result, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return Result{}, err
	}
	var failures []string
	for _, pkg := range packages {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		i.report(Progress{Stage: "selecting", Package: pkg.Label})
		result, err := i.installPackage(ctx, root, pkg)
		if err == nil {
			return result, nil
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		failures = append(failures, pkg.Label+"："+err.Error())
		i.report(Progress{Stage: "retrying", Package: pkg.Label})
	}
	return Result{}, fmt.Errorf("下载未完成，可重试、手动选择备用来源或选择本机文件。\n%s", strings.Join(failures, "\n"))
}

func (i *Installer) installPackage(ctx context.Context, root string, pkg Package) (Result, error) {
	stage, err := os.MkdirTemp(root, ".download-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(stage)
	payload := filepath.Join(stage, "payload")
	if err := os.Mkdir(payload, 0700); err != nil {
		return Result{}, err
	}
	for index, artifact := range pkg.Artifacts {
		archive := filepath.Join(stage, fmt.Sprintf("artifact-%d", index))
		var failures []string
		downloaded := false
		for _, raw := range artifact.URLs {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			if err := i.download(ctx, raw, archive, artifact, pkg.Label); err != nil {
				failures = append(failures, err.Error())
				i.report(Progress{Stage: "retrying", Package: pkg.Label, Source: safeDownloadURL(raw)})
				continue
			}
			downloaded = true
			break
		}
		if !downloaded {
			return Result{}, fmt.Errorf("%s：%s", artifact.Name, strings.Join(failures, "；"))
		}
		i.report(Progress{Stage: "extracting", Package: pkg.Label, File: artifact.Name})
		if err := extract(ctx, archive, payload, artifact); err != nil {
			return Result{}, err
		}
	}
	ffmpeg, ffprobe, err := findBinaries(payload)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	provenance := fmt.Sprintf("Downloaded by PheebKit. Package: %s\nVersion: %s\nKeep this directory's upstream license and build/source notices when redistributing.\n", pkg.ID, pkg.Version)
	for _, artifact := range pkg.Artifacts {
		urls := make([]string, len(artifact.URLs))
		for index, raw := range artifact.URLs {
			urls[index] = safeDownloadURL(raw)
		}
		provenance += fmt.Sprintf("%s SHA256 %s\n%s\n", artifact.Name, artifact.SHA256, strings.Join(urls, "\n"))
	}
	if err := os.WriteFile(filepath.Join(payload, "kompeg-download.txt"), []byte(provenance), 0600); err != nil {
		return Result{}, err
	}
	final := filepath.Join(root, pkg.ID+"-"+filepath.Base(stage)[len(".download-"):])
	ffmpegRelative, _ := filepath.Rel(payload, ffmpeg)
	ffprobeRelative, _ := filepath.Rel(payload, ffprobe)
	if err := os.Rename(payload, final); err != nil {
		return Result{}, err
	}
	return Result{Directory: final, FFmpeg: filepath.Join(final, ffmpegRelative), FFprobe: filepath.Join(final, ffprobeRelative), Version: pkg.Version, Label: pkg.Label}, nil
}
