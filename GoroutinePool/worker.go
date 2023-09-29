package GoroutinePool

import (
	"context"
	"errors"
)

type Worker struct {
	taskQueue chan Task
}

func newWorker() *Worker {
	return &Worker{
		taskQueue: make(chan Task, 1),
	}
}

// start starts the worker in a separate goroutine.
// The worker will run Tasks from its taskQueue until the taskQueue is closed.
// For the length of the taskQueue is 1, the worker will be pushed back to the pool after executing 1 Task
func (w *Worker) start(pool *GoroutinePool, workerIndex int) {
	go func() {
		for t := range w.taskQueue {
			if t != nil {
				result, err := w.executeTask(t, pool)
				w.handleResult(result, err, pool)
			}
			// 虽然还有任务，但当前worker可以被重新分发任务，因此视作是归还了任务
			pool.pushWorker(workerIndex)
		}
	}()
}

func (w *Worker) executeTask(t Task, pool *GoroutinePool) (interface{}, error) {
	for i := 0; i <= pool.retryCount; i++ {
		var (
			result interface{}
			err    error
		)
		if pool.timeout > 0 {
			result, err = w.executeTaskWithTimeout(t, pool)
		} else {
			result, err = w.executeTaskWithoutTimeout(t)
		}
		if err == nil || i == pool.retryCount {
			return result, err
		}
	}
	return nil, nil
}

func (w *Worker) executeTaskWithTimeout(t Task, pool *GoroutinePool) (interface{}, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), pool.timeout)
	defer cancel()

	// Create a channel to receive the result of the task
	resultChan := make(chan interface{})
	errChan := make(chan error)

	// Run the task in a separate goroutine
	go func() {
		res, err := t()
		select {
		case resultChan <- res:
		case errChan <- err:
		case <-ctx.Done():
			// The context was canceled, stop the task
			return
		}
	}()

	var (
		result interface{}
		err    error
	)

	// Wait for the task to finish or for the context to timeout
	select {
	case result = <-resultChan:
		err = <-errChan
		return result, err
	case <-ctx.Done():
		// The context wa timeout, the task took too long
		return nil, errors.New("task timeout")
	}
}

func (w *Worker) executeTaskWithoutTimeout(t Task) (interface{}, error) {
	return t()
}

func (w *Worker) handleResult(result interface{}, err error, pool *GoroutinePool) {
	if err != nil && pool.errCallback != nil {
		pool.errCallback(err)
	} else if pool.resultCallback != nil {
		pool.resultCallback(result)
	}
}
