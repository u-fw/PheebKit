package windows

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.Len()
	if n > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	_, err := b.Buffer.Write(p)
	return n, err
}

func capture(ctx context.Context, binary string, args []string, combined bool) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	hideProcess(cmd)
	stdout := &boundedBuffer{limit: 32 * 1024 * 1024}
	stderr := &boundedBuffer{limit: 64 * 1024}
	cmd.Stdout = stdout
	if combined {
		cmd.Stderr = stdout
	} else {
		cmd.Stderr = stderr
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%s: %w: %s", binary, err, strings.TrimSpace(stderr.String()))
	}
	if stdout.overflow {
		return nil, fmt.Errorf("FFmpeg response exceeds 32 MiB safety limit")
	}
	return stdout.Bytes(), nil
}
