package log

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TrimForLogOptions - config for TrimForLogs
type TrimForLogOptions struct {
	MaxStringLength      int
	MaxSliceLength       int
	SensitiveFields      []string
	SensitivePlaceholder string
}

var DefaultTrimForLogsOpts = TrimForLogOptions{
	MaxStringLength:      100,
	MaxSliceLength:       10,
	SensitivePlaceholder: "(SENSITIVE)",
}

// TrimForLogs marshals v to a generic JSON-like structure, truncates strings/slices exceeding
// configured limits, and replaces values of SensitiveFields with SensitivePlaceholder.
// Returns v as-is if it cannot be marshaled.
func TrimForLogs(v any, opts TrimForLogOptions) any {
	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return v
	}

	sensitive := make(map[string]struct{}, len(opts.SensitiveFields))
	for _, field := range opts.SensitiveFields {
		sensitive[strings.ToLower(field)] = struct{}{}
	}

	return trimValue(decoded, opts, sensitive)
}

func trimValue(v any, opts TrimForLogOptions, sensitive map[string]struct{}) any {
	switch val := v.(type) {
	case map[string]any:
		return trimMap(val, opts, sensitive)
	case []any:
		return trimSlice(val, opts, sensitive)
	case string:
		return trimString(val, opts.MaxStringLength)
	default:
		return val
	}
}

func trimMap(m map[string]any, opts TrimForLogOptions, sensitive map[string]struct{}) map[string]any {
	result := make(map[string]any, len(m))
	for key, value := range m {
		if _, ok := sensitive[strings.ToLower(key)]; ok {
			result[key] = opts.SensitivePlaceholder
			continue
		}
		result[key] = trimValue(value, opts, sensitive)
	}
	return result
}

func trimSlice(s []any, opts TrimForLogOptions, sensitive map[string]struct{}) []any {
	kept := s
	omitted := 0
	if opts.MaxSliceLength > 0 && len(s) > opts.MaxSliceLength {
		kept = s[:opts.MaxSliceLength]
		omitted = len(s) - opts.MaxSliceLength
	}

	result := make([]any, 0, len(kept)+1)
	for _, item := range kept {
		result = append(result, trimValue(item, opts, sensitive))
	}
	if omitted > 0 {
		result = append(result, fmt.Sprintf("... (%d more)", omitted))
	}
	return result
}

func trimString(s string, maxLength int) string {
	if maxLength > 0 && len(s) > maxLength {
		return s[:maxLength] + "...(truncated)"
	}
	return s
}
