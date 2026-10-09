//go:build windows

// Command spike verifies the Windows vertical slice without the GUI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kompeg/internal/compiler"
	"kompeg/internal/core"
	"kompeg/internal/runner/windows"
)

func main() {
	input := flag.String("input", "", "absolute input video path")
	output := flag.String("output", "", "new absolute output .mp4/.mkv path")
	flag.Parse()
	if *input == "" || *output == "" {
		flag.Usage()
		os.Exit(2)
	}
	r := windows.New(os.Getenv("KOMPEG_FFMPEG"), os.Getenv("KOMPEG_FFPROBE"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	src := core.Source{URI: *input, DisplayName: filepath.Base(*input), Kind: core.SourceLocalFile}
	m, err := r.Probe(ctx, src)
	if err != nil {
		fail(err)
	}
	caps, err := r.Capabilities(ctx)
	if err != nil {
		fail(err)
	}
	settings := core.DefaultSettings()
	settings.Preset = "fast"
	plan, err := compiler.Compile(core.Project{SchemaVersion: 1, ID: "windows-spike", Inputs: []core.Source{src}, Output: core.Source{URI: *output, DisplayName: filepath.Base(*output), Kind: core.SourceLocalFile}, Settings: settings}, m, caps)
	if err != nil {
		fail(err)
	}
	plan.JobID = fmt.Sprintf("spike-%d", time.Now().UnixNano())
	plan.Preview = compiler.Preview(r.FFmpeg, plan.Args)
	fmt.Fprintln(os.Stderr, plan.Preview)
	result, err := r.Start(ctx, plan, core.Callbacks{
		Progress: func(p core.Progress) {
			fmt.Fprintf(os.Stderr, "progress %.1f%% fps %.1f speed %.2fx out_time %.2fs\n", p.Percent, p.FPS, p.Speed, p.CurrentTimeSeconds)
		},
	})
	if err != nil {
		fail(err)
	}
	encoded, err := r.Probe(ctx, result.Output)
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Input  core.Media  `json:"input"`
		Output core.Media  `json:"output"`
		Result core.Result `json:"result"`
	}{m, encoded, result}); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
