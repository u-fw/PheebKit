// Package sourcefile detects local media replacements between probing and use.
package sourcefile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"pheebkit/internal/compat"
)

var ErrChanged = errors.New("源媒体文件已更改，请重新导入并解析后再开始压缩。")

// Fingerprint combines filesystem identity/metadata with bounded content samples.
// Files up to 4 MiB are hashed in full: some filesystems provide coarse change
// timestamps even when the write time is restored. Larger files retain bounded
// 192 KiB sampling plus identity/change metadata. This is not a complete content
// integrity guarantee for large media on filesystems with weak timestamps.
func Fingerprint(path string) (string, error) {
	return fingerprint(path, false)
}

func fingerprint(path string, legacy bool) (string, error) {
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
	version := compat.FingerprintDomain(legacy)
	fmt.Fprintf(hash, "%s\x00%d\x00%d\x00%s\x00", version, before.Size(), before.ModTime().UnixNano(), identity)
	if err := writeContentSamples(hash, file, before.Size(), legacy); err != nil {
		return "", ErrChanged
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
		// Persisted pre-v2 references retain their original comparison semantics.
		// Every fresh probe and new execution plan receives the stronger v2 hash.
		if len(expected) == sha256.Size*2 {
			if _, err := hex.DecodeString(expected); err == nil {
				if previous, err := fingerprint(path, true); err == nil && previous == expected {
					return nil
				}
			}
		}
		return ErrChanged
	}
	return nil
}

const fullHashLimit int64 = 4 * 1024 * 1024

func writeContentSamples(hash io.Writer, file io.ReaderAt, size int64, legacy bool) error {
	if !legacy && size <= fullHashLimit {
		fmt.Fprintf(hash, "full:%d\x00", size)
		_, err := io.CopyN(hash, io.NewSectionReader(file, 0, size), size)
		return err
	}
	const sampleSize int64 = 64 * 1024
	for _, offset := range []int64{0, max(0, size/2-sampleSize/2), max(0, size-sampleSize)} {
		count := min(sampleSize, size-offset)
		fmt.Fprintf(hash, "%d:%d\x00", offset, count)
		if _, err := io.CopyN(hash, io.NewSectionReader(file, offset, count), count); err != nil {
			return err
		}
	}
	return nil
}
