package jobq

import "errors"

var (
	ErrNoTaskInQueue      = errors.New("no task in queue")
	ErrQueueHasBeenClosed = errors.New("queue has been closed")
	ErrMaxCapacity        = errors.New("max capacity reached")

	// ErrQueueShutdown is returned when an operation is performed
	// on a queue that has already been shutdown and released.
	ErrQueueShutdown = errors.New("queue has been closed and released")

	// ErrMissingWorker is returned when a queue is created without
	// a worker implementation
	ErrMissingWorker = errors.New("missing worker module")
)
