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
	ModalDialog                                Selector = "div[aria-modal=\"true\"][role=\"dialog\"][tabindex=\"-1\"]"
	ModalDialogCloseBtn                        Selector = "button[type=\"button\"][aria-label=\"закрыть\"]"
	PageTitleCount                             Selector = "span[data-marker=\"page-title/count\"]"
	PageTitleText                              Selector = "h1"
	SubmitFiltersBtn                           Selector = "button[data-marker=\"search-filters/submit-button\"]"
	LocationChangeButton                       Selector = "div[data-marker=\"search-form/change-location\"]"
	BaseEstateWidgetTypeFilterButton           Selector = "input[data-marker=\"categoryId\"]"
	BaseEstateWidgetTypeFilterDropdown         Selector = "div[class^=\"dropdown-list-dropdown-list\"]"
	BaseEstateWidgetActionFilterButton         Selector = "input[data-marker=\"param[201]\"]"
	BaseEstateWidgetDurationDailyRentButton    Selector = "input[data-marker=\"param[528](5477)/input\"]"
	WidgetSubmitButton                         Selector = "a[data-marker=\"search-form-widget/action-button-0\"]"
	DailyRentWidgetPageCalendarButton          Selector = "div[data-marker=\"params[2903]/sticker\"]"
	DailyRentWidgetPageCalendarNextMonthButton Selector = "button[data-marker=\"params[2903]/next-button\"]"
	DailyRentWidgetPageCalendarTitle           Selector = "div[class^=\"datepicker-title\"]"
	FilterCalendarResetButton                  Selector = "a[data-marker=\"params[2903]-reset\"]"
)

func CalendarBtn(t *time.Time) Selector {
	const calendarButtonSelectorTemplate = "td[data-marker^=\"params[\"][data-marker$=\"/day(%d)\"] div[role=button][class*=\"styles-module-day_hoverable-\"]"
	return Selector(fmt.Sprintf(calendarButtonSelectorTemplate, t.Day()))
}

func DailyRentWidgetPageCalendarDayButton(t *time.Time) Selector {
	const template = "td[data-marker=\"day(%d)\"] > div"
	return Selector(fmt.Sprintf(template, t.Day()))
}
