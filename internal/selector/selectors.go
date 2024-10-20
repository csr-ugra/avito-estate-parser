package selector

import (
	"fmt"
	"time"
)

type Selector string

func (s Selector) String() string {
	return string(s)
}

const (
	ModalDialog                   Selector = "div[aria-modal=\"true\"][role=\"dialog\"][tabindex=\"-1\"]"
	ModalDialogCloseBtn           Selector = "button[type=\"button\"][aria-label=\"закрыть\"]"
	PageTitleCount                Selector = "span[data-marker=\"page-title/count\"]"
	PageTitleText                 Selector = "h1"
	SubmitFiltersBtn              Selector = "button[data-marker=\"search-filters/submit-button\"]"
	LocationChangeButton                   = "div[data-marker=\"search-form/change-location\"]"
	EstateTypeFilterButton                 = "input[data-marker=\"categoryId\"]"
	EstateTypeFilterDropdown               = "div[class^=\"dropdown-list-dropdown-list\"]"
	EstateActionFilterButton               = "input[data-marker=\"param[201]\"]"
	EstateDurationDailyRentButton          = "input[data-marker=\"param[528](5477)/input\"]"
	EstateSubmitButton                     = "a[data-marker=\"search-form-widget/action-button-0\"]"
)

func CalendarBtn(t *time.Time) Selector {
	const calendarButtonSelectorTemplate = "td[data-marker^=\"params[\"][data-marker$=\"/day(%d)\"] div[role=button][class*=\"styles-module-day_hoverable-\"]"
	return Selector(fmt.Sprintf(calendarButtonSelectorTemplate, t.Day()))
}
