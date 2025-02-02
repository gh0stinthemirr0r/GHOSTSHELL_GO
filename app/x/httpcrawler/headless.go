package httpcrawler

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// HeadlessBrowser represents a headless browser instance
type HeadlessBrowser struct {
	browser *rod.Browser
}

// NewHeadlessBrowser initializes a new headless browser instance
func NewHeadlessBrowser() (*HeadlessBrowser, error) {
	path, err := launcher.New().Headless(true).Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch headless browser: %w", err)
	}

	browser := rod.New().ControlURL(path).MustConnect()
	return &HeadlessBrowser{browser: browser}, nil
}

// TakeScreenshot captures a screenshot of the given URL and saves it to the specified file
func (hb *HeadlessBrowser) TakeScreenshot(url, outputFile string) error {
	page := hb.browser.MustPage(url).MustWaitLoad()
	err := page.Screenshot(outputFile)
	if err != nil {
		return fmt.Errorf("failed to take screenshot of %s: %w", url, err)
	}
	return nil
}

// ExtractHTML retrieves the full HTML content of the given URL
func (hb *HeadlessBrowser) ExtractHTML(url string) (string, error) {
	page := hb.browser.MustPage(url).MustWaitLoad()
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("failed to extract HTML from %s: %w", url, err)
	}
	return html, nil
}

// Close shuts down the headless browser
func (hb *HeadlessBrowser) Close() {
	hb.browser.MustClose()
}
