package valueTypes

import "strings"

func normalizeScalarString(value string) string {
	return strings.TrimSpace(value)
}

func isEmptyScalarString(value string) bool {
	value = normalizeScalarString(value)
	return value == "" || value == "--" || strings.EqualFold(value, "null")
}
