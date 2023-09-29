package GoroutinePool

import (
	"context"
	"sort"
	"sync"
	"time"
)

type Pool interface {
	// Submit 提交任务
	Submit(task Task)
	// Wait 等待执行任务
	Wait()
	// Release 释放协程池
	Release()
	// GetRunning 获取运行中的协程数量
	GetRunning() int
	// GetWorkers 获取工作协程数量
	GetWorkers() int
	// GetTaskQueenSize 获取任务队列中的任务数量
	GetTaskQueenSize() int
}

type Task func() (interface{}, error)

type GoroutinePool struct {
	lock           sync.Locker
	workers        []*Worker
	workerStack    []int
	maxWorkers     int
	minWorkers     int
	taskQueue      chan Task
	taskQueueSize  int
	retryCount     int
	cond           *sync.Cond
	timeout        time.Duration
	resultCallback func(interface{})
	errCallback    func(error)
	adjustInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewGoroutinePool(maxWorkers int, options ...Option) *GoroutinePool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &GoroutinePool{
		lock:           new(sync.Mutex),
		maxWorkers:     maxWorkers,
		minWorkers:     maxWorkers,
		workers:        nil,
		workerStack:    nil,
		taskQueue:      nil,
		taskQueueSize:  1e6,
		retryCount:     0,
		timeout:        0,
		cond:           nil,
		adjustInterval: 1 * time.Second,
		ctx:            ctx,
		cancel:         cancel,
	}
	// apply options
	for _, opt := range options {
		opt(pool)
	}
	pool.taskQueue = make(chan Task, pool.taskQueueSize)
	pool.workers = make([]*Worker, pool.minWorkers)
	pool.workerStack = make([]int, pool.minWorkers)

	if pool.cond == nil {
		pool.cond = sync.NewCond(pool.lock)
	}
	// create workers
	for i := 0; i < pool.minWorkers; i++ {
		worker := newWorker()
		pool.workers[i] = worker
		pool.workerStack[i] = i
		// 真正去执行任务
		worker.start(pool, i)
	}
	// process requests
	go pool.adjustWorkers()
	go pool.dispatch()
	return pool
}

func (pool *GoroutinePool) Submit(task Task) {
	pool.taskQueue <- task
}

// Wait waits for all tasks to be dispatched and completed
func (pool *GoroutinePool) Wait() {
	for {
		pool.lock.Lock()
		workerStackLen := len(pool.workerStack)
		pool.lock.Unlock()

		if len(pool.taskQueue) == 0 && workerStackLen == len(pool.workers) {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func (pool *GoroutinePool) Release() {
	// 不再接受后续的请求
	close(pool.taskQueue)
	pool.cancel()
	pool.cond.L.Lock()
	// 等待现行所有任务执行完成
	for len(pool.workerStack) != pool.minWorkers {
		pool.cond.Wait()
	}
	pool.cond.L.Unlock()
	for _, worker := range pool.workers {
		close(worker.taskQueue)
	}
	pool.workers = nil
	pool.workerStack = nil
}

// GetRunning 获取运行中的协程数量
func (pool *GoroutinePool) GetRunning() int {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	return len(pool.workers) - len(pool.workerStack)
}

// GetWorkers 获取工作协程数量
func (pool *GoroutinePool) GetWorkers() int {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	return len(pool.workers)
}

// GetTaskQueenSize 获取任务队列中的任务数量
func (pool *GoroutinePool) GetTaskQueenSize() int {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	return pool.taskQueueSize
}

func (pool *GoroutinePool) popWorker() int {
	pool.lock.Lock()
	workerIndex := pool.workerStack[len(pool.workerStack)-1]
	pool.workerStack = pool.workerStack[:len(pool.workerStack)-1]
	pool.lock.Unlock()
	return workerIndex
}

func (pool *GoroutinePool) pushWorker(workerIndex int) {
	pool.lock.Lock()
	pool.workerStack = append(pool.workerStack, workerIndex)
	pool.lock.Unlock()
	// 加入/归还了新的worker，唤醒阻塞的任务
	pool.cond.Signal()
}

func (pool *GoroutinePool) adjustWorkers() {
	ticker := time.NewTicker(pool.adjustInterval)
	defer ticker.Stop()

	var adjustFlag bool

	for {
		adjustFlag = false
		select {
		case <-ticker.C:
			pool.cond.L.Lock()
			if len(pool.taskQueue) > len(pool.workers)*3/4 && len(pool.workers) < pool.maxWorkers {
				// 扩容
				adjustFlag = true
				// double the number of workers until it reaches the maximum
				newWorkerNum := min(len(pool.workers)*2, pool.maxWorkers) - len(pool.workers)
				for i := 0; i < newWorkerNum; i++ {
					worker := newWorker()
					pool.workers = append(pool.workers, worker)
					pool.workerStack = append(pool.workerStack, len(pool.workers)-1)
					worker.start(pool, len(pool.workers)-1)
				}
			} else if len(pool.taskQueue) == 0 && len(pool.workerStack) == len(pool.workers) && len(pool.workers) > pool.minWorkers {
				adjustFlag = true
				removeWorkerNum := (len(pool.workers) - pool.minWorkers + 1) / 2
				// sort the workIndex before removing workers
				sort.Ints(pool.workerStack)
				pool.workers = pool.workers[:len(pool.workers)-removeWorkerNum]
				pool.workerStack = pool.workerStack[:len(pool.workerStack)-removeWorkerNum]
			}
			pool.cond.L.Unlock()
			if adjustFlag {
				// 唤醒所有的任务
				pool.cond.Broadcast()
			}
		case <-pool.ctx.Done():
			return
		}
	}
}

func (pool *GoroutinePool) dispatch() {
	for t := range pool.taskQueue {
		pool.cond.L.Lock()
		// 没有可用的worker，等待
		for len(pool.workerStack) == 0 {
			pool.cond.Wait()
		}
		pool.cond.L.Unlock()
		workerIndex := pool.popWorker()
		pool.workers[workerIndex].taskQueue <- t
	}
}
