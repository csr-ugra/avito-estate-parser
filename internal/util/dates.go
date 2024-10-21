package util

import "time"

// LastDayOfMonth returns the last day of the specified month and year.
func LastDayOfMonth(t time.Time) time.Time {
	nextMonth := t.Month() + 1
	year := t.Year()
	if nextMonth > 12 {
		nextMonth = 1
		year++
	}

	firstOfNextMonth := time.Date(year, nextMonth, 1, 0, 0, 0, 0, time.UTC)

	lastDay := firstOfNextMonth.AddDate(0, 0, -1)

	return lastDay
}

// MonthString returns the Russian name of the month ("Январь", "Февраль", ...).
func MonthString(t time.Time) string {
	m := map[time.Month]string{
		time.January:   "Январь",
		time.February:  "Февраль",
		time.March:     "Март",
		time.April:     "Апрель",
		time.May:       "Май",
		time.June:      "Июнь",
		time.July:      "Июль",
		time.August:    "Август",
		time.September: "Сентябрь",
		time.October:   "Октябрь",
		time.November:  "Ноябрь",
		time.December:  "Декабрь",
	}

	return m[t.Month()]
}
