package work

import (
	"context"
	"errors"
	"sync"
)

type Executor interface {
	Execute() error
	OnError(error)
}

type Pool struct {
	numWorkers     int
	tasks          chan Executor
	tasksCompleted chan bool
	start          sync.Once
	stop           sync.Once
	quit           chan struct{}
	wg             sync.WaitGroup
}

// Создает новый пул воркеров с заданными параметрами
func NewPool(numWorkers int, taskChannelSize int) (*Pool, error) {
	if numWorkers <= 0 || taskChannelSize <= 0 {
		return nil, errors.New("Invalid parameters: number of workers and tasks must be more than zero")
	}
	return &Pool{
		numWorkers:     numWorkers,
		tasks:          make(chan Executor, taskChannelSize),
		tasksCompleted: make(chan bool),
		start:          sync.Once{},
		stop:           sync.Once{},
		quit:           make(chan struct{}),
	}, nil
}

func (p *Pool) Start(ctx context.Context) {
	p.start.Do(func() {
		//p.startWorker(ctx)
	})
}

func (p *Pool) Stop() {
	p.stop.Do(func() {
		close(p.quit)
		p.wg.Wait()
		close(p.tasksCompleted)
	})
}
