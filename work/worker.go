package work

import "context"

// Worker represents an interface for a work that processes tasks.
// It provides methods to run tasks, shut down the work, queue
// tasks, and request tasks from the queue.
type Worker interface {
	// Start starts the work and processes the given task in the provided
	// context. It returns an error if task cannot be processed.
	Start(ctx context.Context, task TaskMessage) error

	// Shutdown stops the work and performs any necessary cleanup.
	// It returns an error if the shutdown process fails.
	Shutdown() error

	// EnQueue adds a task to the work's queue. Errors if cannot enqueue.
	EnQueue(task TaskMessage) error

	// Request retrieves a task from the work's queue. It returns the
	// queued message or errors if failed.
	Request() (TaskMessage, error)
}

// QueuedMessage represents an interface for a message that can be queued.
// It requires an implementation of the Bytes method, which returns the
// message content as a slice of bytes.
type QueuedMessage interface {
	Bytes() []byte
}

// TaskMessage represents an interface for a task message that can be queued.
// It embeds QueuedMessage and requires the implementation of a Payload method
// to retrieve the payload of the message.
type TaskMessage interface {
	QueuedMessage
	Payload() []byte
}
