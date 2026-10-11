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
	Operation         string          `json:"operation,omitempty"`
	Codec             string          `json:"codec"`
	Encoder           string          `json:"encoder"`
	Quality           int             `json:"quality"`
	Preset            string          `json:"preset"`
	Resolution        int             `json:"resolution"`
	Audio             string          `json:"audio"`
	Mode              string          `json:"mode"`
	BitrateKbps       int             `json:"bitrateKbps"`
	TargetSizeMB      float64         `json:"targetSizeMB"`
	RawArguments      []string        `json:"rawArguments"`
	Trim              *TimeRange      `json:"trim,omitempty"`
	AudioSelection    *TrackSelection `json:"audioSelection,omitempty"`
	SubtitleSelection *TrackSelection `json:"subtitleSelection,omitempty"`
	Transform         *VideoTransform `json:"transform,omitempty"`
	VolumeDB          float64         `json:"volumeDB,omitempty"`
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
	settings.RawArguments = append([]string(nil), settings.RawArguments...)
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
