package bundledffmpeg

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	maxPayloadBytes = int64(400 << 20)
	maxExpanded     = int64(1 << 30)
	maxFiles        = 10000
)

var (
	// PayloadSHA256 is supplied by the reviewed release build when the
	// pheebkit_bundled_ffmpeg tag is enabled.
	PayloadSHA256 = ""
	// DistributionVersion identifies the publisher-pinned FFmpeg package.
	DistributionVersion = ""
)

var (
	ErrUnavailable = errors.New("this build does not include FFmpeg")
	safeVersion    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	safeDigest     = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type Metadata struct {
	SchemaVersion       int    `json:"schemaVersion"`
	Provider            string `json:"provider"`
	Package             string `json:"package"`
	Version             string `json:"version"`
	VersionLine         string `json:"versionLine"`
	License             string `json:"license"`
	CorrespondingSource string `json:"correspondingSource"`
	ArchiveURL          string `json:"archiveURL"`
	ArchiveSHA256       string `json:"archiveSHA256"`
	FFmpegSHA256        string `json:"ffmpegSHA256"`
	FFprobeSHA256       string `json:"ffprobeSHA256"`
	PackagedAt          string `json:"packagedAt,omitempty"`
}

type Runtime struct {
	Directory     string   `json:"directory"`
	FFmpeg        string   `json:"ffmpeg"`
	FFprobe       string   `json:"ffprobe"`
	PayloadSHA256 string   `json:"payloadSHA256"`
	Metadata      Metadata `json:"metadata"`
}

type BuildDescription struct {
	Kind                   string `json:"kind"`
	PayloadSHA256          string `json:"payloadSHA256,omitempty"`
	DistributionVersion    string `json:"distributionVersion,omitempty"`
	ExtractedAtFirstLaunch bool   `json:"extractedAtFirstLaunch,omitempty"`
}

var (
	resolveMu sync.Mutex
	resolved  *Runtime
)

// DescribeBuild is side-effect free and can be emitted by --build-info.
func DescribeBuild() BuildDescription {
	if len(embeddedPayload()) == 0 {
		return BuildDescription{Kind: "external"}
	}
	return BuildDescription{
		Kind:                   "ffmpeg",
		PayloadSHA256:          PayloadSHA256,
		DistributionVersion:    DistributionVersion,
		ExtractedAtFirstLaunch: true,
	}
}

// Resolve validates and atomically materializes the trusted embedded payload.
// A successful extraction is reused across launches and application updates
// that contain the same payload.
func Resolve() (Runtime, error) {
	payload := embeddedPayload()
	if len(payload) == 0 {
		return Runtime{}, ErrUnavailable
	}
	resolveMu.Lock()
	defer resolveMu.Unlock()
	if resolved != nil {
		return *resolved, nil
	}
	expected := strings.ToLower(strings.TrimSpace(PayloadSHA256))
	if !safeDigest.MatchString(expected) {
		return Runtime{}, errors.New("bundled FFmpeg build is missing its payload digest")
	}
	version := strings.TrimSpace(DistributionVersion)
	if !safeVersion.MatchString(version) {
		return Runtime{}, errors.New("bundled FFmpeg build has an invalid distribution version")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return Runtime{}, fmt.Errorf("locate user cache for bundled FFmpeg: %w", err)
	}
	root := filepath.Join(cache, "PheebKit", "runtime", "ffmpeg")
	runtime, err := preparePayload(payload, expected, version, root)
	if err != nil {
		return Runtime{}, err
	}
	resolved = &runtime
	return runtime, nil
}

func preparePayload(payload []byte, expected, version, root string) (Runtime, error) {
	if int64(len(payload)) > maxPayloadBytes {
		return Runtime{}, errors.New("bundled FFmpeg payload exceeds the size limit")
	}
	sum := sha256.Sum256(payload)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return Runtime{}, fmt.Errorf("bundled FFmpeg payload digest mismatch: got %s, want %s", actual, expected)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return Runtime{}, fmt.Errorf("create bundled FFmpeg cache: %w", err)
	}
	final := filepath.Join(root, "ffmpeg-"+version+"-"+actual[:12])
	if runtime, err := validateRuntime(final, actual); err == nil {
		return runtime, nil
	} else if removeErr := removeInvalidRuntime(root, final); removeErr != nil {
		return Runtime{}, fmt.Errorf("replace invalid bundled FFmpeg cache: %w", removeErr)
	}
	stage, err := os.MkdirTemp(root, ".extract-")
	if err != nil {
		return Runtime{}, fmt.Errorf("create bundled FFmpeg staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	if err := extractPayload(payload, stage); err != nil {
		return Runtime{}, err
	}
	if _, err := validateRuntime(stage, actual); err != nil {
		return Runtime{}, err
	} else if err := os.Rename(stage, final); err != nil {
		// Another app process may have completed the same extraction first.
		if concurrent, validateErr := validateRuntime(final, actual); validateErr == nil {
			return concurrent, nil
		}
		return Runtime{}, fmt.Errorf("publish bundled FFmpeg runtime: %w", err)
	}
	return validateRuntime(final, actual)
}

func removeInvalidRuntime(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || strings.ContainsRune(relative, filepath.Separator) {
		return errors.New("refusing to replace a runtime outside the direct cache directory")
	}
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("refusing to replace a non-directory runtime cache entry")
	}
	return os.RemoveAll(target)
}

func extractPayload(payload []byte, destination string) error {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return errors.New("bundled FFmpeg payload is not a valid ZIP archive")
	}
	if len(reader.File) == 0 || len(reader.File) > maxFiles {
		return errors.New("bundled FFmpeg payload has an invalid entry count")
	}
	seen := make(map[string]struct{}, len(reader.File))
	var total int64
	for _, entry := range reader.File {
		name := entry.Name
		if name == "" || strings.ContainsAny(name, "\\:") || !filepath.IsLocal(filepath.FromSlash(name)) {
			return fmt.Errorf("bundled FFmpeg payload contains an unsafe path: %q", name)
		}
		for _, component := range strings.Split(strings.TrimSuffix(name, "/"), "/") {
			if component == "" || component == ".." || strings.TrimRight(component, " .") != component {
				return fmt.Errorf("bundled FFmpeg payload contains an unsafe path: %q", name)
			}
		}
		key := strings.ToLower(strings.ReplaceAll(name, "/", `\`))
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("bundled FFmpeg payload contains a duplicate path: %q", name)
		}
		seen[key] = struct{}{}
		mode := entry.Mode()
		if mode.Type() != 0 && mode.Type() != os.ModeDir {
			return fmt.Errorf("bundled FFmpeg payload contains a special file: %q", name)
		}
		if entry.UncompressedSize64 > uint64(maxExpanded-total) {
			return errors.New("bundled FFmpeg payload exceeds the expanded-size limit")
		}
		total += int64(entry.UncompressedSize64)
		target := filepath.Join(destination, filepath.FromSlash(name))
		if mode.IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if err != nil {
			input.Close()
			return err
		}
		written, copyErr := io.Copy(output, io.LimitReader(input, int64(entry.UncompressedSize64)+1))
		closeOutputErr := output.Close()
		closeInputErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOutputErr != nil {
			return closeOutputErr
		}
		if closeInputErr != nil {
			return closeInputErr
		}
		if written != int64(entry.UncompressedSize64) {
			return fmt.Errorf("bundled FFmpeg payload has an invalid size for %q", name)
		}
	}
	return nil
}

func validateRuntime(directory, payloadDigest string) (Runtime, error) {
	data, err := os.ReadFile(filepath.Join(directory, "BUNDLED_FFMPEG.json"))
	if err != nil {
		return Runtime{}, fmt.Errorf("read bundled FFmpeg metadata: %w", err)
	}
	var metadata Metadata
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return Runtime{}, fmt.Errorf("parse bundled FFmpeg metadata: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Runtime{}, errors.New("bundled FFmpeg metadata contains trailing data")
	}
	if metadata.SchemaVersion != 1 || metadata.Version == "" || metadata.FFmpegSHA256 == "" || metadata.FFprobeSHA256 == "" {
		return Runtime{}, errors.New("bundled FFmpeg metadata is incomplete")
	}
	for name, expected := range map[string]string{"ffmpeg": metadata.FFmpegSHA256, "ffprobe": metadata.FFprobeSHA256} {
		path := filepath.Join(directory, "bin", name+".exe")
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return Runtime{}, fmt.Errorf("bundled %s executable is missing", name)
		}
		hash, err := fileDigest(path)
		if err != nil {
			return Runtime{}, err
		}
		if !strings.EqualFold(hash, expected) {
			return Runtime{}, fmt.Errorf("bundled %s executable digest mismatch", name)
		}
	}
	return Runtime{
		Directory:     directory,
		FFmpeg:        filepath.Join(directory, "bin", "ffmpeg.exe"),
		FFprobe:       filepath.Join(directory, "bin", "ffprobe.exe"),
		PayloadSHA256: payloadDigest,
		Metadata:      metadata,
	}, nil
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
