package main

import (
	"context"
	"encoding/json"
	"fmt"
	"jobq"
	"jobq/work"
	"log"
	"strconv"
	"time"
)

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

func main() {
	messageType()
}

func messageType() {
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

func taskFunc() {
	var (
		num = 100
		res = make(chan string, num)
	)
	queue := jobq.NewPool(5)
	defer queue.Release()

	for i := 0; i < num; i++ {
		go func(i int) {
			_ = queue.EnQueueTask(func(ctx context.Context) error {
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
