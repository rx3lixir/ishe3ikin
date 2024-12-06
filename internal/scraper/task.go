package scraper

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/rx3lixir/ish3ikin/internal/config/taskconfig"
)

type ScraperTask struct {
	Task    taskconfig.Task
	Context context.Context
	Scraper Scraper
	Logger  *log.Logger
}

func NewScraperTask(task taskconfig.Task, ctx context.Context, scraper Scraper, logger log.Logger) *ScraperTask {
	return &ScraperTask{
		Task:    task,
		Context: ctx,
		Scraper: scraper,
		Logger:  &logger,
	}
}

func (s *ScraperTask) Execute() (interface{}, error) {
	res, err := s.Scraper.Scrape(s.Context, s.Task)
	if err != nil {
		return nil, err
	}

	s.Logger.Infof("Scraped Result for %v: %s", s.Task.URL, res)
	return res, nil
}

func (s *ScraperTask) OnError(err error) {
	s.Logger.Error("Failed to scrape a task")
}
