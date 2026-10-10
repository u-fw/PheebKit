package ffmpegsetup

import (
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// A custom link is a file URL, not a repository or mirror-root convention.
func validCustomURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == "" && !strings.ContainsAny(raw, "\r\n\t ")
}

func CustomArchive(raw, digest string, trustBuild bool) (Package, error) {
	raw, digest = strings.TrimSpace(raw), strings.ToLower(strings.TrimSpace(digest))
	if !validCustomURL(raw) {
		return Package{}, errors.New("请输入完整的 HTTPS 文件下载链接，不支持账号密码或 URL 片段。")
	}
	if digest == "" {
		packages, err := Catalog("stable")
		if err != nil {
			return Package{}, err
		}
		pkg := packages[0]
		pkg.ID, pkg.Label = "custom-gyan-9.0.2", "自定义镜像 · Gyan 9.0.2 完整构建"
		pkg.Artifacts[0].URLs = []string{raw}
		return pkg, nil
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 {
		return Package{}, errors.New("SHA-256 必须是完整的 64 位十六进制校验值。")
	}
	if !trustBuild {
		return Package{}, errors.New("其他构建会运行下载包中的程序，请先确认信任该来源。SHA-256 只能校验文件一致性，不能证明来源可信。")
	}
	return Package{ID: "custom-archive", Label: "自定义构建 · 来源由用户确认", Artifacts: []Artifact{{Name: "ffmpeg-package", Format: "auto", SHA256: digest, URLs: []string{raw}}}}, nil
}

func safeDownloadURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid URL"
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	u.ForceQuery = false
	return u.String()
}

// User-chosen links may redirect to CDN hosts; built-ins keep their allowlist.
func NewCustom(report func(Progress)) *Installer {
	i := New(report)
	i.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !validCustomURL(req.URL.String()) {
			return errors.New("自定义下载只能重定向到 HTTPS 文件地址（最多 5 次）。")
		}
		return nil
	}
	return i
}
