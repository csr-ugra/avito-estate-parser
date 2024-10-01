package parser_chromedp

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/csr-ugra/avito-estate-parser/internal"
	"github.com/csr-ugra/avito-estate-parser/internal/log"
	"github.com/csr-ugra/avito-estate-parser/internal/selector"
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

func Start(ctx context.Context, config *util.Config, tasks []*internal.ParsingTask) (results []*internal.ParsingTaskResult, err error) {
	logger := *log.GetLogger()
	results = make([]*internal.ParsingTaskResult, 0, len(tasks))

	allocatorCtx, allocatorCancel := chromedp.NewRemoteAllocator(ctx, config.DevtoolsWebsocketUrl.Value, chromedp.NoModifyURL)
	defer allocatorCancel()

	for _, task := range tasks {
		const maxRetryCount = 3

		taskLogger := logger.WithFields(logrus.Fields{
			"TaskId":   task.Id,
			"Target":   task.Target.Name,
			"Location": task.Location.Name,
		})

		attempt := 1
		var result *internal.ParsingTaskResult

		for attempt <= maxRetryCount {
			chromeCtx, chromeCancel := chromedp.NewContext(allocatorCtx)

			result, err = runTask(chromeCtx, task, taskLogger)
			if err != nil {
				logger.Error(err)
				attempt++

				//chromeCancel()
				taskLogger.WithField("Attempt", attempt).Warn("[{TaskId}] trying again")
				chromeCancel()
				time.Sleep(2 * time.Second)
				continue
			}

			chromeCancel()
			break
		}

		if result != nil {
			results = append(results, result)

			taskLogger.
				WithFields(logrus.Fields{
					"DateStart":  result.Task.DateStart,
					"DateEnd":    result.Task.DateEnd,
					"TotalCount": result.EstateTotalCount,
					"FreeCount":  result.EstateFreeCount,
				}).Info("[{TaskId}] done")
		}

		time.Sleep(2 * time.Second)
	}

	return results, err
}

func indexOfTheWeekInMonth(now time.Time) int {
	beginningOfTheMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	_, thisWeek := now.ISOWeek()
	_, beginningWeek := beginningOfTheMonth.ISOWeek()
	return thisWeek - beginningWeek
}

func runTask(ctx context.Context, task *internal.ParsingTask, log logrus.FieldLogger) (result *internal.ParsingTaskResult, err error) {
	log.WithField("Url", task.Url).Info("[{TaskId}] navigating to {Url}")
	if err = chromedp.Run(ctx, chromedp.Navigate(task.Url)); err != nil {
		return nil, fmt.Errorf("error navigating to %s: %v", task.Url, err)
	}

	// close popup
	err = chromedp.Run(ctx, input.DispatchKeyEvent(input.KeyDown).WithKey("Escape"))
	if err != nil {
		log.Warn("failed to dispatch 'escape' keydown event")
	}

	// get header text of the page
	log.Info("[{TaskId}] getting page title")
	pageTitle, err := getTitle(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting page title: %v", err)
	}

	if util.NormalizeStr(pageTitle) != util.NormalizeStr(task.ValidateTitle) {
		return nil, fmt.Errorf("page title doesn't match expected: %s != %s", pageTitle, task.ValidateTitle)
	}
	log.Info("[{TaskId}] page title matches expected")

	// get total estate objects count
	// since it's the first visit, there is should be no filters applied,
	// therefore count at the top of the page is total available estate objects
	log.Info("[{TaskId}] getting estate objects count from title")
	estateObjectsCountTotal, err := getCountFromHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total estate count: %v", err)
	}
	log.WithField("TotalCount", estateObjectsCountTotal).Info("[{TaskId}] got total count of estate objects of {TotalCount}")

	// select dates on calendar if calendar is present
	// data-marker begins with "params[" and ends with "/day(%d)"
	btnDayStartSelector := selector.CalendarBtn(task.DateStart)
	btnDayEndSelector := selector.CalendarBtn(task.DateEnd)
	buttons := []string{btnDayStartSelector, btnDayEndSelector}
	for _, selector := range buttons {
		err = click(ctx, selector)

		if errors.Is(err, &internal.ElementNotFoundError{}) {
			log.WithField("Selector", selector).Warnf("[{TaskId}] calendar button not found, skipping click")
			continue
		}

		// wait for changes to reflect
		err = chromedp.Run(ctx, chromedp.Sleep(2*time.Second))
		if err != nil {
			return nil, err
		}
	}

	// wait for submit button to be enabled and click it
	if err = clickSubmitButton(ctx, log); err != nil {
		return nil, fmt.Errorf("error clicking submit button: %v", err)
	}

	// wait for page to load new count
	log.Info("[{TaskId}] waiting 2 seconds for estate object count in title to update")
	if err = chromedp.Run(ctx, chromedp.Sleep(2*time.Second)); err != nil {
		return nil, fmt.Errorf("error waiting for new count: %v", err)
	}

	// get free for rent estate objects count
	log.Info("[{TaskId}] getting estate objects count from title")
	estateObjectsCountFree, err := getCountFromHeader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get free estate count: %v", err)
	}
	log.WithField("FreeCount", estateObjectsCountFree).Info("[{TaskId}] got count of free estate objects of {FreeCount}")

	result = &internal.ParsingTaskResult{
		Task:             task,
		EstateTotalCount: estateObjectsCountTotal,
		EstateFreeCount:  estateObjectsCountFree,
	}

	return result, nil
}

func getNodeCount(ctx context.Context, selector string) (count int, err error) {
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		if err = chromedp.Nodes(selector, &nodes, chromedp.AtLeast(0)).Do(ctx); err != nil {
			return err
		}

		if nodes == nil {
			count = 0
			return nil
		}
		count = len(nodes)

		return nil
	}))

	return count, err
}

func clickSubmitButton(ctx context.Context, log logrus.FieldLogger) (err error) {
	const submitButtonPrefixText = "показать"

	if err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var submitButtonText string
		for {
			// get text on submit button
			err = chromedp.Text(selector.SubmitFiltersBtn, &submitButtonText).Do(ctx)
			if err != nil {
				return err
			}

			if strings.HasPrefix(strings.ToLower(submitButtonText), submitButtonPrefixText) {
				break
			}

			// wait before checking again
			log.Info("[{TaskId}] waiting for submit button to be clickable...")
			time.Sleep(500 * time.Millisecond)
		}

		return nil
	})); err != nil {
		return err
	}

	return click(ctx, selector.SubmitFiltersBtn)
}

func getText(ctx context.Context, selector string) (text string, err error) {
	count, err := getNodeCount(ctx, selector)

	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", internal.NewElementNotFoundError(selector)
	}

	err = chromedp.Run(ctx, chromedp.Text(selector, &text))
	return text, err
}

func click(ctx context.Context, selector string) (err error) {
	count, err := getNodeCount(ctx, selector)

	if err != nil {
		return err
	}

	if count == 0 {
		return internal.NewElementNotFoundError(selector)
	}

	err = chromedp.Run(ctx, chromedp.Click(selector, chromedp.AtLeast(1)))
	return err
}

func getTitle(ctx context.Context) (title string, err error) {
	return getText(ctx, selector.PageTitleText)
}

func getCountFromHeader(ctx context.Context) (count int, err error) {
	countStr, err := getText(ctx, selector.PageTitleCount)

	return strconv.Atoi(countStr)
}

func closePopupModal(ctx context.Context) (err error) {
	var selector = fmt.Sprintf("%s %s", selector.ModalDialog, selector.ModalDialogCloseBtn)

	count, err := getNodeCount(ctx, selector)

	if err != nil {
		return err
	}

	if count == 0 {
		return internal.NewElementNotFoundError(selector)
	}

	return chromedp.Run(ctx, input.DispatchKeyEvent(input.KeyDown).WithKey("Escape"))
}
