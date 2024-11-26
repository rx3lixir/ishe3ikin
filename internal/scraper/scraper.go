package scraper

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/rx3lixir/ish3ikin/internal/config/taskconfig"
)

type RodScraper struct {
	Tasks   taskconfig.TaskConfig
	Browser *rod.Browser
	Logger  *log.Logger
}

// NewRodScraper создает новый RodScraper.
func NewRodScraper(browser *rod.Browser, config taskconfig.TaskConfig, logger *log.Logger) *RodScraper {
	return &RodScraper{
		Tasks:   config,
		Browser: browser,
		Logger:  logger,
	}
}

// Scraper интерфейс для выполнения скрапинга.
type Scraper interface {
	Scrape(ctx context.Context) (map[string]string, error)
}

func (r *RodScraper) Scrape(ctx context.Context) (map[string]string, error) {
	r.Logger.Info("🌐Starting scraping", "url:", r.Tasks.URL)
	page := r.Browser.MustPage()

	err := page.Navigate(r.Tasks.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %v", r.Tasks.URL)
	}

	results := make(map[string]string)
	for key, selector := range r.Tasks.Selectors {
		if selector == "" {
			results[key] = ""
			continue
		}

		element, err := page.Timeout(time.Second * 10).Element(selector)
		if err != nil {
			r.Logger.Warn("⭕Failed to find selector", "selector:", selector, "error:", err)
			results[key] = ""
			continue
		}

		text, err := element.Text()
		if err != nil {
			return nil, fmt.Errorf("failed to get text for selector '%s': %w", selector, err)
		}

		results[key] = text
	}

	return results, nil
}

// TaskRunner управляет выполнением задач.
type TaskRunner struct {
	Scrapers []Scraper
}

func NewTaskRunner(scrapers []Scraper) *TaskRunner {
	return &TaskRunner{Scrapers: scrapers}
}

func (t *TaskRunner) Run(ctx context.Context) ([]map[string]string, []error) {
	var wg sync.WaitGroup
	resultsChan := make(chan map[string]string)
	errorsChan := make(chan error)

	for _, scraper := range t.Scrapers {
		wg.Add(1)
		go func(s Scraper) {
			defer wg.Done()
			result, err := s.Scrape(ctx)
			if err != nil {
				errorsChan <- err
				return
			}
			resultsChan <- result
		}(scraper)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	var results []map[string]string
	var errorList []error

	for {
		select {
		case res, ok := <-resultsChan:
			if !ok {
				resultsChan = nil
			} else {
				results = append(results, res)
			}
		case err, ok := <-errorsChan:
			if !ok {
				errorsChan = nil
			} else {
				errorList = append(errorList, err)
			}
		case <-ctx.Done():
			errorList = append(errorList, errors.New("scraping tasks timed out"))
			return results, errorList
		}

		if resultsChan == nil && errorsChan == nil {
			break
		}
	}

	return results, errorList
}
