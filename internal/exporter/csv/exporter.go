package exporter

import (
	"encoding/csv"
	"fmt"
	"os"
)

// Exporter интерфейс для экспорта данных.
type Exporter interface {
	Export(data []map[string]string) error
}

// CSVExporter экспортирует данные в CSV.
type CSVExporter struct {
	FileName string
}

// NewCSVExporter создаёт новый экземпляр CSVExporter.
func NewCSVExporter(fileName string) *CSVExporter {
	return &CSVExporter{FileName: fileName}
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
