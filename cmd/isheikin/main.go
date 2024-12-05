package main

import (
	"context"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-rod/rod"
	"github.com/rx3lixir/ish3ikin/internal/config/appconfig"
	"github.com/rx3lixir/ish3ikin/internal/config/taskconfig"
	"github.com/rx3lixir/ish3ikin/internal/lib/logger"
	"github.com/rx3lixir/ish3ikin/internal/lib/work"
	"github.com/rx3lixir/ish3ikin/internal/scraper"
)

const (
	workerCount = 1
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

	// Определяем длину очереди для воркер пула
	queueSize := len(loadedTasks)

	pool := work.NewWorkerPool(workerCount, queueSize)
	pool.Run()

	for _, task := range loadedTasks {
		scrapeTask := &ScrapeTask{
			ctx:     ctx,
			browser: browser,
			task:    task,
			logger:  logger,
		}
		pool.AddTask(scrapeTask)
	}

	pool.Shutdown()
}

// ScrapeTask реализует интерфейс Task.
type ScrapeTask struct {
	ctx     context.Context
	browser *rod.Browser
	task    taskconfig.TaskConfig
	logger  *log.Logger
}

func (st *ScrapeTask) Execute() (interface{}, error) {
	scraper := scraper.NewRodScraper(st.browser, st.task, st.logger)
	res, err := scraper.Scrape(st.ctx)
	if err != nil {
		st.logger.Error("Error scraping items", err)
		return nil, err
	}
	st.logger.Printf("Founded items: %v", res)
	return res, nil
}
