package ffmpegsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"testing"
	"time"
)

// Replay a real publisher-hashed archive without repeatedly downloading it.
func TestRealPublisherSevenZipArchive(t *testing.T) {
	path := os.Getenv("PHEEBKIT_SEVENZIP_ARCHIVE")
	if path == "" {
		t.Skip("set PHEEBKIT_SEVENZIP_ARCHIVE to the verified Gyan 9.0.2 full archive")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	_, err = io.Copy(digest, file)
	file.Close()
	if err != nil || hex.EncodeToString(digest.Sum(nil)) != "f0e46253c70dfe902bac915dfb4224f0cf1b7c6eeab9da2ccc9a5581f9a71b13" {
		t.Fatal("publisher hash verification failed", err)
	}
	directory := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := extract(ctx, path, directory, Artifact{Format: "auto"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := findBinaries(directory); err != nil {
		t.Fatal(err)
	}
	t.Log("real publisher archive extracted with bounded decoders")
}
