package parser_rod

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
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

func Start(ctx context.Context, cfg *util.Config, tasks []*internal.ParsingTask) (results []*internal.ParsingTaskResult, err error) {
	logger := *log.GetLogger()
	results = make([]*internal.ParsingTaskResult, 0, len(tasks))

	// connect to remote
	l := launcher.MustNewManaged(cfg.DevtoolsWebsocketUrl.Value)
	browser := rod.New().Client(l.MustClient()).MustConnect()

	// launch default (local if installed, download otherwise)
	//browser := rod.New().MustConnect()
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
		}

		time.Sleep(2 * time.Second)
	}

	return results, err
}

func runTask(browser *rod.Browser, task *internal.ParsingTask, log log.Logger) (result *internal.ParsingTaskResult, err error) {
	page := browser.MustPage()
	defer page.Close()

	log.Debug("navigating to task url")
	waitNetwork := page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	wait := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	err = page.Navigate(task.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %v", task.Url, err)
	}
	wait()
	waitNetwork()

	if err = page.KeyActions().Press(input.Escape).Do(); err != nil {
		log.Warn("failed to dispatch 'escape' keydown event")
	}

	pageTitle, err := getText(page, selector.PageTitleText)
	if err != nil {
		return nil, fmt.Errorf("error getting page title: %v", err)
	}

	if util.NormalizeStr(pageTitle) != util.NormalizeStr(task.ValidateTitle) {
		return nil, fmt.Errorf("page title != expected: %s != %s", pageTitle, task.ValidateTitle)
	}
	log.Debug("page title matches expected")

	// get total estate objects count
	// since it's the first visit, there is should be no filters applied,
	// therefore count at the top of the page is total available estate objects
	log.Debug("getting estate objects count from title")
	estateObjectsCountTotal, err := getCountFromHeader(page)
	if err != nil {
		return nil, fmt.Errorf("failed to get total estate count: %v", err)
	}
	log.WithField("TotalCount", estateObjectsCountTotal).Info("got total count of estate objects of {TotalCount}")

	// select dates on calendar if calendar is present
	// data-marker begins with "params[" and ends with "/day(%d)"
	btnDayStartSelector := selector.CalendarBtn(task.DateStart)
	btnDayEndSelector := selector.CalendarBtn(task.DateEnd)
	buttons := []string{btnDayStartSelector, btnDayEndSelector}
	for _, sel := range buttons {
		log.Debug("clicking calendar")
		err = click(page, sel)

		if errors.Is(err, &internal.ElementNotFoundError{}) {
			log.WithField("Selector", sel).Warnf("calendar button not found, skipping click")
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
	log.WithField("FreeCount", estateObjectsCountFree).Info("got count of free estate objects of {FreeCount}")

	result = &internal.ParsingTaskResult{
		Task:             task,
		EstateTotalCount: estateObjectsCountTotal,
		EstateFreeCount:  estateObjectsCountFree,
	}

	return result, nil
}

func getElementCount(page *rod.Page, selector string) int {
	elements, err := page.Elements(selector)
	if err != nil {
		return 0
	}

	return len(elements)
}

func getText(page *rod.Page, selector string) (string, error) {
	count := getElementCount(page, selector)
	if count == 0 {
		return "", internal.NewElementNotFoundError(selector)
	}

	el := page.MustElement(selector)
	return el.Text()
}

func click(page *rod.Page, selector string) error {
	count := getElementCount(page, selector)
	if count == 0 {
		return internal.NewElementNotFoundError(selector)
	}

	page.MustElement(selector).MustClick()
	return nil
}

func getCountFromHeader(page *rod.Page) (count int, err error) {
	countStr, err := getText(page, selector.PageTitleCount)

	return strconv.Atoi(util.NormalizeStr(countStr))
}
