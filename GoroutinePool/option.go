package GoroutinePool

import (
	"sync"
	"time"
)

// Option represents an option for the pool
type Option func(*GoroutinePool)

// WithLock sets the lock for the pool
func WithLock(lock sync.Locker) Option {
	return func(pool *GoroutinePool) {
		pool.lock = lock
		pool.cond = sync.NewCond(pool.lock)
	}
}

// WithMinWorkers sets the minimum number of the workers for the pool
func WithMinWorkers(minWorkers int) Option {
	return func(pool *GoroutinePool) {
		pool.minWorkers = minWorkers
	}
}

// WithTimeout sets the timeout for the pool
func WithTimeout(timeout time.Duration) Option {
	return func(pool *GoroutinePool) {
		pool.timeout = timeout
	}
}

// WithResultCallBack sets the result callback for the pool
func WithResultCallBack(callback func(interface{})) Option {
	return func(pool *GoroutinePool) {
		pool.resultCallback = callback
	}
}

// WithRetryCount sets the retry count for the pool.
func WithRetryCount(retryCount int) Option {
	return func(pool *GoroutinePool) {
		pool.retryCount = retryCount
	}
}

// WithTaskQueueSize sets the size of the task queue for the pool.
func WithTaskQueueSize(size int) Option {
	return func(pool *GoroutinePool) {
		pool.taskQueueSize = size
	}
}
