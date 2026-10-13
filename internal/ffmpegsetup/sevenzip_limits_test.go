package ffmpegsetup

import (
	"bytes"
	"encoding/binary"
	"github.com/ulikunitz/xz/lzma"
	"io"
	"testing"
)

func TestSevenZipRejectsOversizedDictionaryBeforeDecode(t *testing.T) {
	input := []io.ReadCloser{io.NopCloser(bytes.NewReader(nil))}
	for _, property := range []byte{33, 40, 255} {
		if _, err := boundedLZMA2([]byte{property}, 1, input); err == nil {
			t.Fatal("unbounded LZMA2 dictionary accepted")
		}
	}
	properties := make([]byte, 5)
	binary.LittleEndian.PutUint32(properties[1:], maxDictionary+1)
	if _, err := boundedLZMA(properties, 1, input); err == nil {
		t.Fatal("unbounded LZMA dictionary accepted")
	}
}
func TestBoundedSevenZipLZMAStreamsRemainCompatible(t *testing.T) {
	data := bytes.Repeat([]byte("ffmpeg archive payload"), 64)
	var classic, second bytes.Buffer
	writer, err := (lzma.WriterConfig{DictCap: 1 << 20, Size: int64(len(data)), SizeInHeader: true}).NewWriter(&classic)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	raw := classic.Bytes()
	first, err := boundedLZMA(raw[:5], uint64(len(data)), []io.ReadCloser{io.NopCloser(bytes.NewReader(raw[13:]))})
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(first)
	first.Close()
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("LZMA round trip failed", err)
	}
	writer2, err := (lzma.Writer2Config{DictCap: 1 << 20}).NewWriter2(&second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = writer2.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = writer2.Close(); err != nil {
		t.Fatal(err)
	}
	decoded, err := boundedLZMA2([]byte{16}, uint64(len(data)), []io.ReadCloser{io.NopCloser(bytes.NewReader(second.Bytes()))})
	if err != nil {
		t.Fatal(err)
	}
	got, err = io.ReadAll(decoded)
	decoded.Close()
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("LZMA2 round trip failed", err)
	}
}
func TestSevenZipHeaderLimitsRejectOverflow(t *testing.T) {
	var header [32]byte
	binary.LittleEndian.PutUint64(header[12:20], ^uint64(0))
	if checkSevenZipHeader(bytes.NewReader(header[:]), 32) == nil {
		t.Fatal("header offset overflow accepted")
	}
	binary.LittleEndian.PutUint64(header[12:20], 0)
	binary.LittleEndian.PutUint64(header[20:28], maxSevenZipHeader+1)
	if checkSevenZipHeader(bytes.NewReader(header[:]), 64) == nil {
		t.Fatal("oversized metadata accepted")
	}
}
