package compiler

import (
	"fmt"
	"strconv"
	"strings"
)

type parameterValidator func(string) bool

// These are codec-library options, not ordinary FFmpeg argv. FFmpeg's -n does
// not protect their side files: x265 analysis-save and x264 pass=1 can write
// files themselves. Keep a positive list of purely in-memory encoding controls,
// with exact key spelling and value syntax. New options require checking the
// upstream library parser for input, output, and implicit multi-pass file I/O.
// Sources: FFmpeg libx264.c/libx265.c/libsvtav1.c; x265's CLI documentation;
// SVT-AV1 Docs/Parameters.md. Rate-control mode and source geometry stay managed
// by the project settings and are intentionally absent from this list.
var safeCodecParameters = map[string]map[string]parameterValidator{
	"-x264-params": {
		"aq-mode":          integerParameter(0, 3),
		"aq-strength":      decimalParameter(0, 3),
		"psy-rd":           oneOrPairParameter(decimalParameter(0, 10)),
		"psy":              booleanParameter,
		"deblock":          oneOrPairParameter(integerParameter(-6, 6)),
		"subme":            integerParameter(0, 11),
		"trellis":          integerParameter(0, 2),
		"ref":              integerParameter(1, 16),
		"me":               enumParameter("dia", "hex", "umh", "esa", "tesa"),
		"rc-lookahead":     integerParameter(0, 250),
		"keyint":           integerParameter(1, 2_000_000),
		"min-keyint":       integerParameter(1, 2_000_000),
		"bframes":          integerParameter(0, 16),
		"b-adapt":          integerParameter(0, 2),
		"b-pyramid":        enumParameter("none", "strict", "normal", "0", "1", "2"),
		"scenecut":         integerParameter(0, 100),
		"weightp":          integerParameter(0, 2),
		"weightb":          booleanParameter,
		"open-gop":         booleanParameter,
		"mbtree":           booleanParameter,
		"chroma-qp-offset": integerParameter(-12, 12),
	},
	"-x265-params": {
		"aq-mode":                integerParameter(0, 4),
		"aq-strength":            decimalParameter(0, 3),
		"psy-rd":                 decimalParameter(0, 5),
		"psy-rdoq":               decimalParameter(0, 50),
		"rd":                     integerParameter(1, 6),
		"rdoq-level":             integerParameter(0, 2),
		"subme":                  integerParameter(0, 7),
		"ref":                    integerParameter(1, 16),
		"rc-lookahead":           integerParameter(0, 250),
		"keyint":                 integerParameter(1, 2_000_000),
		"min-keyint":             integerParameter(1, 2_000_000),
		"bframes":                integerParameter(0, 16),
		"b-adapt":                integerParameter(0, 2),
		"scenecut":               integerParameter(0, 100),
		"rect":                   booleanParameter,
		"amp":                    booleanParameter,
		"sao":                    booleanParameter,
		"strong-intra-smoothing": booleanParameter,
		"cutree":                 booleanParameter,
		"weightp":                booleanParameter,
		"weightb":                booleanParameter,
		"open-gop":               booleanParameter,
		"deblock":                oneOrPairParameter(integerParameter(-6, 6)),
		"log-level":              enumParameter("none", "error", "warning", "info", "debug", "full", "-1", "0", "1", "2", "3", "4"),
	},
	"-svtav1-params": {
		"aq-mode":            integerParameter(0, 2),
		"tune":               integerParameter(0, 4),
		"film-grain":         integerParameter(0, 50),
		"film-grain-denoise": booleanParameter,
		"enable-tf":          booleanParameter,
		"enable-overlays":    booleanParameter,
		"enable-qm":          booleanParameter,
		"qm-min":             integerParameter(0, 15),
		"qm-max":             integerParameter(0, 15),
	},
}

func validateCodecParameters(option, value string, rules map[string]parameterValidator) error {
	if len(value) > 4096 || strings.Count(value, ":") >= 32 {
		return fmt.Errorf("高级参数 %s 仅支持最多 32 个安全编码子选项（总长不超过 4096 字节）。", option)
	}
	parameters := strings.Split(value, ":")
	seen := make(map[string]bool, len(parameters))
	for _, parameter := range parameters {
		key, argument, hasValue := strings.Cut(parameter, "=")
		validate, allowed := rules[key]
		if !allowed {
			return fmt.Errorf("高级参数 %s 的子选项 %q 未获支持；文件读写、多遍编码和未知选项不可使用。", option, key)
		}
		if !hasValue || !validate(argument) {
			return fmt.Errorf("高级参数 %s 的子选项 %s 值无效；请使用明确的 key=value，数值使用十进制，双值使用逗号。", option, key)
		}
		if seen[key] {
			return fmt.Errorf("高级参数 %s 的子选项 %s 重复；请只保留一个值。", option, key)
		}
		seen[key] = true
	}
	return nil
}

func booleanParameter(value string) bool { return value == "0" || value == "1" }

func integerParameter(minimum, maximum int) parameterValidator {
	return func(value string) bool {
		if !decimalSyntax(value, false) {
			return false
		}
		number, err := strconv.Atoi(value)
		return err == nil && number >= minimum && number <= maximum
	}
}

func decimalParameter(minimum, maximum float64) parameterValidator {
	return func(value string) bool {
		if !decimalSyntax(value, true) {
			return false
		}
		number, err := strconv.ParseFloat(value, 64)
		return err == nil && number >= minimum && number <= maximum
	}
}

// Reject quoting/escaping, whitespace, exponent notation, Unicode lookalikes,
// NaN and trailing material that upstream atoi/atof might silently ignore.
func decimalSyntax(value string, allowDot bool) bool {
	if strings.HasPrefix(value, "-") {
		value = value[1:]
	}
	if value == "" {
		return false
	}
	digits, dots := 0, 0
	for _, character := range value {
		if character >= '0' && character <= '9' {
			digits++
		} else if character == '.' && allowDot {
			dots++
		} else {
			return false
		}
	}
	return digits > 0 && dots <= 1
}

func enumParameter(values ...string) parameterValidator {
	return func(value string) bool {
		for _, allowed := range values {
			if value == allowed {
				return true
			}
		}
		return false
	}
}

func oneOrPairParameter(validate parameterValidator) parameterValidator {
	return func(value string) bool {
		parts := strings.Split(value, ",")
		return (len(parts) == 1 || len(parts) == 2) && validate(parts[0]) && (len(parts) == 1 || validate(parts[1]))
	}
}
