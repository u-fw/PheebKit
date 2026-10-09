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
	Codec        string   `json:"codec"`
	Encoder      string   `json:"encoder"`
	Quality      int      `json:"quality"`
	Preset       string   `json:"preset"`
	Resolution   int      `json:"resolution"`
	Audio        string   `json:"audio"`
	Mode         string   `json:"mode"`
	BitrateKbps  int      `json:"bitrateKbps"`
	TargetSizeMB float64  `json:"targetSizeMB"`
	RawArguments []string `json:"rawArguments"`
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
