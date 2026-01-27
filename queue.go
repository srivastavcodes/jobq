package jobq

import (
	"context"
	"errors"
	"jobq/job"
	"jobq/work"
	"sync"
	"sync/atomic"
	"time"
)

// Queue is a message queue with worker management, job scheduling,
// retry logic, and graceful shutdown capabilities.
type Queue struct {
	rg *routineGroup // rg is used to manage and wait for goroutines.
	mu sync.Mutex

	metric         *metric // metric tracks queue and worker stats.
	logger         Logger
	maxWorkerCount int64 // maxWorkerCount is the number of worker goroutines to process jobs.

	quit   chan struct{} // quit is used as a signal to shut down all goroutines.
	ready  chan struct{} // ready signals worker readiness.
	notify chan struct{} // notify notifies workers of a new job.

	worker work.Worker // worker implementation that processes jobs.

	stopOnce      sync.Once     // stopOnce ensures shutdown is only performed once.
	stopFlag      atomic.Bool   // stopFlag indicates if shutdown has already started.
	afterFn       func()        // optional callback after each job execution.
	retryInterval time.Duration // interval between each retry request.
}

func NewQueue(opts ...Option) (*Queue, error) {
	o := NewOptions(opts...)
	q := &Queue{
		maxWorkerCount: o.maxWorkerCount,
		metric:         &metric{},
		logger:         o.logger,
		quit:           make(chan struct{}),
		ready:          make(chan struct{}, 1),
		notify:         make(chan struct{}, 1),
		worker:         o.worker,
		afterFn:        o.afterFn,
		retryInterval:  o.retryInterval,
		rg:             new(routineGroup),
	}
	if q.worker == nil {
		return nil, ErrMissingWorker
	}
	return q, nil
}

// Start launches all worker goroutines and begins processing jobs.
// If worker count is zero, Start is a no-op.
func (q *Queue) Start() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.maxWorkerCount == 0 {
		return
	}
	q.rg.Run(func() { q.start() })
}

// EnQueue enqueues a single job (work.QueuedMessage) into the queue.
// Accepts job options for customizations.
func (q *Queue) EnQueue(msg work.QueuedMessage, opts ...job.AllowOption) error {
	data := job.NewMessage(msg, opts...)
	return q.queue(&data)
}

// EnQueueTask enqueues a single task function into the queue.
// Accepts job options for customizations.
func (q *Queue) EnQueueTask(task job.TaskFunc, opts ...job.AllowOption) error {
	data := job.NewTask(task, opts...)
	return q.queue(&data)
}

// queue is an internal helper to enqueue a job.Message into the worker.
// It increments the submitted tasks metric and notifies workers if possible
func (q *Queue) queue(msg *job.Message) error {
	if q.stopFlag.Load() {
		return ErrQueueShutdown
	}
	if err := q.worker.Submit(msg); err != nil {
		return err
	}
	q.metric.IncSubmittedTasks()
	// we try to notify a worker that a new job is available, if the
	// channel is full means worker is busy, hence we skip and avoid
	// blocking.
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return nil
}

// Shutdown initiates a graceful shutdown of the queue. It signals all
// goroutines to stop, shuts down the worker, and closes the quit chan.
// Shutdown is idempotent and safe to call multiple times.
func (q *Queue) Shutdown() {
	if !q.stopFlag.CompareAndSwap(false, true) {
		return
	}
	q.stopOnce.Do(func() {
		if q.BusyWorkers() > 0 {
			q.logger.Infof("shutting down all tasks: %d workers", q.BusyWorkers())
		}
		if err := q.worker.Shutdown(); err != nil {
			q.logger.Errorf("failed to shutdown worker: %v", err)
		}
		close(q.quit)
	})
}

// Release performs a graceful shutdown and waits for all goroutines
// to finish.
func (q *Queue) Release() {
	q.Shutdown()
	q.Wait()
}

// UpdateWorkerCount updates the number of worker goroutines. It also
// triggers scheduling to adjust to the new worker count.
func (q *Queue) UpdateWorkerCount(count int64) {
	q.mu.Lock()
	q.maxWorkerCount = count
	q.mu.Unlock()
	q.signalWorker()
}

// BusyWorkers returns the number of workers currently processing jobs.
func (q *Queue) BusyWorkers() int64 {
	return q.metric.BusyWorkers()
}

// SuccessTasks returns the number of successfully completed tasks.
func (q *Queue) SuccessTasks() uint64 {
	return q.metric.SuccessTasks()
}

// FailedTasks returns the number of failed tasks.
func (q *Queue) FailedTasks() uint64 {
	return q.metric.FailedTasks()
}

// SubmittedTasks returns the number of tasks submitted to the queue.
func (q *Queue) SubmittedTasks() uint64 {
	return q.metric.SubmittedTasks()
}

// CompletedTasks returns the total number of completed tasks (success + failure).
func (q *Queue) CompletedTasks() uint64 {
	return q.metric.CompletedTasks()
}

// Wait blocks until all goroutines in the routine group have finished.
func (q *Queue) Wait() {
	q.rg.Wait()
}

// work executes a single task, handling panics and updating metrics accordingly.
// After execution, it schedules the next worker if needed
func (q *Queue) work(task work.TaskMessage) {
	var err error
	// defer block to handle panics, update metrics, and run afterFn callback.
	defer func() {
		q.metric.DecBusyWorker()
		recErr := recover()
		if recErr != nil {
			q.logger.Errorf("")
		}
		q.signalWorker()

		// update success or failure metric based on execution results.
		if err == nil && recErr == nil {
			q.metric.IncSuccessTasks()
		} else {
			q.metric.IncFailedTasks()
		}
		if q.afterFn != nil {
			q.afterFn()
		}
	}()
	if err = q.run(task); err != nil {
		q.logger.Errorf("failed to run task: %v", err)
	}
}

// signalWorker checks if more workers can be started based on the current busy
// count. If so, it signals readiness to start a new worker.
func (q *Queue) signalWorker() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.maxWorkerCount <= q.BusyWorkers() {
		return
	}
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// run dispatches the task to the appropriate handler based on its type.
// Returns an error if the task type is invalid.
func (q *Queue) run(task work.TaskMessage) error {
	switch t := task.(type) {
	case *job.Message:
		return q.handle(t)
	default:
		return errors.New("invalid task type")
	}
}

// handle executes a job.Message, supporting - retries, timeouts, and panic
// recovery. Returns an error if the job fails or times out.
func (q *Queue) handle(msg *job.Message) error {
	// resultch: receives the result of the job execution mainly whether an error occurred or not.
	// panicch: receives any panic that may have occurred
	var (
		resultch  = make(chan error, 1)
		panicch   = make(chan any, 1)
		startTime = time.Now()
	)
	ctx, cancel := context.WithTimeout(context.Background(), msg.Timeout)
	defer cancel()

	// running the job in a separate goroutine to support timeout and panic rec.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicch <- r
			}
		}()
		var err error

		// backoff setup for retry logic
		backoff := Backoff{
			Min:    msg.RetryMin,
			Max:    msg.RetryMax,
			Jitter: msg.Jitter,
			Factor: msg.RetryFactor,
		}
		delay := msg.RetryDelay
		for {
			if msg.TaskFunc != nil {
				err = msg.TaskFunc(ctx)
			} else {
				err = q.worker.Start(ctx, msg)
			}
			// break if no error occurred (task success) or no retries left.
			if err == nil || msg.RetryCount == 0 {
				break
			}
			// decreasing cause, err != nil according to above logic.
			msg.RetryCount--

			if msg.RetryDelay <= 0 {
				delay = backoff.Duration()
			}
			select {
			case <-time.After(delay): // wait before retrying
				q.logger.Infof("retries remaining=%d, current delay=%d", msg.RetryCount, delay)
			case <-ctx.Done(): // timeout
				err = ctx.Err()
				break
			}
		}
		resultch <- err
	}()
	select {
	case p := <-panicch:
		panic(p)
	case err := <-resultch: // job finished
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-q.quit: // queue is shutting down
		// cancel job and wait for remaining time or job completion
		cancel()
		remaining := msg.Timeout - time.Since(startTime)
		select {
		case p := <-panicch:
			panic(p)
		case err := <-resultch:
			return err
		case <-time.After(remaining):
			return context.DeadlineExceeded
		}
	}
}

// start launches the main worker loop, which manages job scheduling and task
// execution.
// - It uses a ticker to periodically retry job requests if the queue is empty.
// - For each available worker slot, it requests a new task from the worker.
// - If a task is available, it is sent to be processed by a new goroutine.
// - The loop exists when the quit channel is closed.
func (q *Queue) start() {
	ticker := time.NewTicker(q.retryInterval)
	defer ticker.Stop()
	for {
		// ensure the number of busy workers does not exceed the configured
		// worker count.
		q.signalWorker()
		select {
		case <-q.ready: // wait for a worker slot to become available.
		case <-q.quit:
			return
		}
		task, err := q.requestTaskWithRetry(ticker.C)
		if err != nil {
			q.signalWorker()
			if errors.Is(err, context.Canceled) {
				return
			}
			continue
		}
		// start processing the new task in a separate goroutine
		q.metric.IncBusyWorker()
		q.rg.Run(func() {
			q.work(task)
		})
	}
}

// requestTaskWithRetry attempts to get a task, retrying on an empty queue.
func (q *Queue) requestTaskWithRetry(ticker <-chan time.Time) (work.TaskMessage, error) {
	for {
		task, err := q.worker.Request()
		if task != nil {
			return task, err
		}
		// permanent error
		if err != nil && !errors.Is(err, ErrNoTaskInQueue) {
			return nil, err
		}
		select {
		case <-q.quit:
			return nil, context.Canceled
		case <-ticker: // retry
		case <-q.notify: // new task might be available
		}
	}
}
