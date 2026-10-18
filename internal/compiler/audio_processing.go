package compiler

import (
	"fmt"
	"math"
	"strings"

	"pheebkit/internal/core"
)

func finiteBetween(v, lo, hi float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= lo && v <= hi
}

func validateAudioProcessing(s core.Settings) error {
	p := s.AudioProcessing
	if p == nil {
		return nil
	}
	if s.Operation != "" && s.Operation != "compress" && s.Operation != "extract-audio" {
		return fmt.Errorf("声音处理仅适用于压缩或音频提取")
	}
	if s.Audio == "copy" || s.Audio == "none" {
		return fmt.Errorf("声音处理需要重新编码且保留音轨")
	}
	if s.Recovery != nil {
		return fmt.Errorf("请先恢复视频，再应用声音处理")
	}
	if !finiteBetween(p.DenoiseReductionDB, 0, 97) || (p.DenoiseReductionDB > 0 && p.DenoiseReductionDB < .01) {
		return fmt.Errorf("降噪强度需要为 0（关闭）或 0.01–97 dB")
	}
	if p.DenoiseReductionDB > 0 && !finiteBetween(p.NoiseFloorDB, -80, -20) {
		return fmt.Errorf("噪声底需要为 -80 至 -20 dB")
	}
	if p.Normalize != nil {
		n := p.Normalize
		if !finiteBetween(n.IntegratedLUFS, -70, -5) || !finiteBetween(n.TruePeakDBTP, -9, 0) || !finiteBetween(n.RangeLU, 1, 50) {
			return fmt.Errorf("响度目标、真峰值或响度范围超出允许范围")
		}
	}
	if len(p.Equalizer) > 8 {
		return fmt.Errorf("均衡器最多 8 个频段")
	}
	for _, b := range p.Equalizer {
		if !finiteBetween(b.FrequencyHz, 20, 20000) || !finiteBetween(b.Q, .1, 20) || !finiteBetween(b.GainDB, -24, 24) {
			return fmt.Errorf("均衡器需要 20–20000 Hz、Q 0.1–20、增益 ±24 dB")
		}
	}
	if !finiteBetween(p.FadeInSeconds, 0, 600) || !finiteBetween(p.FadeOutSeconds, 0, 600) {
		return fmt.Errorf("淡入淡出需要为 0–600 秒")
	}
	allowed := map[int]bool{0: true, 8000: true, 11025: true, 12000: true, 16000: true, 22050: true, 24000: true, 32000: true, 44100: true, 48000: true, 64000: true, 88200: true, 96000: true}
	if !allowed[p.SampleRate] {
		return fmt.Errorf("请选择支持的输出采样率")
	}
	if s.Audio == "opus" && p.SampleRate != 0 && p.SampleRate != 48000 {
		return fmt.Errorf("Opus 解码采样率固定为 48000 Hz，请选择自动或 48000 Hz")
	}
	if s.Audio == "mp3" && p.SampleRate > 48000 {
		return fmt.Errorf("MP3 输出采样率不能高于 48000 Hz")
	}
	if p.Channels != "" && p.Channels != "source" && p.Channels != "mono" && p.Channels != "stereo" {
		return fmt.Errorf("声道设置需要为保留、单声道或立体声")
	}
	for _, b := range p.Equalizer {
		if b.GainDB != 0 && b.FrequencyHz >= float64(processingSampleRate(p))/2 {
			return fmt.Errorf("有效均衡频率需要低于输出采样率的一半")
		}
	}
	return nil
}

// ValidateAudioDuration is also used before narrowing intent to a sample.
func ValidateAudioDuration(s core.Settings, duration float64) error {
	p := s.AudioProcessing
	if p == nil || p.FadeInSeconds+p.FadeOutSeconds == 0 {
		return nil
	}
	if !finiteBetween(duration, 0, math.MaxFloat64) || duration <= 0 {
		return fmt.Errorf("淡入淡出需要已知输出时长")
	}
	if p.FadeInSeconds+p.FadeOutSeconds > duration+1e-9 {
		return fmt.Errorf("淡入与淡出的总时长不能超过输出时长（%.3f 秒）", duration)
	}
	return nil
}

func processingSampleRate(p *core.AudioProcessing) int {
	if p.SampleRate == 0 {
		return 48000
	}
	return p.SampleRate
}
func processingChannels(p *core.AudioProcessing) int {
	return map[string]int{"mono": 1, "stereo": 2}[p.Channels]
}

func audioProcessingFilter(s core.Settings, duration float64, caps core.Capabilities, tracks []core.Stream) (string, error) {
	p := s.AudioProcessing
	if p == nil {
		return "", nil
	}
	if len(tracks) == 0 {
		return "", fmt.Errorf("声音处理需要至少一条选中的音轨")
	}
	if len(tracks) > 4 {
		return "", fmt.Errorf("声音处理最多同时处理 4 条音轨")
	}
	if err := ValidateAudioDuration(s, duration); err != nil {
		return "", err
	}
	rate := processingSampleRate(p)
	first := fmt.Sprintf("aresample=%d:osf=fltp", rate)
	if p.Channels == "mono" || p.Channels == "stereo" {
		first += ":out_chlayout=" + p.Channels
	}
	chain := []string{first}
	required := []string{"aresample"}
	if s.VolumeDB != 0 {
		chain = append(chain, "volume="+decimal(s.VolumeDB)+"dB")
		required = append(required, "volume")
	}
	if p.DenoiseReductionDB > 0 {
		chain = append(chain, "afftdn=nr="+decimal(p.DenoiseReductionDB)+":nf="+decimal(p.NoiseFloorDB))
		required = append(required, "afftdn")
	}
	for _, b := range p.Equalizer {
		if b.GainDB != 0 {
			chain = append(chain, "equalizer=f="+decimal(b.FrequencyHz)+":t=q:w="+decimal(b.Q)+":g="+decimal(b.GainDB))
			required = append(required, "equalizer")
		}
	}
	if n := p.Normalize; n != nil {
		chain = append(chain, "aresample=osf=dbl", "loudnorm=I="+decimal(n.IntegratedLUFS)+":TP="+decimal(n.TruePeakDBTP)+":LRA="+decimal(n.RangeLU)+":linear=false", "aresample="+fmt.Sprint(rate)+":osf=fltp")
		required = append(required, "loudnorm")
	}
	if p.FadeInSeconds > 0 {
		chain = append(chain, "afade=t=in:st=0:d="+decimal(p.FadeInSeconds)+":curve=tri")
		required = append(required, "afade")
	}
	if p.FadeOutSeconds > 0 {
		chain = append(chain, "afade=t=out:st="+decimal(math.Max(0, duration-p.FadeOutSeconds))+":d="+decimal(p.FadeOutSeconds)+":curve=tri")
		required = append(required, "afade")
	}
	format := map[string]string{"aac": "fltp", "opus": "flt", "mp3": "fltp", "flac": "s32", "wav": "s16"}[s.Audio]
	if format == "" {
		return "", fmt.Errorf("当前音频编码不支持声音处理")
	}
	chain = append(chain, fmt.Sprintf("aresample=%d:osf=%s", rate, format))
	for _, name := range required {
		if !hasFilter(caps, name) {
			return "", fmt.Errorf("当前 FFmpeg 缺少声音处理所需的 %s 滤镜", name)
		}
	}
	return strings.Join(chain, ","), nil
}

func expectAudioProcessing(expected *core.OutputExpectation, s core.Settings, tracks []core.Stream) {
	if s.AudioProcessing != nil {
		expected.AudioSampleRate = processingSampleRate(s.AudioProcessing)
		expected.AudioChannels = processingChannels(s.AudioProcessing)
		if expected.AudioChannels == 0 {
			for _, track := range tracks {
				expected.AudioSourceChannels = append(expected.AudioSourceChannels, track.Channels)
			}
		}
	}
}
