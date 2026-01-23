package jobq

import "errors"

var (
	ErrNoTaskInQueue      = errors.New("\x1b[31mjobq ::\x1b[0m no task in queue")
	ErrQueueHasBeenClosed = errors.New("\x1b[31mjobq ::\x1b[0m queue has been closed")
	ErrMaxCapacity        = errors.New("\x1b[31mjobq ::\x1b[0m max capacity reached")
	ErrQueueShutdown      = errors.New("\x1b[31mjobq ::\x1b[0m queue has been closed and released")
)
