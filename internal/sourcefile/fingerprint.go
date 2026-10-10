// Package sourcefile detects local media replacements between probing and use.
package sourcefile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrChanged = errors.New("源媒体文件已更改，请重新导入并解析后再开始压缩。")

// Fingerprint combines filesystem identity/metadata with bounded content samples.
// Reading at most 192 KiB keeps multi-gigabyte media checks inexpensive. Platform
// metadata includes change time where available, so edits outside the sampled
// regions are also detected without hashing the complete video on every preview.
func Fingerprint(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("无法读取源媒体文件，请重新导入：%w", err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return "", errors.New("源媒体文件不存在或不是普通文件，请重新导入。")
	}
	identity, err := platformIdentity(file, before)
	if err != nil {
		return "", fmt.Errorf("读取源媒体文件标识失败：%w", err)
	}
	hash := sha256.New()
	fmt.Fprintf(hash, "kompeg-source-v1\x00%d\x00%d\x00%s\x00", before.Size(), before.ModTime().UnixNano(), identity)
	const sampleSize int64 = 64 * 1024
	for _, offset := range []int64{0, max(0, before.Size()/2-sampleSize/2), max(0, before.Size()-sampleSize)} {
		count := min(sampleSize, before.Size()-offset)
		fmt.Fprintf(hash, "%d:%d\x00", offset, count)
		if _, err := io.CopyN(hash, io.NewSectionReader(file, offset, count), count); err != nil {
			return "", ErrChanged
		}
	}
	after, err := file.Stat()
	if err != nil {
		return "", ErrChanged
	}
	afterIdentity, err := platformIdentity(file, after)
	if err != nil || afterIdentity != identity || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return "", ErrChanged
	}
	// The pathname must still reference the opened file, rather than a replacement
	// that appeared while its predecessor was being sampled.
	current, err := os.Stat(path)
	if err != nil || !os.SameFile(before, current) || current.Size() != before.Size() || !current.ModTime().Equal(before.ModTime()) {
		return "", ErrChanged
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func Verify(path, expected string) error {
	if expected == "" {
		return nil // Compatibility for plans produced before source fingerprints.
	}
	actual, err := Fingerprint(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return ErrChanged
	}
	return nil
}
