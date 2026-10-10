package compiler

import (
	"fmt"
	"strings"
)

// Preview emits a PowerShell command for copying. Execution always uses Args.
func Preview(binary string, args []string) string {
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	parts := []string{"&", quote(binary)}
	for _, argument := range args {
		parts = append(parts, quote(argument))
	}
	return strings.Join(parts, " ")
}

// The spike exposes advanced codec options while retaining ownership of input,
// output, mapping, progress, filters and overwrite policy. Full raw plans can be
// added as a separate validated project type after this execution boundary works.
func validateRaw(args []string) error {
	allowed := map[string]bool{"-profile:v": true, "-level:v": true, "-g": true, "-bf": true,
		"-maxrate": true, "-bufsize": true, "-x264-params": true, "-x265-params": true,
		"-svtav1-params": true, "-metadata": true}
	if len(args)%2 != 0 || len(args) > 64 {
		return fmt.Errorf("advanced arguments require at most 32 option/value pairs")
	}
	for i := 0; i < len(args); i += 2 {
		if !allowed[args[i]] {
			return fmt.Errorf("advanced option %q is unavailable in this spike", args[i])
		}
		if strings.ContainsRune(args[i+1], 0) || strings.HasPrefix(args[i+1], "-") || args[i+1] == "" {
			return fmt.Errorf("advanced option %s requires a nonempty value", args[i])
		}
		if rules, codecParameters := safeCodecParameters[args[i]]; codecParameters {
			if err := validateCodecParameters(args[i], args[i+1], rules); err != nil {
				return err
			}
		}
	}
	return nil
}
