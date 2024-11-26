package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
)

// ScrapeConfig описывает конфигурацию для скрапинга.
type ScrapeConfig struct {
	URL       string
	Type      string
	Name      string
	Selectors map[string]string
}

// ConfigLoader определяет интерфейс для загрузки конфигурации.
type ConfigLoader interface {
	Load(filePath string) ([]ScrapeConfig, error)
}

// JSONConfigLoader реализует загрузку конфигурации из JSON.
type JSONConfigLoader struct{}

func (j *JSONConfigLoader) Load(filePath string) ([]ScrapeConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configs []ScrapeConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return configs, nil
}

// Exporter определяет интерфейс для экспорта данных после процесса скрапинга.
type Exporter interface {
	Export(data []map[string]string) error
}

type CSVExporter struct {
	FileName string
}

func (e *CSVExporter) Export(data []map[string]string) error {
	file, err := os.Create(e.FileName)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if len(data) > 0 {
		headers := make([]string, 0, len(data[0]))
		for key := range data[0] {
			headers = append(headers, key)
		}
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	for _, record := range data {
		row := make([]string, 0, len(record))
		for _, value := range record {
			row = append(row, value)
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// Scraper определяет интерфейс для выполнения задач скрапинга.
type Scraper interface {
	Scrape(ctx context.Context) (map[string]string, error)
}

// RodScraper реализует Scraper с использованием библиотеки rod.
type RodScraper struct {
	Config  ScrapeConfig
	Browser *rod.Browser
	Logger  *log.Logger
}

func NewRodScraper(browser *rod.Browser, config ScrapeConfig, logger *log.Logger) *RodScraper {
	return &RodScraper{
		Config:  config,
		Browser: browser,
		Logger:  logger,
	}
}

func (r *RodScraper) Scrape(ctx context.Context) (map[string]string, error) {
	r.Logger.Info("Starting scrape", "url", r.Config.URL)
	page := r.Browser.MustPage()

	err := page.Context(ctx).Navigate(r.Config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %v", r.Config.URL)
	}

	results := make(map[string]string)
	for key, selector := range r.Config.Selectors {
		if selector == "" {
			results[key] = ""
			continue
		}

		element, err := page.Timeout(time.Second * 10).Element(selector)
		if err != nil {
			r.Logger.Warn("Failed to find selector", "selector", selector, "error", err)
			results[key] = ""
			continue
		}

		text, err := element.Text()
		if err != nil {
			return nil, fmt.Errorf("failed to get text for selector '%s': %w", selector, err)
		}

		results[key] = text
	}

	r.Logger.Info("Scraping completed", "url", r.Config.URL)
	return results, nil
}

// TaskRunner управляет выполнением задач скрапинга.
type TaskRunner struct {
	Tasks  []Scraper
	Logger *log.Logger
}

func NewTaskRunner(scrapeTasks []Scraper, logger *log.Logger) *TaskRunner {
	return &TaskRunner{Tasks: scrapeTasks, Logger: logger}
}

func (t *TaskRunner) Run(ctx context.Context) ([]map[string]string, []error) {
	var wg sync.WaitGroup
	resultsChan := make(chan map[string]string)
	errorsChan := make(chan error)

	for _, scraper := range t.Tasks {
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

func main() {
	configPath := flag.String("c", "", "Path to config file")
	timeout := flag.Int("t", 30, "Timeout for each scraping task in seconds")
	outputPath := flag.String("o", "output.csv", "Path to output file")
	flag.Parse()

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
	})
	logger.Info("Application started")

	if *configPath == "" {
		logger.Fatal("No config file provided. Use -c to specify config path")
	}

	loader := &JSONConfigLoader{}
	configs, err := loader.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load config", "error", err)
	}

	browser := rod.New().NoDefaultDevice()
	if err := browser.Connect(); err != nil {
		logger.Fatal("Failed to connect to browser", "error", err)
	}
	defer browser.Close()

	var tasks []Scraper
	for _, config := range configs {
		tasks = append(tasks, NewRodScraper(browser, config, logger))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	runner := NewTaskRunner(tasks, logger)
	results, errors := runner.Run(ctx)

	for _, res := range results {
		logger.Info("Scraping results", "data", res)
	}

	for _, err := range errors {
		logger.Error("Scraping error", "error", err)
	}

	exporter := &CSVExporter{FileName: *outputPath}
	if err := exporter.Export(results); err != nil {
		logger.Error("Failed to export data", "error", err)
	} else {
		logger.Info("Data exported successfully")
	}
}
