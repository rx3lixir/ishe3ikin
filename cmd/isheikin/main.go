package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

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

// Exporter определяет интерфейс для экспорта данных после процесса скраппинга
type Exporter interface {
	Export(data []map[string]string) error
}

type CSVExporter struct {
	FileName string
}

func (e *CSVExporter) Export(data []map[string]string) error {
	// Открываем файл для записи
	file, err := os.Create(e.FileName)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	// Создаём CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Пишем заголовок
	if len(data) > 0 {
		headers := make([]string, 0, len(data[0]))
		for key := range data[0] {
			headers = append(headers, key)
		}
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	// Пишем строки данных
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
}

// NewRodScraper создает новый RodScraper.
func NewRodScraper(browser *rod.Browser, config ScrapeConfig) *RodScraper {
	return &RodScraper{
		Config:  config,
		Browser: browser,
	}
}

func (r *RodScraper) Scrape(ctx context.Context) (map[string]string, error) {
	page := r.Browser.MustPage()

	err := page.Navigate(r.Config.URL)
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
			return nil, fmt.Errorf("failed to find selector '%s': %w", selector, err)
		}

		text, err := element.Text()
		if err != nil {
			return nil, fmt.Errorf("failed to get text for selector '%s': %w", selector, err)
		}

		results[key] = text
	}

	return results, nil
}

// TaskRunner управляет выполнением задач скрапинга.
type TaskRunner struct {
	Scrapers []Scraper
}

// NewTaskRunner создает TaskRunner.
func NewTaskRunner(scrapers []Scraper) *TaskRunner {
	return &TaskRunner{Scrapers: scrapers}
}

// Run запускает все задачи скрапинга.
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

	// Закрытие каналов после завершения всех задач.
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	var results []map[string]string
	var errorList []error

	// Чтение результатов и ошибок.
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

	// Чтение флагов командной строки.
	flag.Parse()

	if *configPath == "" {
		slog.Error("no config file provided. use -c to specify config path")
		os.Exit(1)
	}

	// Загрузка конфигурации.
	loader := &JSONConfigLoader{}
	configs, err := loader.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Подключение к браузеру.
	browser := rod.New().NoDefaultDevice()
	if err := browser.Connect(); err != nil {
		slog.Error("failed to connect to browser", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer browser.Close()

	// Создание скрапера для каждой конфигурации.
	var tasks []Scraper
	for _, config := range configs {
		tasks = append(tasks, NewRodScraper(browser, config))
	}

	// Выполнение задач через TaskRunner.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	runner := NewTaskRunner(tasks)
	results, errors := runner.Run(ctx)

	// Вывод результатов.
	for _, res := range results {
		fmt.Println("Scraping results:", res)
	}

	// Обработка ошибок.
	for _, err := range errors {
		slog.Error("scraping error", slog.String("error", err.Error()))
	}

	// Экспортируем данные скраппинга
	exporter := &CSVExporter{FileName: *outputPath}
	if err := exporter.Export(results); err != nil {
		fmt.Printf("Failed to export data: %v\n", err)
	} else {
		fmt.Println("Data exported successfully")
	}
}
