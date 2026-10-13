package ffmpegsetup

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/bodgit/sevenzip"
	"github.com/ulikunitz/xz/lzma"
)

// The verified Gyan full archive declares 192 MiB; keep a bounded 256 MiB ceiling.
const maxDictionary = 256 << 20
const maxSevenZipHeader = 16 << 20

// This package is the application's only 7z consumer. Install bounded decoders
// before NewReader, including its compressed-header path, can allocate a dictionary.
func init() {
	sevenzip.RegisterDecompressor([]byte{0x03, 0x01, 0x01}, boundedLZMA)
	sevenzip.RegisterDecompressor([]byte{0x21}, boundedLZMA2)
	// Optional codecs without a bounded adapter are explicitly unsupported.
	for _, method := range [][]byte{{0x03, 0x04, 0x01}, {0x04, 0xf7, 0x11, 0x01}, {0x04, 0xf7, 0x11, 0x02}, {0x06, 0xf1, 0x07, 0x01}} {
		sevenzip.RegisterDecompressor(method, func([]byte, uint64, []io.ReadCloser) (io.ReadCloser, error) {
			return nil, errors.New("7z 压缩算法不在受限支持范围（PPMd/Zstandard/Brotli/加密）；请改用 LZMA/LZMA2 或 ZIP")
		})
	}
}

type decodedCloser struct {
	io.Reader
	io.Closer
}

func boundedLZMA(properties []byte, size uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(properties) != 5 || len(readers) != 1 || size > uint64(maxExpanded) || binary.LittleEndian.Uint32(properties[1:]) > maxDictionary {
		return nil, errors.New("7z LZMA 字典或数据大小超出限制")
	}
	header := make([]byte, 13)
	copy(header, properties)
	binary.LittleEndian.PutUint64(header[5:], size)
	reader, err := (lzma.ReaderConfig{DictCap: maxDictionary}).NewReader(bufio.NewReader(io.MultiReader(bytes.NewReader(header), readers[0])))
	if err != nil {
		return nil, err
	}
	return &decodedCloser{Reader: reader, Closer: readers[0]}, nil
}

func boundedLZMA2(properties []byte, size uint64, readers []io.ReadCloser) (io.ReadCloser, error) {
	if len(properties) != 1 || len(readers) != 1 {
		return nil, errors.New("7z LZMA2 属性无效")
	}
	if properties[0] > 32 || size > uint64(maxExpanded) {
		return nil, fmt.Errorf("7z LZMA2 字典或数据大小超出限制（属性 %d；声明大小 %d）", properties[0], size)
	}
	dictionary := (2 | (int(properties[0]) & 1)) << (properties[0]/2 + 11)
	if dictionary > maxDictionary {
		return nil, errors.New("7z LZMA2 字典超出 256 MiB")
	}
	// A stream cannot reference more history than its complete unpacked size.
	// Avoid reserving 192 MiB for the publisher's small metadata stream.
	if size < uint64(dictionary) {
		dictionary = max(lzma.MinDictCap, int(size))
	}
	reader, err := (lzma.Reader2Config{DictCap: dictionary}).NewReader2(readers[0])
	if err != nil {
		return nil, err
	}
	return &decodedCloser{Reader: reader, Closer: readers[0]}, nil
}

func checkSevenZipHeader(reader io.ReaderAt, size int64) error {
	var header [32]byte
	if _, err := reader.ReadAt(header[:], 0); err != nil {
		return err
	}
	offset, length := binary.LittleEndian.Uint64(header[12:20]), binary.LittleEndian.Uint64(header[20:28])
	if size < 32 || length > maxSevenZipHeader || offset > uint64(size-32) || length > uint64(size-32)-offset {
		return errors.New("7z 元数据大小或位置超出限制")
	}
	return nil
}
