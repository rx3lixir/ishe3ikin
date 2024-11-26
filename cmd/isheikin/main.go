package main

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/rx3lixir/icheikin/internal/config/appconfig"
	"github.com/rx3lixir/icheikin/internal/config/taskconfig"
	"github.com/rx3lixir/icheikin/internal/exporter/csv"
	"github.com/rx3lixir/icheikin/internal/lib/logger"
	"github.com/rx3lixir/icheikin/internal/scraper"
)

func main() {
	// Инициализируем логгер
	logger := logger.NewLogger()

	// Загружаем конфигурацию
	cfg := appconfig.LoadAppConfig()
	if cfg.ConfigPath == "" {
		logger.Fatal("No config file provided. Use -c to specify config path.")
	}

	// Загружаем задачи для скраппинга
	taskLoader := taskconfig.JSONTasksLoader{}
	tsk, err := taskLoader.Load(cfg.ConfigPath)
	if err != nil {
		logger.Fatal("Faled to load config", "error", err.Error())
	}

	// Создаем инстанс браузера для работы
	browser := rod.New()
	if err := browser.Connect(); err != nil {
		logger.Fatalf("Failed to connect to browser: %v", err)
	}
	defer browser.Close()

	// Создаем задачи для скраппинга
	var tasks []scraper.Scraper
	for _, t := range tsk {
		tasks = append(tasks, scraper.NewRodScraper(browser, t, logger))
	}

	// Cоздаем контекст для скраппинга
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	// Запускаем задачи
	runner := scraper.NewTaskRunner(tasks)
	results, errors := runner.Run(ctx)

	// Логирование ошибок
	for _, err := range errors {
		logger.Error("Scraping error", "error", err)
	}

	// Экспорт данных
	exporter := exporter.NewCSVExporter(cfg.OutputPath)
	if err := exporter.Export(results); err != nil {
		logger.Fatalf("Failed to export data: %v", err)
	} else {
		logger.Info("✅Data exported succesfuly!")
	}
}
