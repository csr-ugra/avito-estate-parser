package parser

import (
	"github.com/csr-ugra/avito-estate-parser/internal"
	"github.com/csr-ugra/avito-estate-parser/internal/log"
	"github.com/csr-ugra/avito-estate-parser/internal/selector"
	"github.com/csr-ugra/avito-estate-parser/internal/util"
	"github.com/csr-ugra/avito-estate-parser/internal/util/assert"
	"github.com/go-rod/rod"
	"strconv"
	"time"
)

// connect to running browser
func getBrowser(devtoolsWebsocketUrl string) *rod.Browser {
	if devtoolsWebsocketUrl == "" {
		// since designed to run in docker compose with dedicated browser image,
		// downloading browser to this container is a stupid decision
		log.GetLogger().Panicln("failed to attach to browser, devtools url not specified")
		//path, _ := launcher.LookPath()
		//devtoolsWebsocketUrl = launcher.New().Bin(path).MustLaunch()
	}

	return rod.New().
		SlowMotion(1 * time.Second).
		Trace(true).
		ControlURL(devtoolsWebsocketUrl).
		MustConnect()
}

func getElement(page *rod.Page, sel selector.Selector) (el *rod.Element, err error) {
	el, err = page.Sleeper(rod.NotFoundSleeper).Element(sel.String())
	if err != nil {
		return el, internal.NewElementNotFoundError(sel)
	}

	return el, nil
}

func tryGetElement(page *rod.Page, sel selector.Selector, maxRetryCount uint, sleepTime time.Duration) (el *rod.Element, err error) {
	for i := uint(0); i < maxRetryCount; i++ {
		el, err = getElement(page, sel)
		if err == nil {
			return el, nil
		}

		if i < maxRetryCount-1 {
			time.Sleep(sleepTime)
		}
	}

	return nil, internal.NewElementNotFoundError(sel)
}

func countElements(page *rod.Page, sel selector.Selector) int {
	elements, err := page.Elements(sel.String())
	if err != nil {
		return 0
	}

	return len(elements)
}

func getElementText(el *rod.Element) (string, error) {
	assert.NotNil(el, "expecting element to get text from to be not nil")

	return el.Text()
}

func getText(page *rod.Page, sel selector.Selector) (string, error) {
	count := countElements(page, sel)
	if count == 0 {
		return "", internal.NewElementNotFoundError(sel)
	}

	el := page.MustElement(sel.String())
	return el.Text()
}

func clickElement(el *rod.Element) {
	assert.NotNil(el, "expecting element to click to be not nil")

	el.MustClick()
}

func click(page *rod.Page, sel selector.Selector) error {
	count := countElements(page, sel)
	if count == 0 {
		return internal.NewElementNotFoundError(sel)
	}

	page.MustElement(sel.String()).MustClick()
	return nil
}

func getInt(el *rod.Element) (int, error) {
	assert.NotNil(el, "expecting element to get int from to be not nil")

	str, err := getElementText(el)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(util.Normalize(str))
}
