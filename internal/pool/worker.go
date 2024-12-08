package pool

import (
	"sync"

	"github.com/pav-kh/yourfeed-telegram/internal/models"
)

const (
	messageBufferSize = 100
	numWorkers        = 5
)

// WorkerPool manages a pool of worker goroutines for concurrent message processing
type WorkerPool struct {
	tasks    chan *models.MessageTask
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// NewWorkerPool creates a new WorkerPool instance
func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		tasks:    make(chan *models.MessageTask, messageBufferSize),
		shutdown: make(chan struct{}),
	}
}

// Start starts the worker pool with the given number of workers
func (wp *WorkerPool) Start(workerFunc func(*models.MessageTask)) {
	wp.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go wp.worker(workerFunc)
	}
}

// worker processes messages from the task queue
func (wp *WorkerPool) worker(workerFunc func(*models.MessageTask)) {
	defer wp.wg.Done()

	for {
		select {
		case task := <-wp.tasks:
			workerFunc(task)
		case <-wp.shutdown:
			return
		}
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task *models.MessageTask) bool {
	select {
	case wp.tasks <- task:
		return true
	default:
		return false
	}
}

// Shutdown performs a graceful shutdown of the worker pool
func (wp *WorkerPool) Shutdown() {
	close(wp.shutdown)
	wp.wg.Wait()
}
