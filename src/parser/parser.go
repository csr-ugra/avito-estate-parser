package parser

import (
	"context"
	"errors"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/csr-ugra/avito-estate-parser/src/db"
	"github.com/uptrace/bun"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const devtoolWebsocketUrlEnvName = "DEVTOOLS_WEBSOCKET_URL"

func Start(ctx context.Context, connection bun.IDB) {
	devtoolsWsURL := os.Getenv(devtoolWebsocketUrlEnvName)
	if devtoolsWsURL == "" {
		log.Fatalf("environment variable %s is not set", devtoolWebsocketUrlEnvName)
	}

	tasks, err := loadTasks(ctx, connection)
	if err != nil {
		log.Fatalln(err)
	}

	allocatorCtx, allocatorCancel := chromedp.NewRemoteAllocator(ctx, devtoolsWsURL, chromedp.NoModifyURL)
	defer allocatorCancel()

	chromeCtx, chromeCancel := chromedp.NewContext(allocatorCtx)
	defer chromeCancel()
	results := make([]*ParsingTaskResult, 0, len(tasks))
	for _, task := range tasks {

		prefix := fmt.Sprintf("task [%d]: %s в %s -", task.Id, task.Target.Name, task.Location.Name)

		result, err := runTask(chromeCtx, task)
		if err != nil {
			log.Printf("%s error: %s", prefix, err)

			//chromeCancel()
			time.Sleep(2 * time.Second)

			continue
		}

		results = append(results, result)

		log.Printf("%s done: in period of %s - %s for rent %d ot of total %d",
			prefix, result.Task.DateStart.Format(time.DateOnly), result.Task.DateEnd.Format(time.DateOnly), result.EstateFreeCount, result.EstateTotalCount)

		//chromeCancel()
		time.Sleep(2 * time.Second)
	}

	//if err = saveTaskResults(ctx, connection, results); err != nil {
	//	log.Fatalln(fmt.Errorf("save task results error: %w", err))
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

func runTask(ctx context.Context, task *ParsingTask) (result *ParsingTaskResult, err error) {
	// navigate
	log.Printf("navigating to %s", task.Url)
	if err = chromedp.Run(ctx, chromedp.Navigate(task.Url)); err != nil {
		return nil, fmt.Errorf("error navigating to %s: %v", task.Url, err)
	}

	if err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		const selector = "button[type=\"button\"][aria-label=\"закрыть\"]"

		var nodes []*cdp.Node
		if err := chromedp.Nodes(selector, &nodes, chromedp.ByQuery, chromedp.AtLeast(0)).Do(ctx); err != nil {
			return err
		}

		if len(nodes) == 0 {
			log.Printf("popup not found")
			return nil
		}

		err = chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
		return err
	})); err != nil {
		return nil, fmt.Errorf("error closing popup: %v", err)
	}

	const titleSelector = "h1[data-marker=\"page-title/text\"]"

	// wait title visible
	//log.Printf("waiting for title to be visible")
	//if err = chromedp.Run(ctx, chromedp.WaitVisible(titleSelector)); err != nil {
	//	return nil, fmt.Errorf("error getting section %s: %v", task.Url, err)
	//}

	// click header to discard popup model
	//log.Printf("clicking title to discard modal popup")
	//if err = chromedp.Run(ctx, chromedp.Click(titleSelector)); err != nil {
	//	return nil, fmt.Errorf("error discarding popup model %s: %v", task.Url, err)
	//}

	// get header text of the page
	log.Printf("getting page title")
	var pageTitle string
	if err = chromedp.Run(ctx, chromedp.Text(titleSelector, &pageTitle)); err != nil {
		return nil, fmt.Errorf("error getting page title: %v", err)
	}

	if normalizeStr(pageTitle) != normalizeStr(task.ValidateTitle) {
		return nil, fmt.Errorf("page title doesn't match expected: %s != %s", pageTitle, task.ValidateTitle)
	}
	log.Printf("page title matches expected")

	// get total estate objects count
	// since it's the first visit, there is should be no filters applied,
	// therefore count at the top of the page is total available estate objects
	log.Printf("gettting estate objects count from title")
	var estateObjectsCountTotalStr string
	if err = chromedp.Run(ctx, chromedp.Text("span[data-marker=\"page-title/count\"]", &estateObjectsCountTotalStr)); err != nil {
		return nil, fmt.Errorf("error getting total estate objects: %v", err)
	}

	estateObjectsCountTotalStr = strings.Join(strings.Fields(estateObjectsCountTotalStr), "")
	estateObjectsCountTotal, err := strconv.Atoi(estateObjectsCountTotalStr)
	if err != nil {
		return nil, fmt.Errorf("error converting count in title (%s) to int: %v", estateObjectsCountTotalStr, err)
	}

	// select dates on calendar if calendar is present
	// data-marker begins with "params[" and ends with "/day(%d)"
	const calendarButtonSelectorTemplate = "tr[data-marker^=\"params[\"][data-marker$=\"/week(%d)\"] td[data-marker^=\"params[\"][data-marker$=\"/day(%d)\"] div[role=button]"
	btnDayStartSelector := fmt.Sprintf(calendarButtonSelectorTemplate, indexOfTheWeekInMonth(*task.DateStart), task.DateStart.Day())
	btnDayEndSelector := fmt.Sprintf(calendarButtonSelectorTemplate, indexOfTheWeekInMonth(*task.DateStart), task.DateEnd.Day())
	if err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		buttons := []string{btnDayStartSelector, btnDayEndSelector}
		for _, selector := range buttons {

			log.Printf("looking for calendar button")
			var nodes []*cdp.Node
			if err := chromedp.Nodes(selector, &nodes, chromedp.ByQuery, chromedp.AtLeast(0)).Do(ctx); err != nil {
				return err
			}

			if len(nodes) == 0 {
				log.Printf("calendar button with selector %s not found, skipping click", btnDayStartSelector)
				continue
			}

			err = chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
			if err != nil {
				return err
			}

			log.Printf("calendar button clicked, waiting 2 seconds")

			err = chromedp.Sleep(2 * time.Second).Do(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})); err != nil {
		return nil, fmt.Errorf("error clicking calendar buttons: %v", err)
	}

	// wait for submit button to be enabled and click it
	const submitButtonSelector = "button[data-marker=\"search-filters/submit-button\"]"
	const submitButtonExpectedText = "показать"
	if err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var submitButtonText string
		for {
			// get text on submit button
			err = chromedp.Text(submitButtonSelector, &submitButtonText).Do(ctx)
			if err != nil {
				return err
			}

			if strings.HasPrefix(strings.ToLower(submitButtonText), submitButtonExpectedText) {
				break
			}

			// wait before checking again
			log.Printf("waiting for submit button to be clickable...")
			time.Sleep(500 * time.Millisecond)
		}

		log.Printf("clicking submit button")
		if err = chromedp.Click(submitButtonSelector, chromedp.ByQuery).Do(ctx); err != nil {
			return fmt.Errorf("error clicking submit button: %v", err)
		}

		return nil
	})); err != nil {
		return nil, fmt.Errorf("error clicking submit button: %v", err)
	}

	// wait for page to load new count
	log.Printf("waiting 2seconds for estate object count in title to update")
	if err = chromedp.Run(ctx, chromedp.Sleep(2*time.Second)); err != nil {
		return nil, fmt.Errorf("error waiting for new count: %v", err)
	}

	// get free for rent estate objects count
	log.Printf("gettting estate objects count from title")
	var estateObjectsCountAvailableForRentStr string
	if err = chromedp.Run(ctx, chromedp.Text("span[data-marker=\"page-title/count\"]", &estateObjectsCountAvailableForRentStr, chromedp.NodeVisible)); err != nil {
		return nil, fmt.Errorf("error getting free estate objects: %v", err)
	}

	estateObjectsCountAvailableForRentStr = strings.Join(strings.Fields(estateObjectsCountAvailableForRentStr), "")
	estateObjectsCountAvailableForRent, err := strconv.Atoi(estateObjectsCountAvailableForRentStr)
	if err != nil {
		return nil, fmt.Errorf("error converting count in title (%s) to int: %v", estateObjectsCountAvailableForRentStr, err)
	}

	result = &ParsingTaskResult{
		Task:             task,
		EstateTotalCount: estateObjectsCountTotal,
		EstateFreeCount:  estateObjectsCountAvailableForRent,
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
