package taskconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

// TaskConfig описывает конфигурацию для скрапинга.
type TaskConfig struct {
	URL       string
	Type      string
	Name      string
	Selectors map[string]string
}

// Loader определяет интерфейс загрузки конфигурации.
type Loader interface {
	Load(filePath string) ([]TaskConfig, error)
}

// JSONConfigLoader реализует загрузку из JSON.
type JSONTasksLoader struct{}

func (j *JSONTasksLoader) Load(filePath string) ([]TaskConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configs []TaskConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return configs, nil
}
