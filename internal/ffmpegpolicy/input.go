// Package ffmpegpolicy defines the supported local-input trust boundary.
package ffmpegpolicy

// LocalInput rejects implicit network, crypto/data and concatenation protocols.
// FFmpeg's whitelist is an input option; generated output pipes remain usable.
func LocalInput(path string) []string { return []string{"-protocol_whitelist", "file", "-i", path} }
