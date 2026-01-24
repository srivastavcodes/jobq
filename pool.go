package jobq

// NewPool initializes a new pool.
func NewPool(size int64, opts ...Option) *Queue {
	o := []Option{
		WithWorkerCount(size),
		WithWorker(NewRing(opts...)),
	}
	o = append(o, opts...)
	queue, err := NewQueue(o...)
	if err != nil {
		panic("error creating queue: " + err.Error())
	}
	queue.Start()
	return queue
}
