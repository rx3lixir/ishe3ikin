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
func LoadAppConfig() *AppConfig {
	configPath := flag.String("c", "", "Path to config file")
	timeout := flag.Int("t", 30, "Timeout for each scraping task in seconds")
	outputPath := flag.String("o", "output.csv", "Path to output file")
	flag.Parse()

	return &AppConfig{
		ConfigPath: *configPath,
		Timeout:    *timeout,
		OutputPath: *outputPath,
	}
}
