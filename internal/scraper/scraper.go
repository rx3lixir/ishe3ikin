package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/stealth"
	"github.com/rx3lixir/ish3ikin/internal/config/taskconfig"
)

type Scraper interface {
	Scrape(ctx context.Context, task taskconfig.Task) (map[string]string, error)
}

type RodScraper struct {
	Browser *rod.Browser
	Logger  log.Logger
}

func NewRodScraper(browser *rod.Browser, logger log.Logger) *RodScraper {
	return &RodScraper{
		Browser: browser,
		Logger:  logger,
	}
}

// Scrape выполняет скрапинг и возвращает результаты.
func (r *RodScraper) Scrape(ctx context.Context, task taskconfig.Task) (map[string]string, error) {
	r.Logger.Info("🌐 Starting scraping", "url:", task.URL)

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("Scraping canceled before creating page: %w", ctx.Err())
	default:
	}

	page, err := stealth.Page(r.Browser)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %v", err)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("Scraping canceled during naviagation to page: %w", ctx.Err())
	default:
	}

	err = page.Navigate(task.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %v", err)
	}

	err = page.WaitLoad()
	if err != nil {
		r.Logger.Warn("⭕ Page did not load fully", "url:", task.URL, "error:", err)
	}

	results := make(map[string]string)
	results["URL"] = task.URL
	results["Type"] = task.Type

	for key, selector := range task.Selectors {
		select {
		case <-ctx.Done():
			r.Logger.Warn("⭕ Scraping canceled during selector processing", "key:", key)
			return results, fmt.Errorf("scraping canceled: %w", ctx.Err())
		default:
		}

		if selector == "" {
			results[key] = ""
			continue
		}

		elements, err := page.Elements(selector)
		if err != nil || len(elements) == 0 {
			r.Logger.Warn("⭕ No elements found", "selector:", selector, "error:", err)
			results[key] = ""
			continue
		}

		var texts []string
		for _, element := range elements {
			select {
			case <-ctx.Done():
				r.Logger.Warn("⭕ Scraping canceled during element processing", "key:", key)
				return results, fmt.Errorf("scraping canceled: %w", ctx.Err())
			default:
			}
			text, err := element.Text()
			if err != nil {
				r.Logger.Warn("⭕ Failed to get text for element", "selector:", selector, "error:", err)
				continue
			}
			texts = append(texts, text)
		}

		results[key] = strings.Join(texts, "\n")
		r.Logger.Info("✅ Successfully scraped", "key:", key, "count:", len(texts))
	}

	return results, nil
}
