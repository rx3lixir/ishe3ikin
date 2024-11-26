package logger

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
)

// NewLogger создает новый экземпляр логгера с предварительно заданной конфигурацией.
func NewLogger() *log.Logger {
	logger := log.NewWithOptions(os.Stdout, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
	})
	return logger
}
