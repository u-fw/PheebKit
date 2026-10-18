// Package locales supplies application-owned native dialog translations.
// The same catalog is imported by React; tool output and user data are not keys.
package locales

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
)

//go:embed en.json
var englishJSON []byte

var english = func() map[string]string {
	var messages map[string]string
	if err := json.Unmarshal(englishJSON, &messages); err != nil {
		panic("invalid application translation catalog: " + err.Error())
	}
	return messages
}()

var prefixes = func() []string {
	var keys []string
	for key := range english {
		if strings.HasSuffix(key, "：") || strings.HasSuffix(key, ":") {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	return keys
}()

func Translate(message, language string) string {
	if language == "en-US" {
		if translated, ok := english[message]; ok {
			return translated
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(message, prefix) {
				return english[prefix] + strings.TrimPrefix(message, prefix)
			}
		}
	}
	return message
}
