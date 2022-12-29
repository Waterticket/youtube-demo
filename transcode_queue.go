package main

import (
	"github.com/adjust/rmq/v5"
	"time"
)

var (
	transcodeQueue rmq.Queue
)

func transcodeQueueInit(connection rmq.Connection) error {
	if transcodeQueue == nil {
		var err error
		if transcodeQueue, err = connection.OpenQueue("transcode"); err != nil {
			return err
		}

		if err = transcodeQueue.StartConsuming(10, 100*time.Millisecond); err != nil {
			return err
		}
	}

	return nil
}

func transcodeQueuePush(message string) error {
	if err := transcodeQueue.Publish(message); err != nil {
		return err
	}

	return nil
}

func transcodeTask() {

}
