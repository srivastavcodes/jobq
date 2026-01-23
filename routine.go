package jobq

import "sync"

// routineGroup is a thin wrapper around sync.WaitGroup. Initialize it with
// new(routineGroup).
type routineGroup struct {
	wg sync.WaitGroup
}

// Run spins a new goroutine which executes the provided function. Use Wait
// to guarantee that the function gets executed.
func (r *routineGroup) Run(fn func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		fn()
	}()
}

// Wait blocks the execution flow until all the goroutines have exited
// successfully. See Wait of sync.WaitGroup.
func (r *routineGroup) Wait() {
	r.wg.Wait()
}
