package compiler

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"kompeg/internal/core"
	"kompeg/internal/ffmpegpolicy"
	"kompeg/internal/sourcefile"
)

func validateEditing(s core.Settings) error {
	if len(s.Segments) > 16 {
		return fmt.Errorf("最多保留 16 个片段")
	}
	if len(s.Segments) > 0 && s.Trim != nil {
		return fmt.Errorf("单区间与多片段不能同时启用")
	}
	if (len(s.Segments) > 0 || s.SubtitleBurn != nil) && s.Operation != "" && s.Operation != "compress" {
		return fmt.Errorf("多片段和字幕烧录需要视频压缩任务")
	}
	for _, r := range s.Segments {
		if math.IsNaN(r.StartSeconds) || math.IsNaN(r.EndSeconds) || math.IsInf(r.StartSeconds, 0) || math.IsInf(r.EndSeconds, 0) || r.StartSeconds < 0 || r.EndSeconds <= r.StartSeconds {
			return fmt.Errorf("片段需要有限的 0 ≤ 起点 < 终点")
		}
	}
	b := s.SubtitleBurn
	if b == nil {
		return nil
	}
	if b.Source != "embedded" && b.Source != "external" {
		return fmt.Errorf("请选择内嵌文字字幕或本地字幕文件")
	}
	if b.Source == "external" && (!filepath.IsAbs(b.Path) || strings.ContainsRune(b.Path, 0) || subtitleFormat(b.Path) == "") {
		return fmt.Errorf("外挂字幕需要本地绝对路径的 SRT / ASS / SSA / VTT 文件")
	}
	if b.Source == "embedded" && (b.Path != "" || b.StreamIndex < 0 || b.StreamIndex > 65535) {
		return fmt.Errorf("内嵌字幕轨道无效")
	}
	if math.IsNaN(b.OffsetSeconds) || math.IsInf(b.OffsetSeconds, 0) || math.Abs(b.OffsetSeconds) > 86400 {
		return fmt.Errorf("字幕偏移需在 ±86400 秒内")
	}
	if b.FontSize < 8 || b.FontSize > 144 || b.Outline < 0 || b.Outline > 10 || b.Shadow < 0 || b.Shadow > 10 || b.Alignment < 1 || b.Alignment > 9 || b.Margin < 0 || b.Margin > 500 {
		return fmt.Errorf("字幕字号、边框、阴影、位置或边距超出范围")
	}
	if len([]rune(b.Font)) < 1 || len([]rune(b.Font)) > 64 {
		return fmt.Errorf("字体名需要 1–64 个字符")
	}
	for _, c := range b.Font {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && !strings.ContainsRune(" ._-", c) {
			return fmt.Errorf("字体名只能包含文字、数字、空格、点、下划线或连字符")
		}
	}
	color := regexp.MustCompile(`^#[a-fA-F0-9]{6}$`)
	if !color.MatchString(b.Color) || !color.MatchString(b.OutlineColor) {
		return fmt.Errorf("字幕颜色需要 #RRGGBB")
	}
	return nil
}

func subtitleFormat(path string) string {
	return map[string]string{".srt": "srt", ".ass": "ass", ".ssa": "ass", ".vtt": "webvtt"}[strings.ToLower(filepath.Ext(path))]
}
func hasFilter(c core.Capabilities, name string) bool {
	for _, f := range c.Filters {
		if f == name {
			return true
		}
	}
	return false
}
func compileSubtitleAsset(s core.Settings, media core.Media, c core.Capabilities, output string) (*core.SubtitleAsset, error) {
	b := s.SubtitleBurn
	if b == nil {
		return nil, nil
	}
	if !hasFilter(c, "subtitles") {
		return nil, fmt.Errorf("当前 FFmpeg 未包含 subtitles/libass 滤镜，请更换支持文字字幕烧录的构建")
	}
	if !hasFilter(c, "setpts") {
		return nil, fmt.Errorf("字幕时间对齐需要 setpts 滤镜")
	}
	if !hasEncoderName(c, "ass") {
		return nil, fmt.Errorf("当前 FFmpeg 缺少 ASS 字幕编码器")
	}
	asset := &core.SubtitleAsset{Source: media.Source, Fingerprint: media.SourceFingerprint, StreamIndex: b.StreamIndex}
	if b.Source == "external" {
		info, err := os.Stat(b.Path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 16*1024*1024 {
			return nil, fmt.Errorf("字幕文件需要可读的常规文件，且不超过 16 MiB")
		}
		if samePath(b.Path, output) {
			return nil, fmt.Errorf("输出不能覆盖外挂字幕")
		}
		fp, err := sourcefile.Fingerprint(b.Path)
		if err != nil {
			return nil, err
		}
		asset.Source = core.Source{URI: b.Path, Kind: core.SourceLocalFile}
		asset.Fingerprint = fp
		asset.StreamIndex = 0
		asset.Format = subtitleFormat(b.Path)
	} else {
		found := false
		for _, track := range media.Streams {
			if track.Index == b.StreamIndex && track.CodecType == "subtitle" {
				found = map[string]bool{"subrip": true, "srt": true, "ass": true, "ssa": true, "webvtt": true, "mov_text": true, "text": true}[track.CodecName]
			}
		}
		if !found {
			return nil, fmt.Errorf("请选择可烧录的文字字幕轨道；图片字幕/OCR 尚未支持")
		}
	}
	return asset, nil
}
func hasEncoderName(c core.Capabilities, n string) bool {
	for _, e := range c.Encoders {
		if e.Name == n {
			return true
		}
	}
	return false
}

func burnFilter(b *core.SubtitleBurn, start float64) string {
	if b == nil {
		return ""
	}
	assColor := func(c string) string { return "&H00" + c[5:7] + c[3:5] + c[1:3] }
	bold := 0
	if b.Bold {
		bold = 1
	}
	style := fmt.Sprintf("Fontname=%s,Fontsize=%d,PrimaryColour=%s,OutlineColour=%s,Outline=%d,Shadow=%d,Alignment=%d,MarginV=%d,MarginL=%d,MarginR=%d,Bold=%d", b.Font, b.FontSize, assColor(b.Color), assColor(b.OutlineColor), b.Outline, b.Shadow, b.Alignment, b.Margin, b.Margin, b.Margin, bold)
	// Only validated typed style values enter this graph. The runner stages this
	// fixed filename inside its private working directory using restricted input.
	return "setpts=PTS+(" + decimal(start-b.OffsetSeconds) + ")/TB,subtitles=filename=pheebkit-subtitles.ass:force_style='" + style + "',setpts=PTS-(" + decimal(start-b.OffsetSeconds) + ")/TB"
}
func decimal(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// SelectedDuration validates source-relative ranges and returns their total budget.
func SelectedDuration(s core.Settings, total, rate float64) (float64, error) {
	if len(s.Segments) == 0 {
		d, err := outputDuration(s.Trim, total)
		if err == nil && s.Trim != nil && rate > 0 && d < 1/rate {
			return 0, fmt.Errorf("trim duration must include at least one video frame (%.3f seconds)", 1/rate)
		}
		return d, err
	}
	sum := 0.0
	for _, r := range s.Segments {
		d, err := outputDuration(&r, total)
		if err != nil {
			return 0, err
		}
		if rate > 0 && d < 1/rate {
			return 0, fmt.Errorf("每个片段至少保留一帧视频")
		}
		sum += d
	}
	if math.IsInf(sum, 0) || math.IsNaN(sum) {
		return 0, fmt.Errorf("片段总时长超出可处理范围")
	}
	return sum, nil
}

func segmentGraph(s core.Settings, input string, video core.Stream, audio []core.Stream, filter string) ([]string, error) {
	if len(audio) > 4 {
		return nil, fmt.Errorf("多片段最多保留 4 条音轨，请先选择所需轨道")
	}
	args := []string{}
	graph := []string{}
	labels := ""
	for i, r := range s.Segments {
		// Independent bounded seeks avoid buffering an entire source for reordered
		// split branches. Decode concurrency is bounded by the 16-range limit.
		args = append(args, "-threads", "1", "-ss", decimal(r.StartSeconds), "-t", decimal(r.EndSeconds-r.StartSeconds), "-accurate_seek")
		args = append(args, ffmpegpolicy.LocalInput(input)...)
		v := fmt.Sprintf("v%d", i)
		chain := "setpts=PTS-STARTPTS," + filter
		if s.SubtitleBurn != nil {
			chain += "," + burnFilter(s.SubtitleBurn, r.StartSeconds)
		}
		chain += ",trim=duration=" + decimal(r.EndSeconds-r.StartSeconds) + ",setpts=PTS-STARTPTS"
		graph = append(graph, fmt.Sprintf("[%d:%d]%s[%s]", i, video.Index, chain, v))
		labels += "[" + v + "]"
		for j, a := range audio {
			label := fmt.Sprintf("a%d_%d", i, j)
			audioFilter := "atrim=duration=" + decimal(r.EndSeconds-r.StartSeconds) + ",asetpts=PTS-STARTPTS"
			if s.VolumeDB != 0 {
				audioFilter += ",volume=" + decimal(s.VolumeDB) + "dB"
			}
			graph = append(graph, fmt.Sprintf("[%d:%d]%s[%s]", i, a.Index, audioFilter, label))
			labels += "[" + label + "]"
		}
	}
	outputs := "[joinedv]"
	for j := range audio {
		outputs += fmt.Sprintf("[joineda%d]", j)
	}
	graph = append(graph, fmt.Sprintf("%sconcat=n=%d:v=1:a=%d%s", labels, len(s.Segments), len(audio), outputs))
	args = append(args, "-filter_complex_threads", "1", "-filter_complex", strings.Join(graph, ";"), "-map", "[joinedv]")
	for j := range audio {
		args = append(args, "-map", fmt.Sprintf("[joineda%d]", j))
		if audio[j].Language != "" {
			args = append(args, fmt.Sprintf("-metadata:s:a:%d", j), "language="+audio[j].Language)
		}
	}
	return append(args, "-map_chapters", "-1"), nil
}
