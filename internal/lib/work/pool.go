package work

import (
	"fmt"
	"sync"
)

type Task interface {
	Execute() (interface{}, error)
}

type WorkerPool struct {
	resultChan  chan interface{}
	taskQueue   chan Task
	workerCount int
	wg          sync.WaitGroup
}

func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	return &WorkerPool{
		taskQueue:   make(chan Task, queueSize),
		workerCount: workerCount,
		resultChan:  make(chan interface{}, queueSize),
	}
}

func (wp *WorkerPool) Run() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) Results() <-chan interface{} {
	return wp.resultChan
}

func (wp *WorkerPool) AddTask(task Task) {
	wp.taskQueue <- task
}

func (wp *WorkerPool) Shutdown() {
	close(wp.taskQueue)
	wp.wg.Wait()
	close(wp.resultChan)
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for task := range wp.taskQueue {
		fmt.Printf("Worker %d processing task...\n", id)
		result, err := task.Execute()
		if err != nil {
			fmt.Printf("Worker %d encountered an error: %v\n", id, err)
		} else {
			wp.resultChan <- result
		}
	}

	fmt.Printf("Worker %d shutting down.\n", id)
}
