package parser

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/csr-ugra/avito-estate-parser/src/db"
	"github.com/csr-ugra/avito-estate-parser/src/util"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"strconv"
	"strings"
	"time"
)

func Start(ctx context.Context, connection bun.IDB, config *util.Config) {
	logger := util.GetLogger()

	tasks, err := loadTasks(ctx, connection)
	if err != nil {
		logger.Fatalln(err)
	}

	allocatorCtx, allocatorCancel := chromedp.NewRemoteAllocator(ctx, config.DevtoolsWebsocketUrl.Value, chromedp.NoModifyURL)
	defer allocatorCancel()

	//defer chromeCancel()

	results := make([]*ParsingTaskResult, 0, len(tasks))
	for _, task := range tasks {
		const retryCount = 3

		taskLogger := logger.WithFields(logrus.Fields{
			"TaskId":   task.Id,
			"Target":   task.Target.Name,
			"Location": task.Location.Name,
		})

		attempt := 1
		var result *ParsingTaskResult

		for attempt <= retryCount {
			chromeCtx, chromeCancel := chromedp.NewContext(allocatorCtx)
			result = nil
			err = nil

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

		if err != nil {
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

	//if err = saveTaskResults(ctx, connection, results); err != nil {
	//	logger.Fatalln(fmt.Errorf("save task results error: %w", err))
	//}
}

func indexOfTheWeekInMonth(now time.Time) int {
	beginningOfTheMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	_, thisWeek := now.ISOWeek()
	_, beginningWeek := beginningOfTheMonth.ISOWeek()
	return thisWeek - beginningWeek
}

func loadTasks(ctx context.Context, connection bun.IDB) (tasks []*ParsingTask, err error) {
	locations, err := db.GetLocations(ctx, connection)
	if err != nil {
		return nil, err
	}
	if len(locations) == 0 {
		return nil, errors.New("no locations specified")
	}

	targets, err := db.GetTargets(ctx, connection)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, errors.New("no targets specified")
	}

	taskList, err := db.GetTasks(ctx, connection)
	if err != nil {
		return nil, err
	}
	if len(taskList) == 0 {
		return nil, errors.New("no tasks specified")
	}

	tasks = make([]*ParsingTask, 0, len(taskList))
	for _, task := range taskList {
		dateStart := time.Now().Add(24 * time.Hour)
		dateEnd := time.Now().Add(48 * time.Hour)
		t, err := NewParsingTask(task, locations, targets, dateStart, dateEnd)
		if err != nil {
			return nil, fmt.Errorf("error creating parsing task: %v", err)
		}

		tasks = append(tasks, t)
	}

	return tasks, nil
}

func saveTaskResults(ctx context.Context, connection bun.IDB, results []*ParsingTaskResult) error {
	models := make([]*db.EstateParsingValueModel, 0, len(results))
	for _, result := range results {
		models = append(models, &db.EstateParsingValueModel{
			TaskId:           result.Task.Id,
			DateStart:        result.Task.DateStart,
			DateEnd:          result.Task.DateEnd,
			EstateTotalCount: result.EstateTotalCount,
			EstateFreeCount:  result.EstateFreeCount,
		})
	}

	err := db.SaveValues(ctx, connection, models)

	return fmt.Errorf("error savings task results: %w", err)
}

func runTask(ctx context.Context, task *ParsingTask, log logrus.FieldLogger) (result *ParsingTaskResult, err error) {
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

	if normalizeStr(pageTitle) != normalizeStr(task.ValidateTitle) {
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
	const calendarButtonSelectorTemplate = "td[data-marker^=\"params[\"][data-marker$=\"/day(%d)\"] div[role=button][class*=\"styles-module-day_hoverable-\"]"
	btnDayStartSelector := fmt.Sprintf(calendarButtonSelectorTemplate, task.DateStart.Day())
	btnDayEndSelector := fmt.Sprintf(calendarButtonSelectorTemplate, task.DateEnd.Day())
	buttons := []string{btnDayStartSelector, btnDayEndSelector}
	for _, selector := range buttons {
		err = click(ctx, selector)

		if errors.Is(err, &ElementNotFoundError{}) {
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

	result = &ParsingTaskResult{
		Task:             task,
		EstateTotalCount: estateObjectsCountTotal,
		EstateFreeCount:  estateObjectsCountFree,
	}

	return result, nil
}

func normalizeStr(input string) string {
	var result string
	result = input

	result = strings.Join(strings.Fields(result), "")
	result = strings.ToLower(result)

	return result
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
	const submitButtonSelector = "button[data-marker=\"search-filters/submit-button\"]"
	const submitButtonPrefixText = "показать"

	if err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var submitButtonText string
		for {
			// get text on submit button
			err = chromedp.Text(submitButtonSelector, &submitButtonText).Do(ctx)
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

	return click(ctx, submitButtonSelector)
}

func getText(ctx context.Context, selector string) (text string, err error) {
	count, err := getNodeCount(ctx, selector)

	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", NewElementNotFoundError(selector)
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
		return NewElementNotFoundError(selector)
	}

	err = chromedp.Run(ctx, chromedp.Click(selector, chromedp.AtLeast(1)))
	return err
}

func getTitle(ctx context.Context) (title string, err error) {
	const selector = "h1"
	return getText(ctx, selector)
}

func getCountFromHeader(ctx context.Context) (count int, err error) {
	const selector = "span[data-marker=\"page-title/count\"]"
	countStr, err := getText(ctx, selector)

	return strconv.Atoi(countStr)
}

func closePopupModal(ctx context.Context) (err error) {
	const modalSelector = "div[aria-modal=\"true\"][role=\"dialog\"][tabindex=\"-1\"]"
	const closeButtonSelector = "button[type=\"button\"][aria-label=\"закрыть\"]"
	var selector = fmt.Sprintf("%s %s", modalSelector, closeButtonSelector)

	count, err := getNodeCount(ctx, selector)

	if err != nil {
		return err
	}

	if count == 0 {
		return NewElementNotFoundError(selector)
	}

	return chromedp.Run(ctx, input.DispatchKeyEvent(input.KeyDown).WithKey("Escape"))
}
