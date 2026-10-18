package core

type AudioProcessing struct {
	Normalize          *Loudness       `json:"normalize,omitempty" ts_type:"Loudness | null"`
	DenoiseReductionDB float64         `json:"denoiseReductionDB"`
	NoiseFloorDB       float64         `json:"noiseFloorDB"`
	Equalizer          []EqualizerBand `json:"equalizer" ts_type:"EqualizerBand[] | null"`
	FadeInSeconds      float64         `json:"fadeInSeconds"`
	FadeOutSeconds     float64         `json:"fadeOutSeconds"`
	SampleRate         int             `json:"sampleRate"`
	Channels           string          `json:"channels"`
}

type Loudness struct {
	IntegratedLUFS float64 `json:"integratedLUFS"`
	TruePeakDBTP   float64 `json:"truePeakDBTP"`
	RangeLU        float64 `json:"rangeLU"`
}

type EqualizerBand struct {
	FrequencyHz float64 `json:"frequencyHz"`
	Q           float64 `json:"q"`
	GainDB      float64 `json:"gainDB"`
}
