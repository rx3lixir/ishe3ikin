package main

import (
	"context"
	"log"
	"time"

	"github.com/go-rod/rod"
	"github.com/rx3lixir/ish3ikin/internal/config/appconfig"
	"github.com/rx3lixir/ish3ikin/internal/config/taskconfig"
	"github.com/rx3lixir/ish3ikin/internal/lib/logger"
	"github.com/rx3lixir/ish3ikin/internal/lib/work"
	scrp "github.com/rx3lixir/ish3ikin/internal/scraper"
)

const (
	numWorkers = 6
)

func main() {
	// Инициализация логгера
	logger := logger.NewLogger()

	// Загрузка конфигурации
	cfg := appconfig.NewAppConfig()

	// В зависимости от расширения файла конфигурации создаем лоадер
	loader := taskconfig.NewJSONLoader()

	// Создаем контекст
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*time.Duration(cfg.Timeout)))
	defer cancel()

	// Загружаем задачи
	tasks, err := loader.Load(cfg.ConfigPath)
	if err != nil {
		logger.Error("Failed to load tasks", err)
	}

	// Создаем инстанс браузера
	browser := rod.New()
	if err := browser.Connect(); err != nil {
		logger.Error("Error connecting to browser", err)
	}
	defer browser.Close()

	// Создаем новый скраппер
	scraper := scrp.NewRodScraper(browser, *logger)

	// Инициализируем воркерпул
	pool, err := work.NewPool(numWorkers, len(tasks))
	if err != nil {
		log.Fatalf("Failed to create worker pool: %v", err)
	}

	pool.Start(ctx)

	// Добавляем задачи
	for _, task := range tasks {
		scraperTask := scrp.NewScraperTask(task, ctx, scraper, *logger)
		pool.AddTask(scraperTask)
	}

	// Выводим результаты
	go func() {
		for res := range pool.Results() {
			logger.Printf("Got results: %v\n", res)
		}
	}()

	pool.Stop()
	logger.Info("All tasks completed!")
}
