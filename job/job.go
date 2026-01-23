package job

import (
	"context"
	"encoding/json"
	"jobq/work"
	"time"
)

// TaskFunc is the task function
type TaskFunc func(ctx context.Context) error

// Message describes a task and its metadata.
type Message struct {
	Task TaskFunc `json:"-"`

	// Timeout is the duration for which the task can be processed by the handler.
	Timeout time.Duration `json:"timeout"`

	// Body is payload data of the task.
	Body []byte `json:"body"`

	// RetryCount sets the retry count; default is 0.
	RetryCount int64 `json:"retry_count"`

	// RetryDelay sets delay between retries; default is 100ms.
	RetryDelay time.Duration `json:"retry_delay"`

	// RetryFactor is the multiplying factor for each increment step; defaults to 2.
	RetryFactor float64 `json:"retry_factor"`

	// Jitter eases contention by randomizing backoff steps.
	Jitter bool `json:"jitter"`

	// RetryMin is the minimum value of the counter; defaults to 100ms.
	RetryMin time.Duration `json:"retry_min"`

	// RetryMax is the maximum value of the counter; defaults to 10s.
	RetryMax time.Duration `json:"retry_max"`
}

// Payload returns the payload data of the Message. It returns the byte
// slice of the payload.
func (m *Message) Payload() []byte { return m.Body }

func (m *Message) Bytes() []byte {
	bytes, err := json.Marshal(m.Body)
	if err != nil {
		return nil
	}
	return bytes
}

// NewMessage create a new message
func NewMessage(m work.QueuedMessage, opts ...AllowOption) Message {
	o := NewOptions(opts...)

	return Message{
		RetryCount:  o.retryCount,
		RetryDelay:  o.retryDelay,
		RetryFactor: o.retryFactor,
		RetryMin:    o.retryMin,
		RetryMax:    o.retryMax,
		Body:        m.Bytes(),
		Timeout:     o.timeout,
	}
}

func NewTask(task TaskFunc, opts ...AllowOption) Message {
	o := NewOptions(opts...)

	return Message{
		Timeout:     o.timeout,
		Task:        task,
		RetryCount:  o.retryCount,
		RetryDelay:  o.retryDelay,
		RetryFactor: o.retryFactor,
		RetryMin:    o.retryMin,
		RetryMax:    o.retryMax,
	}
}

// Encode takes a Message struct and marshals it into a byte slice using msgpack.
// If the marshalling process encounters an error, the function will panic.
// It returns the marshalled byte slice.
//
// Parameters:
//   - m: A pointer to the Message struct to be encoded.
func Encode(m *Message) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return b
}

// Decode takes a byte slice and unmarshals it into a Message struct using msgpack.
// If the unmarshalling process encounters an error, the function will panic.
// It returns a pointer to the unmarshalled Message.
//
// Parameters:
//   - b: A byte slice containing the msgpack-encoded data.
func Decode(b []byte) *Message {
	var msg Message
	if err := json.Unmarshal(b, &msg); err != nil {
		panic(err)
	}
	return &msg
}
