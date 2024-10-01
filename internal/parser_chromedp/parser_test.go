package parser_chromedp

import (
	"testing"
	"time"
)

func FuzzTest_indexOfTheWeekInMonth(f *testing.F) {
	// seed corpus entries
	f.Add(time.Date(1999, 12, 31, 5, 12, 4, 0, time.UTC).Unix())
	f.Add(time.Date(2014, 2, 1, 6, 56, 30, 0, time.UTC).Unix())
	f.Add(time.Date(2024, 9, 26, 0, 0, 0, 0, time.UTC).Unix())
	f.Add(time.Date(2024, 9, 26, 15, 41, 2, 0, time.UTC).Unix())
	f.Add(time.Date(2024, 9, 30, 15, 41, 2, 0, time.UTC).Unix())
	f.Add(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	f.Add(time.Date(2039, 4, 13, 9, 24, 0, 0, time.UTC).Unix())
	f.Add(time.Now().Unix())

	const maxWeekIndex = 5

	f.Fuzz(func(t *testing.T, input int64) {
		if got := indexOfTheWeekInMonth(time.Unix(input, 0)); got > maxWeekIndex {
			t.Errorf("indexOfTheWeekInMonth() = %v, want %v", got, maxWeekIndex)
		}
	})
}
