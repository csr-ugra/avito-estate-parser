package parser

import (
	"context"
	"errors"
	"fmt"
	"github.com/csr-ugra/avito-estate-parser/internal"
	"github.com/csr-ugra/avito-estate-parser/internal/log"
	"github.com/csr-ugra/avito-estate-parser/internal/selector"
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

func Start(ctx context.Context, cfg *util.Config, tasks []*internal.ParsingTask) (results []*internal.ParsingTaskResult, err error) {
	logger := *log.GetLogger()
	results = make([]*internal.ParsingTaskResult, 0, len(tasks))

	browser := getBrowser(cfg.DevtoolsWebsocketUrl.Value)
	for _, task := range tasks {
		const maxRetryCount = 3

		taskLogger := logger.WithFields(logrus.Fields{
			"TaskId":       task.Id,
			"TargetId":     task.Target.Name,
			"TargetName":   task.Target.Name,
			"LocationId":   task.Location.Id,
			"LocationName": task.Location.Name,
			"Url":          task.Url,
			"Description":  task.Description,
			"DateStart":    task.DateStart.Format(time.DateOnly),
			"DateEnd":      task.DateEnd.Format(time.DateOnly),
		})

		attempt := 1

		for attempt <= maxRetryCount {
			result, err := runTask(browser, task, taskLogger)
			if err != nil {
				taskLogger.Error(err)
				attempt++

				taskLogger.WithField("ParsingAttempt", attempt).Warn("failed to compete task, trying again")
				time.Sleep(2 * time.Second)
				continue
			}

			results = append(results, result)
			break
		}

		time.Sleep(2 * time.Second)
	}

	return results, err
}

func runTask(browser *rod.Browser, task *internal.ParsingTask, log log.Logger) (result *internal.ParsingTaskResult, err error) {
	page := browser.MustPage()
	// ignoring error explicitly since we don't really care
	defer func(page *rod.Page) {
		_ = page.Close()
	}(page)

	log.Debug("navigating to task url")
	waitNetwork := page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	err = page.Navigate(task.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %v", task.Url, err)
	}

	log.Debug("waiting for network idle")
	waitNetwork()

	log.Debug("closing popups just in case")
	for i := 0; i < 3; i++ {
		if err = page.KeyActions().Press(input.Escape).Do(); err != nil {
			log.Warn("failed to dispatch 'escape' keydown event")
		}

		time.Sleep(500 * time.Millisecond)
	}

	return parsePage(page, task, log)
}

// checks if page title is expected for given task and parses counts from page
// if not tries to navigate to target page and parse it
func parsePage(page *rod.Page, task *internal.ParsingTask, log log.Logger) (result *internal.ParsingTaskResult, err error) {
	pageTitle, err := getText(page, selector.PageTitleText)
	if err != nil {
		return nil, fmt.Errorf("error getting page title: %w", err)
	}

	isEstateListPage := util.Normalize(pageTitle) == util.Normalize(task.ValidateTitle)
	if isEstateListPage {
		log.Debug("page title matches expected")
		return parseEstateListPage(page, task, log)
	}

	log.WithFields(logrus.Fields{
		"TitleExpected": task.ValidateTitle,
		"TitleActual":   pageTitle,
	}).Warn("page title does not match expected")

	if strings.HasPrefix(pageTitle, "Недвижимость в ") {
		log.Info("trying to navigate to target page from base estate page")
		err = tryNavigateFromBaseEstateWidget(page, task, log)
		if err != nil {
			return nil, fmt.Errorf("error navigating to target page: %w", err)
		}

		return parseEstateListPage(page, task, log)
	}

	if strings.HasPrefix(pageTitle, "Жильё посуточно") {
		log.Info("trying to navigate to target page from daily rent widget")
		err = tryNavigateFromDailyRentWidget(page, task, log)
		if err != nil {
			return nil, fmt.Errorf("error navigating to target page: %w", err)
		}

		return parseEstateListPage(page, task, log)
	}

	return nil, fmt.Errorf("current page is unknown, can't navigate; page title is %q", pageTitle)
}

func parseEstateListPage(page *rod.Page, task *internal.ParsingTask, log log.Logger) (result *internal.ParsingTaskResult, err error) {
	// get total estate objects count
	// since it's the first visit, there is should be no filters applied,
	// therefore count at the top of the page is total available estate objects
	log.Debug("getting estate objects count from title")
	estateObjectsCountTotal, err := getCountFromHeader(page)
	if err != nil {
		return nil, fmt.Errorf("failed to get total estate count: %v", err)
	}

	// select dates on calendar if calendar is present
	// data-marker begins with "params[" and ends with "/day(%d)"
	btnDayStartSelector := selector.CalendarBtn(task.DateStart)
	btnDayEndSelector := selector.CalendarBtn(task.DateEnd)
	buttons := []selector.Selector{btnDayStartSelector, btnDayEndSelector}
	for _, sel := range buttons {
		log.Debug("clicking calendar")
		err = click(page, sel)

		if errors.Is(err, &internal.ElementNotFoundError{}) {
			log.WithField("Selector", sel).Warn("calendar button not found, skipping click")
			continue
		}

		// wait for changes to reflect
		time.Sleep(2 * time.Second)
	}

	log.Debug("clicking submit filters")
	err = click(page, selector.SubmitFiltersBtn)
	if err != nil {
		return nil, fmt.Errorf("failed to submit filters: %v", err)
	}
	time.Sleep(2 * time.Second)

	log.Debug("getting estate objects count from title")
	estateObjectsCountFree, err := getCountFromHeader(page)
	if err != nil {
		return nil, fmt.Errorf("failed to get free estate count: %v", err)
	}
	log.WithFields(logrus.Fields{
		"FreeCount":  estateObjectsCountFree,
		"TotalCount": estateObjectsCountTotal,
	}).Info("got counts of estate objects: {FreeCount}/{TotalCount}")

	result = &internal.ParsingTaskResult{
		Task:             task,
		EstateTotalCount: estateObjectsCountTotal,
		EstateFreeCount:  estateObjectsCountFree,
	}

	return result, nil
}

// navigate from estate daily rent widget,
// eg. from https://www.avito.ru/hanty-mansiyskiy_ao/doma_dachi_kottedzhi/sdam/posutochno-ASgBAgICAkSUA9IQoAjKVQ
func tryNavigateFromDailyRentWidget(page *rod.Page, task *internal.ParsingTask, log log.Logger) (err error) {
	err = checkLocation(page, task, log)
	if err != nil {
		return err
	}

	log.Info("setting dates")
	calendarButton, err := getElement(page, selector.DailyRentWidgetPageCalendarButton)
	if err != nil {
		return err
	}

	clickElement(calendarButton)

	time.Sleep(time.Second)

	for _, date := range []*time.Time{task.DateStart, task.DateEnd} {
		// check if calendar month is the same as task date or need to change
		calendarMonth, err := getText(page, selector.DailyRentWidgetPageCalendarTitle)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(calendarMonth, util.MonthString(*date)) {
			calendarNextMonthButton, err := getElement(page, selector.DailyRentWidgetPageCalendarNextMonthButton)
			if err != nil {
				return err
			}

			clickElement(calendarNextMonthButton)
		}

		calendarDateStartButton, err := getElement(page, selector.DailyRentWidgetPageCalendarDayButton(date))
		if err != nil {
			return err
		}
		clickElement(calendarDateStartButton)
		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)

	submitButton, err := tryGetElement(page, selector.WidgetSubmitButton, 3, time.Second/2)
	if err != nil {
		return err
	}

	waitNetwork := page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	clickElement(submitButton)
	waitNetwork()

	calendarResetButton, err := tryGetElement(page, selector.FilterCalendarResetButton, 3, time.Second/2)
	if err != nil {
		if errors.Is(err, internal.ElementNotFoundError{}) {
			return nil
		}
		return err
	}
	clickElement(calendarResetButton)

	err = click(page, selector.SubmitFiltersBtn)
	if err != nil {
		return fmt.Errorf("failed to submit filters: %v", err)
	}
	time.Sleep(2 * time.Second)

	return nil
}

// navigate from base estate page,
// eg. from https://www.avito.ru/hanty-mansiyskiy_ao/nedvizhimost
func tryNavigateFromBaseEstateWidget(page *rod.Page, task *internal.ParsingTask, log log.Logger) (err error) {
	err = checkLocation(page, task, log)
	if err != nil {
		return err
	}

	log.WithField("Target", task.Target.FilterText).
		Info("setting estate type target")
	estateTypeButton, err := getElement(page, selector.BaseEstateWidgetTypeFilterButton)
	if err != nil {
		return err
	}

	clickElement(estateTypeButton)

	estateTypeListWrapper, err := getElement(page, selector.BaseEstateWidgetTypeFilterDropdown)
	if err != nil {
		return err
	}

	estateTypeList, err := estateTypeListWrapper.MustElement("div").Elements("div")
	if err != nil {
		return err
	}

	isTargetFilterFound := false
	for _, el := range estateTypeList {
		if isTargetFilterFound {
			break
		}

		text, err := getElementText(el)
		if err != nil {
			return err
		}

		if util.Normalize(text) == util.Normalize(task.Target.FilterText) {
			isTargetFilterFound = true
			clickElement(el)
		}
	}

	if !isTargetFilterFound {
		return fmt.Errorf("target filter %q not found", task.Target.FilterText)
	}

	log.Info("setting target action to rent")
	actionButton, err := getElement(page, selector.BaseEstateWidgetActionFilterButton)
	if err != nil {
		return err
	}

	clickElement(actionButton)

	estateActionListWrapper, err := getElement(page, selector.BaseEstateWidgetTypeFilterDropdown)
	if err != nil {
		return err
	}

	estateActionList, err := estateActionListWrapper.MustElement("div").Elements("div")
	if err != nil {
		return err
	}

	isActionFound := false
	for _, el := range estateActionList {
		if isActionFound {
			break
		}

		text, err := getElementText(el)
		if err != nil {
			return err
		}

		if util.Normalize(text) == util.Normalize("Снять") {
			isActionFound = true
			clickElement(el)
		}
	}

	if !isActionFound {
		return fmt.Errorf("target axtion %q not found", "Снять")
	}

	log.Info("setting target duration to daily rent")
	durationButton, err := getElement(page, selector.BaseEstateWidgetDurationDailyRentButton)
	if err != nil {
		return err
	}

	clickElement(durationButton)

	submitButton, err := getElement(page, selector.WidgetSubmitButton)
	if err != nil {
		return err
	}

	waitNetwork := page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	clickElement(submitButton)
	waitNetwork()

	return err
}

func checkLocation(page *rod.Page, task *internal.ParsingTask, log log.Logger) error {
	log.Debug("checking location")
	locationButton, err := getElement(page, selector.LocationChangeButton)
	if err != nil {
		return err
	}

	location, err := getElementText(locationButton)
	if err != nil {
		return err
	}

	isLocationMatchTask := util.Normalize(location) == util.Normalize(task.Location.Name)
	if isLocationMatchTask {
		return nil
	}

	log.WithFields(logrus.Fields{
		"LocationExpected": task.Location.Name,
		"LocationActual":   location,
	}).Info("location does not match expected, changing...")

	err = changeLocation(page, task.Location.Name)
	if err != nil {
		return fmt.Errorf("error changing location: %w", err)
	}

	return nil
}

func changeLocation(page *rod.Page, targetLocation string) (err error) {
	// todo: change location
	return errors.New("not implemented")
}

func getCountFromHeader(page *rod.Page) (count int, err error) {
	el, err := getElement(page, selector.PageTitleCount)
	if err != nil {
		return 0, err
	}

	return getInt(el)
}
