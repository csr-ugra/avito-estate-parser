package selector

import (
	"fmt"
	"time"
)

const (
	ModalDialog         = "div[aria-modal=\"true\"][role=\"dialog\"][tabindex=\"-1\"]"
	ModalDialogCloseBtn = "button[type=\"button\"][aria-label=\"закрыть\"]"
	PageTitleCount      = "span[data-marker=\"page-title/count\"]"
	PageTitleText       = "h1"
	SubmitFiltersBtn    = "button[data-marker=\"search-filters/submit-button\"]"
)

func CalendarBtn(t *time.Time) string {
	const calendarButtonSelectorTemplate = "td[data-marker^=\"params[\"][data-marker$=\"/day(%d)\"] div[role=button][class*=\"styles-module-day_hoverable-\"]"
	return fmt.Sprintf(calendarButtonSelectorTemplate, t.Day())
}
