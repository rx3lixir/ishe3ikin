package appconfig

import (
	"flag"
)

// AppConfig содержит параметры конфигурации приложения.
type AppConfig struct {
	ConfigPath string
	Timeout    int
	OutputPath string
}

// LoadConfig считывает флаги командной строки и возвращает структуру конфигурации.
func NewAppConfig() *AppConfig {
	configPath := flag.String("c", "", "Path to config file")
	outputPath := flag.String("o", "output.csv", "Path to output file")
	timeOut := flag.Int("t", 10, "Set up a timeot for scraping")

	flag.Parse()

	return &AppConfig{
		ConfigPath: *configPath,
		OutputPath: *outputPath,
		Timeout:    *timeOut,
	}
}
