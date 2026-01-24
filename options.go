package jobq

import (
	"context"
	"jobq/work"
	"runtime"
	"time"
)

var defaultFn = func(context.Context, work.TaskMessage) error { return nil }

var (
	defaultCapacity  = 0
	defaultWorkers   = int64(runtime.NumCPU())
	defaultNewLogger = NewLogger()
	defaultMetric    = NewMetric()
)

// JobFn represents a job function for the workers to execute.
type JobFn func(context.Context, work.TaskMessage) error

// Option configures a mutex.
type Option interface {
	apply(*Options)
}

// OptionFunc is a function that configures a queue.
type OptionFunc func(*Options)

func (of OptionFunc) apply(option *Options) {
	of(option)
}

// WithWorkerCount sets worker count.
func WithWorkerCount(num int64) Option {
	return OptionFunc(func(o *Options) {
		if num <= 0 {
			num = defaultWorkers
		}
		o.maxWorkerCount = num
	})
}

// WithQueueSize sets queue size.
func WithQueueSize(size int) Option {
	return OptionFunc(func(o *Options) {
		o.queueSize = size
	})
}

// WithLogger sets custom logger.
func WithLogger(l Logger) Option {
	return OptionFunc(func(o *Options) {
		o.logger = l
	})
}

// WithMetric sets custom Metric.
func WithMetric(m Metric) Option {
	return OptionFunc(func(o *Options) {
		o.metric = m
	})
}

// WithWorker sets custom worker.
func WithWorker(w work.Worker) Option {
	return OptionFunc(func(o *Options) {
		o.worker = w
	})
}

// WithFn sets custom job function
func WithFn(fn func(context.Context, work.TaskMessage) error) Option {
	return OptionFunc(func(o *Options) {
		o.fn = fn
	})
}

// WithAfterFn sets a callback function for when the job is done.
func WithAfterFn(afterFn func()) Option {
	return OptionFunc(func(o *Options) {
		o.afterFn = afterFn
	})
}

// WithRetryInterval sets the retry interval.
func WithRetryInterval(d time.Duration) Option {
	return OptionFunc(func(o *Options) {
		o.retryInterval = d
	})
}

// Options for custom args in Queue.
type Options struct {
	maxWorkerCount int64
	logger         Logger
	queueSize      int
	worker         work.Worker
	fn             JobFn
	afterFn        func()
	metric         Metric
	retryInterval  time.Duration
}

// NewOptions initialize the default value for the options
func NewOptions(opts ...Option) *Options {
	options := &Options{
		maxWorkerCount: defaultWorkers,
		queueSize:      defaultCapacity,
		logger:         defaultNewLogger,
		worker:         nil,
		fn:             defaultFn,
		metric:         defaultMetric,
		retryInterval:  time.Second,
	}
	for _, o := range opts {
		o.apply(options)
	}
	return options
}
