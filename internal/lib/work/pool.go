package work

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Executor interface {
	Execute() (interface{}, error)
	OnError(error)
}

type Pool struct {
	numWorkers     int
	tasks          chan Executor
	results        chan interface{}
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
		results:        make(chan interface{}),
		tasksCompleted: make(chan bool),
		start:          sync.Once{},
		stop:           sync.Once{},
		quit:           make(chan struct{}),
	}, nil
}

// Функция для получения канала результатов
func (p *Pool) Results() <-chan interface{} {
	return p.results
}

func (p *Pool) Start(ctx context.Context) {
	p.start.Do(func() {
		p.startWorker(ctx)
	})
}

func (p *Pool) Stop() {
	p.stop.Do(func() {
		close(p.quit)
		p.wg.Wait()
		close(p.tasksCompleted)
	})
}

func (p *Pool) AddTask(t Executor) {
	select {
	case p.tasks <- t:
	case <-p.quit:
	}
}

func (p *Pool) TaskCompleted() <-chan bool {
	return p.tasksCompleted
}

func (p *Pool) startWorker(ctx context.Context) {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1) // Увеличиваем счетчик ожидания
		go func(workerNum int) {
			defer p.wg.Done() // Уменьшаем счетчик при завершении воркера
			fmt.Printf("worker number: %v started\n", workerNum)
			for {
				select {
				case <-ctx.Done():
					return
				case <-p.quit:
					return
				case task, ok := <-p.tasks:
					if !ok {
						return
					}
					res, err := task.Execute()
					if err != nil {
						task.OnError(err)
						continue
					}
					select {
					case p.tasksCompleted <- true:
					case p.results <- res:
					default: // Предотвращаем блокировку, если никто не слушает канал
					}
					fmt.Printf("worker number %d finished a task\n", workerNum)
				}
			}
		}(i)
	}
}
