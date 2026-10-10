package ffmpegsetup

type Artifact struct {
	Name, SHA256, Format string
	URLs                 []string
}

type Package struct {
	ID, Version, Label string
	Artifacts          []Artifact
}

// Digests are pinned from the upstream GitHub release metadata, not supplied
// by the download mirror. Updating them requires a reviewed catalog change.
func Catalog(source string) ([]Package, error) {
	stable := Package{ID: "gyan-9.0.2", Version: "9.0.2", Label: "Gyan 稳定版 9.0.2 · 完整构建", Artifacts: []Artifact{{
		Name: "ffmpeg.zip", Format: "zip", SHA256: "759d0a9831c436a0eb331ad36f236c06cb04aaa0005da46571f0e9d3d9206f6b",
		URLs: []string{"https://www.gyan.dev/ffmpeg/builds/packages/ffmpeg-9.0.2-full_build.zip", "https://github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-full_build.zip"},
	}}}
	base := "https://registry.npmmirror.com/-/binary/ffmpeg-static/b6.1.1/"
	domestic := Package{ID: "npmmirror-6.1.1", Version: "6.1.1", Label: "国内备用 6.1.1 · 较旧版本", Artifacts: []Artifact{
		{Name: "ffmpeg.exe", Format: "gzip", SHA256: "8883a3dffbd0a16cf4ef95206ea05283f78908dbfb118f73c83f4951dcc06d77", URLs: []string{base + "ffmpeg-win32-x64.gz"}},
		{Name: "ffprobe.exe", Format: "gzip", SHA256: "f309e6223ad89d2fe54bccd420a7709b66fd27540674e92309578ed491a43c8d", URLs: []string{base + "ffprobe-win32-x64.gz"}},
		{Name: "LICENSE", Format: "plain", SHA256: "8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903", URLs: []string{base + "win32-x64.LICENSE"}},
		{Name: "README.txt", Format: "plain", SHA256: "a636a7183c58006351acbaf35303c0ed85c6e1320fd4e80de453ba6157de6311", URLs: []string{base + "win32-x64.README"}},
	}}
	switch source {
	case "auto":
		return []Package{stable, domestic}, nil
	case "stable":
		return []Package{stable}, nil
	case "cn":
		return []Package{domestic}, nil
	default:
		return nil, fmtInvalidSource(source)
	}
}
