package util

import (
	"github.com/avito-tech/normalize"
	"strings"
)

func Normalize(str string) string {
	result := str

	result = strings.Join(strings.Fields(result), "")
	result = normalize.Normalize(result)

	result = strings.ReplaceAll(result, "\u00a0", "")
	result = strings.ReplaceAll(result, "\u00A0", "")
	result = strings.ReplaceAll(result, "&nbsp;", "")
	result = strings.ReplaceAll(result, "&#160;", "")

	return result
}
