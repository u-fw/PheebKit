package presets

import "pheebkit/internal/core"

func Builtin() []core.Preset {
	balanced := core.DefaultSettings()
	small := balanced
	small.Quality = 30
	high := balanced
	high.Quality = 22
	veryHigh := balanced
	veryHigh.Quality = 18
	return []core.Preset{
		{SchemaVersion: 1, ID: "small", Name: "Small", Settings: small},
		{SchemaVersion: 1, ID: "balanced", Name: "Balanced", Settings: balanced},
		{SchemaVersion: 1, ID: "high", Name: "High", Settings: high},
		{SchemaVersion: 1, ID: "very-high", Name: "Very High", Settings: veryHigh},
	}
}
