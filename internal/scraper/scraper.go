package scraper

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/stealth"
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
	r.Logger.Info("🌐 Starting scraping", "url:", r.Tasks.URL)

	// Создаём новую страницу с использованием stealth
	page, err := stealth.Page(r.Browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %v", r.Tasks.URL)
	}

	// Навигация на указанный URL
	err = page.Navigate(r.Tasks.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %v", r.Tasks.URL)
	}

	// Ожидание полной загрузки страницы
	err = page.WaitLoad()
	if err != nil {
		r.Logger.Warn("⭕ Page did not load fully", "url:", r.Tasks.URL, "error:", err)
	}

	// Создаём результаты
	results := make(map[string]string)

	// Проходимся по селекторам из конфигурации
	for key, selector := range r.Tasks.Selectors {
		// Пропускаем пустые селекторы
		if selector == "" {
			results[key] = ""
			continue
		}

		// Получение всех элементов по селектору
		elements, err := page.Elements(selector)
		if err != nil || len(elements) == 0 {
			r.Logger.Warn("⭕ No elements found", "selector:", selector, "error:", err)
			results[key] = ""
			continue
		}

		// Сбор текста всех найденных элементов
		var texts []string
		for _, element := range elements {
			text, err := element.Text()
			if err != nil {
				r.Logger.Warn("⭕ Failed to get text for element", "selector:", selector, "error:", err)
				continue
			}
			texts = append(texts, text)
		}

		// Объединяем тексты с разделителем (например, перенос строки)
		results[key] = strings.Join(texts, "\n")
		r.Logger.Info("✅ Successfully scraped", "key:", key, "count:", len(texts))
	}

	// Возвращаем результаты
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
