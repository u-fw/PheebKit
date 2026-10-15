package core

const ProjectSchemaVersion = 1

// Project is editable, versioned intent. The executable argv lives in a plan.
type Project struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Inputs        []Source `json:"inputs"`
	Output        Source   `json:"output"`
	Settings      Settings `json:"settings"`
}

type Settings struct {
	AudioProcessing   *AudioProcessing `json:"audioProcessing,omitempty"`
	Recovery          *Recovery        `json:"recovery,omitempty"`
	Operation         string           `json:"operation,omitempty"`
	Codec             string           `json:"codec"`
	Encoder           string           `json:"encoder"`
	Quality           int              `json:"quality"`
	Preset            string           `json:"preset"`
	Resolution        int              `json:"resolution"`
	Audio             string           `json:"audio"`
	AudioBitrateKbps  int              `json:"audioBitrateKbps,omitempty"`
	Mode              string           `json:"mode"`
	BitrateKbps       int              `json:"bitrateKbps"`
	TargetSizeMB      float64          `json:"targetSizeMB"`
	RawArguments      []string         `json:"rawArguments"`
	Trim              *TimeRange       `json:"trim,omitempty"`
	Segments          []TimeRange      `json:"segments,omitempty"`
	SubtitleBurn      *SubtitleBurn    `json:"subtitleBurn,omitempty"`
	AudioSelection    *TrackSelection  `json:"audioSelection,omitempty"`
	SubtitleSelection *TrackSelection  `json:"subtitleSelection,omitempty"`
	Transform         *VideoTransform  `json:"transform,omitempty"`
	VolumeDB          float64          `json:"volumeDB,omitempty"`
}

// SubtitleBurn identifies editable subtitle intent; compilation freezes its file identity.
type SubtitleBurn struct {
	Source        string  `json:"source"`
	Path          string  `json:"path,omitempty"`
	StreamIndex   int     `json:"streamIndex"`
	OffsetSeconds float64 `json:"offsetSeconds"`
	Font          string  `json:"font"`
	FontSize      int     `json:"fontSize"`
	Color         string  `json:"color"`
	OutlineColor  string  `json:"outlineColor"`
	Outline       int     `json:"outline"`
	Shadow        int     `json:"shadow"`
	Alignment     int     `json:"alignment"`
	Margin        int     `json:"margin"`
	Bold          bool    `json:"bold"`
}

// TrackSelection uses global input stream indexes from ffprobe, not an index
// relative to audio/subtitle streams. Nil preserves the original defaults.
type TrackSelection struct {
	Mode    string `json:"mode"`
	Indices []int  `json:"indices,omitempty"`
}

// VideoTransform applies crop, clockwise rotation, then horizontal/vertical flips.
type VideoTransform struct {
	Rotation       int          `json:"rotation"`
	FlipHorizontal bool         `json:"flipHorizontal"`
	FlipVertical   bool         `json:"flipVertical"`
	Crop           *CropMargins `json:"crop,omitempty"`
}

type CropMargins struct {
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
	Right  int `json:"right"`
}

// CloneSettings isolates editable project, preset and queue snapshots.
func CloneSettings(settings Settings) Settings {
	if settings.AudioProcessing != nil {
		copy := *settings.AudioProcessing
		copy.Equalizer = append([]EqualizerBand(nil), copy.Equalizer...)
		if copy.Normalize != nil {
			n := *copy.Normalize
			copy.Normalize = &n
		}
		settings.AudioProcessing = &copy
	}
	if settings.Recovery != nil {
		copy := *settings.Recovery
		copy.DamagedRanges = append([]TimeRange(nil), copy.DamagedRanges...)
		settings.Recovery = &copy
	}
	settings.RawArguments = append([]string(nil), settings.RawArguments...)
	settings.Segments = append([]TimeRange(nil), settings.Segments...)
	if settings.SubtitleBurn != nil {
		copy := *settings.SubtitleBurn
		settings.SubtitleBurn = &copy
	}
	if settings.Trim != nil {
		copy := *settings.Trim
		settings.Trim = &copy
	}
	if settings.AudioSelection != nil {
		copy := *settings.AudioSelection
		copy.Indices = append([]int(nil), copy.Indices...)
		settings.AudioSelection = &copy
	}
	if settings.SubtitleSelection != nil {
		copy := *settings.SubtitleSelection
		copy.Indices = append([]int(nil), copy.Indices...)
		settings.SubtitleSelection = &copy
	}
	if settings.Transform != nil {
		copy := *settings.Transform
		if copy.Crop != nil {
			crop := *copy.Crop
			copy.Crop = &crop
		}
		settings.Transform = &copy
	}
	return settings
}

func CloneProject(project Project) Project {
	project.Inputs = append([]Source(nil), project.Inputs...)
	project.Settings = CloneSettings(project.Settings)
	return project
}

// TimeRange is relative to the source start, in seconds. Nil retains the full input.
type TimeRange struct {
	StartSeconds float64 `json:"startSeconds"`
	EndSeconds   float64 `json:"endSeconds"`
}

type Preset struct {
	SchemaVersion int      `json:"schemaVersion"`
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Settings      Settings `json:"settings"`
}

func DefaultSettings() Settings {
	return Settings{Codec: "hevc", Encoder: "auto", Quality: 26, Preset: "medium", Audio: "aac", Mode: "quality"}
}
