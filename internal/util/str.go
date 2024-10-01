package util

import "strings"

func NormalizeStr(input string) string {
	var result string
	result = input

	result = strings.Join(strings.Fields(result), "")
	result = strings.ToLower(result)

	result = strings.ReplaceAll(result, "\u00a0", "")
	result = strings.ReplaceAll(result, "\u00A0", "")
	result = strings.ReplaceAll(result, "&nbsp;", "")
	result = strings.ReplaceAll(result, "&#160;", "")

	return result
}
