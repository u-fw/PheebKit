package windows

import (
	"strings"
	"testing"

	"kompeg/internal/core"
)

func TestProgressParsesMachineProtocolAndMicroseconds(t *testing.T) {
	var updates []core.Progress
	text := "frame=30\nfps=29.97\nout_time_us=1001000\nout_time=00:00:01.001000\nspeed=1.5x\ntotal_size=2048\nprogress=continue\nframe=60\nout_time_ms=2002000\nprogress=end\n"
	if err := readProgress(strings.NewReader(text), 4, func(p core.Progress) { updates = append(updates, p) }); err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 || updates[0].CurrentTimeSeconds != 1.001 || updates[0].Speed != 1.5 || updates[0].OutputSizeBytes != 2048 {
		t.Fatalf("unexpected progress: %+v", updates)
	}
	if updates[1].CurrentTimeSeconds != 2.002 || updates[1].Percent != 50.05 || !updates[1].Finished {
		t.Fatalf("out_time_ms was not interpreted as microseconds: %+v", updates[1])
	}
}

func TestLogsAreBounded(t *testing.T) {
	var count int
	tail, err := readLogs(strings.NewReader(strings.Repeat(strings.Repeat("a", 5000)+"\n", 100)), func(string) { count++ })
	if err != nil || count != 100 || len(tail) > 24*4100 {
		t.Fatalf("unbounded log tail: size=%d count=%d err=%v", len(tail), count, err)
	}
}
