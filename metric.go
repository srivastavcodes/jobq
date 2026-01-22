package jobq

import "sync/atomic"

type Metric interface {
	WorkerMetric
	TaskOp
	TaskMetric
	WorkerOp
}

type TaskMetric interface {
	SuccessTasks() uint64
	FailedTasks() uint64
	SubmittedTasks() uint64
	CompletedTasks() uint64
}

type TaskOp interface {
	IncSuccessTasks()
	IncFailedTasks()
	IncSubmittedTasks()
}

type WorkerOp interface {
	IncBusyWorker()
	DecBusyWorker()
}

type WorkerMetric interface {
	BusyWorkers() int64
}

var _ Metric = (*metric)(nil)

type metric struct {
	busyWorkers    int64
	successTasks   uint64
	failedTasks    uint64
	submittedTasks uint64
}

func NewMetric() Metric { return &metric{} }

func (m *metric) IncBusyWorker() {
	atomic.AddInt64(&m.busyWorkers, 1)
}

func (m *metric) DecBusyWorker() {
	atomic.AddInt64(&m.busyWorkers, ^int64(0))
}

func (m *metric) BusyWorkers() int64 {
	return atomic.LoadInt64(&m.busyWorkers)
}

func (m *metric) IncSuccessTasks() {
	atomic.AddUint64(&m.successTasks, 1)
}

func (m *metric) IncFailedTasks() {
	atomic.AddUint64(&m.failedTasks, 1)
}

func (m *metric) IncSubmittedTasks() {
	atomic.AddUint64(&m.submittedTasks, 1)
}

func (m *metric) SuccessTasks() uint64 {
	return atomic.LoadUint64(&m.successTasks)
}

func (m *metric) FailedTasks() uint64 {
	return atomic.LoadUint64(&m.failedTasks)
}

func (m *metric) SubmittedTasks() uint64 {
	return atomic.LoadUint64(&m.submittedTasks)
}

func (m *metric) CompletedTasks() uint64 {
	return atomic.LoadUint64(&m.successTasks) + atomic.LoadUint64(&m.failedTasks)
}
