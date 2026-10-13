package ffmpegsetup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveRejectsDotSpaceAliasesBeforeWriting(t *testing.T) {
	parent := t.TempDir()
	destination := filepath.Join(parent, "extract")
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"safe/.. /.. /canary", "safe./tool.exe", "safe /tool.exe", "bin/tool.exe.", "bin/tool.exe "} {
		out := archiveOutput{ctx: context.Background(), directory: destination}
		if err := out.write(name, 0, 1, strings.NewReader("x")); err == nil {
			t.Errorf("ambiguous archive path accepted: %q", name)
		}
	}
	if _, err := os.Stat(filepath.Join(parent, "canary")); !os.IsNotExist(err) {
		t.Fatal("archive wrote outside its extraction root")
	}
}
