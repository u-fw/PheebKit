package compiler

import (
	"fmt"
	"math"

	"kompeg/internal/core"
)

func validateRecovery(s core.Settings) error {
	r := s.Recovery
	if r == nil {
		return nil
	}
	if r.Mode != "delete" && r.Mode != "black" {
		return fmt.Errorf("恢复策略需要删除区间或黑画面填补")
	}
	if r.SourceFingerprint == "" || len(r.SourceFingerprint) > 512 {
		return fmt.Errorf("恢复范围缺少源文件标识，请重新扫描")
	}
	if s.Operation != "" && s.Operation != "compress" {
		return fmt.Errorf("恢复需要视频压缩任务")
	}
	if s.Trim != nil || len(s.Segments) > 0 || s.SubtitleBurn != nil || len(s.RawArguments) > 0 {
		return fmt.Errorf("恢复不能与剪辑、字幕烧录或高级参数同时使用，请先恢复再编辑")
	}
	if s.Audio == "copy" {
		return fmt.Errorf("恢复需要重新编码音频或移除音频")
	}
	if len(r.DamagedRanges) == 0 || len(r.DamagedRanges) > 128 {
		return fmt.Errorf("恢复需要 1–128 个失败区间")
	}
	last := -1.0
	for _, t := range r.DamagedRanges {
		if math.IsNaN(t.StartSeconds) || math.IsNaN(t.EndSeconds) || math.IsInf(t.StartSeconds, 0) || math.IsInf(t.EndSeconds, 0) || t.StartSeconds < 0 || t.EndSeconds <= t.StartSeconds || t.StartSeconds <= last {
			return fmt.Errorf("恢复区间必须有限、按时间排序且不相邻或重叠")
		}
		last = t.EndSeconds
	}
	return nil
}

func recoveryTimeline(s core.Settings, total, rate float64) ([]core.TimeRange, error) {
	if total <= 0 || total > 21600 || math.IsInf(total, 0) || math.IsNaN(total) {
		return nil, fmt.Errorf("恢复需要已知时长且不超过六小时")
	}
	pieces := []core.TimeRange{}
	start := 0.0
	kept := 0.0
	for _, r := range s.Recovery.DamagedRanges {
		if r.EndSeconds > total {
			return nil, fmt.Errorf("恢复区间超过源视频时长")
		}
		if r.StartSeconds > start {
			pieces = append(pieces, core.TimeRange{StartSeconds: start, EndSeconds: r.StartSeconds})
			kept += r.StartSeconds - start
		}
		if s.Recovery.Mode == "black" {
			pieces = append(pieces, r)
		}
		start = r.EndSeconds
	}
	if start < total {
		pieces = append(pieces, core.TimeRange{StartSeconds: start, EndSeconds: total})
		kept += total - start
	}
	if kept <= 0 {
		return nil, fmt.Errorf("没有可保留的正常区间，无法恢复原视频内容")
	}
	if len(pieces) > 16 {
		return nil, fmt.Errorf("恢复后超过 16 个处理片段，请先分段处理；不会自动省略区间")
	}
	for _, r := range pieces {
		if rate > 0 && r.EndSeconds-r.StartSeconds < 1/rate {
			return nil, fmt.Errorf("恢复边界留下不足一帧的片段，请调整源文件范围")
		}
	}
	return pieces, nil
}

func replacementRange(s core.Settings, r core.TimeRange) bool {
	if s.Recovery == nil || s.Recovery.Mode != "black" {
		return false
	}
	for _, bad := range s.Recovery.DamagedRanges {
		if bad == r {
			return true
		}
	}
	return false
}
