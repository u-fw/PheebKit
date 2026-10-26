package ffmpegsetup

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
	"github.com/ulikunitz/xz"
)

// All decoders share the same path, entry-count and expanded-size checks.
// Files are created exclusively; links and device files are never materialized.
type archiveOutput struct {
	ctx       context.Context
	directory string
	count     int
	total     int64
}

func (out *archiveOutput) write(name string, mode os.FileMode, size int64, reader io.Reader) error {
	if err := out.ctx.Err(); err != nil {
		return err
	}
	out.count++
	if out.count > 10000 {
		return errors.New("压缩包中文件过多")
	}
	if !filepath.IsLocal(filepath.FromSlash(name)) || strings.ContainsAny(name, "\\:") {
		return errors.New("压缩包包含不安全路径")
	}
	for _, component := range strings.Split(strings.TrimSuffix(name, "/"), "/") {
		// Windows normalizes trailing dots/spaces. Reject ambiguous aliases
		// before any filesystem calls, while preserving normal TAR ./ roots.
		if component != "." && (component == ".." || strings.TrimRight(component, " .") != component) {
			return errors.New("压缩包包含 Windows 路径别名（末尾点或空格）")
		}
	}
	if mode.Type() != 0 && mode.Type() != os.ModeDir {
		return errors.New("压缩包包含链接或特殊文件")
	}
	if size < 0 || size > maxExpanded-out.total {
		return errors.New("解压总大小超出限制")
	}
	out.total += size
	target := filepath.Join(out.directory, filepath.FromSlash(name))
	if mode.IsDir() {
		return os.MkdirAll(target, 0700)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	return copyBounded(out.ctx, target, reader, size)
}

func extract(ctx context.Context, archive, destination string, artifact Artifact) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	if artifact.Format == "plain" || artifact.Format == "gzip" {
		var reader io.Reader = file
		if artifact.Format == "gzip" {
			zipped, err := gzip.NewReader(file)
			if err != nil {
				return err
			}
			defer zipped.Close()
			reader = zipped
		}
		return copyBounded(ctx, filepath.Join(destination, artifact.Name), reader, maxExpanded)
	}
	if artifact.Format != "auto" && artifact.Format != "zip" {
		return errors.New("未知压缩格式")
	}
	input := bufio.NewReader(&contextReader{ctx: ctx, reader: file})
	magic, _ := input.Peek(8)
	out := &archiveOutput{ctx: ctx, directory: destination}
	if artifact.Format == "zip" || bytes.HasPrefix(magic, []byte("PK")) {
		stat, err := file.Stat()
		if err != nil {
			return err
		}
		reader, err := zip.NewReader(&contextReaderAt{ctx: ctx, reader: file}, stat.Size())
		if err != nil {
			return errors.New("下载内容不是有效的 ZIP 压缩包。请填写文件直链，不要填写下载页面或 Git 仓库地址。")
		}
		if len(reader.File) > 10000 {
			return errors.New("压缩包中文件过多")
		}
		for _, entry := range reader.File {
			if entry.UncompressedSize64 > uint64(maxExpanded) {
				return errors.New("解压文件超出大小限制")
			}
			content, err := entry.Open()
			if err != nil {
				return err
			}
			err = out.write(entry.Name, entry.Mode(), int64(entry.UncompressedSize64), content)
			closeErr := content.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		}
		return nil
	}
	if bytes.HasPrefix(magic, []byte{'7', 'z', 0xbc, 0xaf, 0x27, 0x1c}) {
		stat, err := file.Stat()
		if err != nil {
			return err
		}
		if err := checkSevenZipHeader(file, stat.Size()); err != nil {
			return err
		}
		reader, err := sevenzip.NewReader(&contextReaderAt{ctx: ctx, reader: file}, stat.Size())
		if err != nil {
			return err
		}
		if len(reader.File) > 10000 {
			return errors.New("压缩包中文件过多")
		}
		for _, entry := range reader.File {
			if entry.UncompressedSize > uint64(maxExpanded) {
				return errors.New("解压文件超出大小限制")
			}
			if entry.FileInfo().IsDir() {
				if err := out.write(entry.Name, entry.Mode(), 0, nil); err != nil {
					return err
				}
				continue
			}
			content, err := entry.Open()
			if err != nil {
				return err
			}
			err = out.write(entry.Name, entry.Mode(), int64(entry.UncompressedSize), content)
			closeErr := content.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		}
		return nil
	}
	if bytes.HasPrefix(magic, []byte("Rar!\x1a\x07")) {
		reader, err := rardecode.NewReader(input, rardecode.MaxDictionarySize(128<<20))
		if err != nil {
			return err
		}
		for {
			entry, err := reader.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if entry.Encrypted || entry.HeaderEncrypted || entry.LinkType != rardecode.LinkTypeNone || entry.UnKnownSize {
				return errors.New("不支持加密、链接或大小未知的 RAR 文件")
			}
			if err := out.write(entry.Name, entry.Mode(), entry.UnPackedSize, reader); err != nil {
				return err
			}
		}
	}
	var decoded io.Reader = input
	switch {
	case bytes.HasPrefix(magic, []byte{0x1f, 0x8b}):
		reader, err := gzip.NewReader(input)
		if err != nil {
			return err
		}
		defer reader.Close()
		decoded = reader
	case bytes.HasPrefix(magic, []byte{0xfd, '7', 'z', 'X', 'Z', 0}):
		reader, err := (xz.ReaderConfig{DictCap: 128 << 20}).NewReader(input)
		if err != nil {
			return err
		}
		decoded = reader
	case bytes.HasPrefix(magic, []byte("BZh")):
		decoded = bzip2.NewReader(input)
	}
	// Include a bounded allowance for TAR metadata in addition to payload bytes.
	reader := tar.NewReader(io.LimitReader(&contextReader{ctx: ctx, reader: decoded}, maxExpanded+(16<<20)))
	for {
		entry, err := reader.Next()
		if err == io.EOF {
			if out.count == 0 {
				return errors.New("压缩包为空或格式不受支持")
			}
			return nil
		}
		if err != nil {
			return errors.New("下载内容不是有效的压缩包，请使用 ZIP、7z、RAR 或 TAR 压缩包的文件直链。加密、分卷和自解压包不受支持。")
		}
		if entry.Typeflag != tar.TypeReg && entry.Typeflag != tar.TypeDir {
			return errors.New("压缩包包含链接或特殊文件")
		}
		if err := out.write(entry.Name, entry.FileInfo().Mode(), entry.Size, reader); err != nil {
			return err
		}
	}
}

type contextReaderAt struct {
	ctx    context.Context
	reader io.ReaderAt
}

func (r *contextReaderAt) ReadAt(p []byte, offset int64) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.ReadAt(p, offset)
}
