package jobq

import (
	"context"
	"jobq/work"
	"sync"
	"sync/atomic"
)

var _ work.Worker = (*Ring)(nil)

// TODO: create a pull req removing embedded sync.Mutex from Ring struct

// Ring represents a simple queue using a buffered channel.
type Ring struct {
	mu         sync.Mutex
	tasksQueue []work.TaskMessage // tasksQueue holds the tasks in the ring buffer.
	startFunc  JobFn              // startFunc is the function responsible for processing tasks.
	count      int                // count is the current number of tasks in the queue.
	head       int                // head is the index of the first task in the queue.
	tail       int                // tail is the index where the next task will be added.
	exit       chan struct{}      // exit is used to signal when the queue is shutting down.
	capacity   int                // capacity is the maximum number of tasks the queue can hold.
	logger     Logger             // logger is used for logging messages.
	stopOnce   sync.Once          // stopOnce ensures the shutdown process runs only once.
	stopFlag   atomic.Bool        // stopFlag indicates whether the queue is shutting down.
}

// NewRing creates a new Ring instance with the provided options. It initializes
// the task queue with the default size of 2, sets the capacity based on the
// provided options, and configures the logger and run function. The function
// returns a pointer to the newly created Ring instance.
//
// Parameters:
//   - opts: A variadic list of Option functions to configure the ring instance.
func NewRing(opts ...Option) *Ring {
	o := NewOptions(opts...)

	return &Ring{
		tasksQueue: make([]work.TaskMessage, 2),
		capacity:   o.queueSize,
		exit:       make(chan struct{}),
		logger:     o.logger,
		startFunc:  o.fn,
	}
}

// Start executes a new task using the provided context and task message.
// It calls the startFunc function, which is responsible for processing
// the tasks. The context allows for cancellation and timeout control of
// task execution.
func (r *Ring) Start(ctx context.Context, task work.TaskMessage) error {
	return r.startFunc(ctx, task)
}

// Shutdown gracefully shuts down the worker. It sets the stopFlag to indicate
// that the queue is shutting down and prevents new task from being added.
//
//	If the queue is already shutdown, it returns ErrQueueShutdown.
//
// It waits for all tasks to be processed before completing the shutdown.
func (r *Ring) Shutdown() error {
	// attempt to set the flag from false to true, if fails, the queue
	// is already shutdown.
	if !r.stopFlag.CompareAndSwap(false, true) {
		return ErrQueueShutdown
	}
	r.stopOnce.Do(func() {
		r.mu.Lock()
		count := r.count
		r.mu.Unlock()
		// If there are tasks in the queue, wait for them to be processed.
		if count > 0 {
			<-r.exit
		}
	})
	return nil
}

// EnQueue adds a task to the ring buffer queue. It returns an error if the queue
// has shutdown or has reached its maximum capacity.
func (r *Ring) EnQueue(task work.TaskMessage) error {
	// check if the queue is shutdown
	if r.stopFlag.Load() {
		return ErrQueueShutdown
	}
	// check if the queue has reached maximum capacity
	if r.capacity > 0 && r.count >= r.capacity {
		return ErrMaxCapacity
	}
	r.mu.Lock()
	if r.count == len(r.tasksQueue) {
		r.resize(r.count * 2)
	}
	r.tasksQueue[r.tail] = task
	r.tail = (r.tail + 1) % len(r.tasksQueue)
	r.count++
	r.mu.Unlock()
	return nil
}

// Request retrieves the next task message from the ring queue.
//
//	If the queue has been stopped and is empty, it signals the exit channel and
//	returns an error indicating the queue has been closed.
//
//	If the queue is empty but not stopped, it returns an error indicating there
//	are no tasks in the queue.
//
// If a task is retrieved successfully, it is removed from the queue, and the
// queue may be resized if it is less than half full.
// Returns the task message on success, or an error if the queue is empty or
// has been closed
func (r *Ring) Request() (work.TaskMessage, error) {
	if r.stopFlag.Load() && r.count == 0 {
		select {
		case r.exit <- struct{}{}:
		default:
		}
		return nil, ErrQueueHasBeenClosed
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == 0 {
		return nil, ErrNoTaskInQueue
	}
	data := r.tasksQueue[r.head]
	r.tasksQueue[r.head] = nil
	r.head = (r.head + 1) % len(r.tasksQueue)
	r.count--
	// if taskQueue is bigger than 2 times the data in it, reduce it
	// by half.
	if h := len(r.tasksQueue) / 2; h >= 2 && r.count <= h {
		r.resize(h)
	}
	return data, nil
}

// resize adjusts the size of the ring buffer to the specified capacity size.
// It reallocates the underlying slice to new size and copies the existing
// elements to the new slice in the correct order. The head and tail pointers
// are updated accordingly to maintain the correct order of elements in the
// resized buffer.
//
// Parameters:
//   - size: the new capacity of the ring buffer.
func (r *Ring) resize(size int) {
	nodes := make([]work.TaskMessage, size)

	if r.head < r.tail {
		copy(nodes, r.tasksQueue[r.head:r.tail])
	} else {
		copy(nodes, r.tasksQueue[r.head:])
		copy(nodes[len(r.tasksQueue)-r.head:], r.tasksQueue[:r.tail])
	}
	r.tail = r.count % size
	r.head = 0
	r.tasksQueue = nodes
}
