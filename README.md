# JobQ

JobQ is a small, in-memory job queue written in Go. It's built to be simple, predictable, and easy to reason about.

The core idea behind it is - submit jobs, run them with a fixed number of worker, support retries and timeouts, and shutdown
cleanly.

![couldn't load image: producer->ring buffer->consumer](assets/flow-01.svg)

## Features

- In memory queue (ring buffer).
   - Although it uses an interface internally, so the implementation can be swapped out.
- Worker metrics (submitted, running, success, failures).
- Fixed worker pool with dynamic sizing.
- Context-aware execution (timeouts + cancellation).
- Job retries with backoff and jitter.

## Usage

### Using the Task Function

You can directly schedule tasks by using the `EnQueueTask()` method to be executed in the background. 

```go
func usingTaskFunc() {
    var (
        num = 100
        res = make(chan string, num)
    )
    queue := jobq.NewPool(5)
    defer queue.Release()

    for i := 0; i < num; i++ {
        go func (i int) {
            _ = queue.EnQueueTask(func (ctx context.Context) error {
                res <- fmt.Sprintf("Handling job: %02d", i)
                return nil
            })
        }(i)
    }
    for i := 0; i < num; i++ {
        fmt.Println("Message:", <-res)
        time.Sleep(50 * time.Millisecond)
    }
}
```

### Using the Task Function

You can create a new message by implementing the `Bytes()` method on a custom struct to encode the message. Use the `WithFn()` 
method to define how the message gets handled from the queue.

```go
type job struct {
    Name    string
    Age     int
    Message string
}

func (j *job) Bytes() []byte {
    bytes, err := json.Marshal(j)
    if err != nil {
        panic(err)
    }
    return bytes
}

func usingMessageType() {
	var (
		num = 100
		wc  = make(chan int64)
		res = make(chan string, num)
	)
	queue := jobq.NewPool(5, jobq.WithFn(func(ctx context.Context, msg work.TaskMessage) error {
		var val job
		if err := json.Unmarshal(msg.Payload(), &val); err != nil {
			return err
		}
		res <- val.Name + " " + strconv.Itoa(val.Age) + ", " + val.Message
		return nil
	}))
	for i := 0; i < num; i++ {
		go func(i int) {
			err := queue.EnQueue(&job{
				Name:    "Worker",
				Age:     i,
				Message: fmt.Sprintf("with job: %02d", i),
			})
			if err != nil {
				log.Println(err)
			}
			wc <- queue.BusyWorkers()
		}(i)
	}
	for i := 0; i < num; i++ {
		fmt.Println("Message:", <-res, "worker count:", <-wc)
		time.Sleep(50 * time.Millisecond)
	}
}
```

## How it works (on a high level)

- Jobs are submitted to the `Queue`, either as a message or a function.
- A worker pool pulls tasks from an internal buffer.
- Each job runs in its own goroutine with retry and timeout semantics.
- Metrics are updated internally.
- Shutdown waits for any in-flight work to finish.

## Status
This is a learning project, as a result it won't have all the bells and whistles of a production Job-Queue implementation, and the 
API might change in the future as I learn more and refine this project. 
