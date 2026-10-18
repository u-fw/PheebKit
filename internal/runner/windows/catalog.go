package windows

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"pheebkit/internal/core"
	"pheebkit/internal/sourcefile"
)

var catalogFlags = map[string]string{"encoders": "-encoders", "decoders": "-decoders", "filters": "-filters", "muxers": "-muxers", "demuxers": "-demuxers", "protocols": "-protocols", "bsfs": "-bsfs", "pixel-formats": "-pix_fmts", "sample-formats": "-sample_fmts", "layouts": "-layouts", "colors": "-colors", "hardware": "-hwaccels", "devices": "-devices"}
var catalogName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+\-]{0,79}$`)
var layoutName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+()\-]{0,79}$`)
var codecFlags = regexp.MustCompile(`^[VAS][A-Z.]{5,7}$`)
var filterFlags = regexp.MustCompile(`^[TSC.]{2,3}$`)
var formatFlags = regexp.MustCompile(`^[DE.d]{1,3}$`)
var pixelFlags = regexp.MustCompile(`^[IOHBP.]{5}$`)

func parseCatalog(category, raw string) []core.CatalogEntry {
	result := []core.CatalogEntry{}
	seen := map[string]bool{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, ":") || strings.Contains(line, " = ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		nameIndex := 0
		flags := ""
		switch category {
		case "encoders", "decoders":
			if !codecFlags.MatchString(fields[0]) {
				continue
			}
			nameIndex = 1
			flags = fields[0]
		case "filters":
			if !filterFlags.MatchString(fields[0]) {
				continue
			}
			nameIndex = 1
			flags = fields[0]
		case "muxers", "demuxers", "devices":
			if !formatFlags.MatchString(fields[0]) {
				continue
			}
			nameIndex = 1
			flags = fields[0]
		case "pixel-formats":
			if !pixelFlags.MatchString(fields[0]) {
				continue
			}
			nameIndex = 1
			flags = fields[0]
		case "sample-formats", "colors", "layouts":
			if len(fields) < 2 || fields[0] == "name" || fields[0] == "NAME" {
				continue
			}
		default:
			if len(fields) != 1 {
				continue
			}
		}
		if nameIndex >= len(fields) {
			continue
		}
		for _, name := range strings.Split(fields[nameIndex], ",") {
			valid := catalogName.MatchString(name) || (category == "layouts" && layoutName.MatchString(name))
			if !valid || seen[name] {
				continue
			}
			seen[name] = true
			description := strings.Join(fields[nameIndex+1:], " ")
			if len(description) > 2048 {
				description = description[:2048]
			}
			result = append(result, core.CatalogEntry{Name: name, Flags: flags, Description: description})
			if len(result) >= 20000 {
				return result
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (r *Runner) Catalog(parent context.Context, category string) ([]core.CatalogEntry, error) {
	flag, ok := catalogFlags[category]
	if !ok {
		return nil, fmt.Errorf("unsupported FFmpeg catalog category")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := r.lockInspector(ctx); err != nil {
		return nil, err
	}
	defer r.inspectMu.Unlock()
	identity, err := sourcefile.Fingerprint(r.FFmpeg)
	if err != nil {
		return nil, err
	}
	if identity != r.catalogIdentity {
		r.catalogs = nil
		r.catalogIdentity = identity
	}
	if values, ok := r.catalogs[category]; ok {
		return append([]core.CatalogEntry(nil), values...), nil
	}
	data, err := capture(ctx, r.FFmpeg, []string{"-hide_banner", flag}, false)
	if err != nil {
		return nil, err
	}
	if len(data) > 4*1024*1024 {
		return nil, fmt.Errorf("能力目录超过 4 MiB")
	}
	values := parseCatalog(category, string(data))
	if r.catalogs == nil {
		r.catalogs = map[string][]core.CatalogEntry{}
	}
	r.catalogs[category] = values
	return append([]core.CatalogEntry(nil), values...), nil
}
func (r *Runner) OptionHelp(parent context.Context, category, name string) (string, error) {
	argument := "full"
	if category != "full" {
		kind := map[string]string{"encoders": "encoder", "decoders": "decoder", "filters": "filter", "muxers": "muxer", "demuxers": "demuxer", "bsfs": "bsf", "protocols": "protocol"}[category]
		if kind == "" || !catalogName.MatchString(name) {
			return "", fmt.Errorf("请选择受支持的帮助类别和功能名称")
		}
		entries, err := r.Catalog(parent, category)
		if err != nil {
			return "", err
		}
		found := false
		for _, e := range entries {
			if e.Name == name {
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("当前 FFmpeg 目录中没有此功能")
		}
		argument = kind + "=" + name
	} else if name != "" {
		return "", fmt.Errorf("完整帮助不接受附加参数")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := r.lockInspector(ctx); err != nil {
		return "", err
	}
	defer r.inspectMu.Unlock()
	data, err := capture(ctx, r.FFmpeg, []string{"-hide_banner", "-h", argument}, false)
	if err != nil {
		return "", err
	}
	if len(data) > 4*1024*1024 {
		return "", fmt.Errorf("帮助内容超过 4 MiB")
	}
	return string(data), nil
}

func (r *Runner) lockInspector(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.inspectMu.TryLock() {
		return nil
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if r.inspectMu.TryLock() {
				return nil
			}
		}
	}
}
