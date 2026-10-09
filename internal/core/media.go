// Package core contains platform-independent models and execution contracts.
package core

type Source struct {
	URI         string `json:"uri"`
	DisplayName string `json:"displayName"`
	Kind        string `json:"kind"`
}

const (
	SourceLocalFile  = "local-file"
	SourceContentURI = "content-uri"
)

type Media struct {
	Source          Source   `json:"source"`
	Name            string   `json:"name"`
	Container       string   `json:"container"`
	SizeBytes       int64    `json:"sizeBytes"`
	DurationSeconds float64  `json:"durationSeconds"`
	Bitrate         int64    `json:"bitrate"`
	Streams         []Stream `json:"streams"`
}

type Stream struct {
	Index           int     `json:"index"`
	CodecType       string  `json:"codecType"`
	CodecName       string  `json:"codecName"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	FrameRate       float64 `json:"frameRate"`
	PixelFormat     string  `json:"pixelFormat"`
	BitDepth        int     `json:"bitDepth"`
	ColorSpace      string  `json:"colorSpace"`
	ColorTransfer   string  `json:"colorTransfer"`
	ColorPrimaries  string  `json:"colorPrimaries"`
	HDR             string  `json:"hdr"`
	Language        string  `json:"language"`
	Title           string  `json:"title"`
	AttachedPicture bool    `json:"attachedPicture"`
	TimedThumbnails bool    `json:"timedThumbnails"`
}
