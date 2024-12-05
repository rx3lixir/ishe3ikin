package main

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/rx3lixir/ish3ikin/internal/config/appconfig"
	"github.com/rx3lixir/ish3ikin/internal/config/taskconfig"
	"github.com/rx3lixir/ish3ikin/internal/lib/logger"
	"github.com/rx3lixir/ish3ikin/internal/scraper"
)

const (
	workerCount = 5
)

func main() {
	// Инициализация логгера
	logger := logger.NewLogger()

	// Загрузка конфигурации
	cfg := appconfig.NewAppConfig()

	// Загрузка задач для скрапинга
	taskLoader := taskconfig.JSONTasksLoader{}
	loadedTasks, err := taskLoader.Load(cfg.ConfigPath)
	if err != nil {
		logger.Fatal("Failed to load config", "error", err.Error())
	}

	// Создание инстанса браузера
	browser := rod.New()
	if err := browser.Connect(); err != nil {
		logger.Fatalf("Failed to connect to browser: %v", err)
	}
	defer browser.Close()

	// Создаем контекст
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*time.Duration(cfg.Timeout)))
	defer cancel()

	taskChan := make(chan taskconfig.TaskConfig)
	resChan := make(chan interface{})

	var wg sync.WaitGroup

	// Запуск воркеров
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(ctx, browser, taskChan, resChan, logger, &wg)
	}

	// Отправка задач в цикл
	go func() {
		for _, task := range loadedTasks {
			taskChan <- task
		}
		close(taskChan)
	}()

	go func() {
		for res := range resChan {
			logger.Print("Result:", res)
		}
	}()
	wg.Wait()
	close(resChan)
}

func worker(ctx context.Context, browser *rod.Browser, taskChan <-chan taskconfig.TaskConfig, resultChan chan<- interface{}, logger *log.Logger, wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range taskChan {
		scraper := scraper.NewRodScraper(browser, task, logger)
		res, err := scraper.Scrape(ctx)
		if err != nil {
			logger.Error("Error scraping items", err)
			continue
		}
		resultChan <- res
	}
}
