package main

import (
	"bytes"
	"fmt"
	"github.com/adjust/rmq/v5"
	"log"
	"os/exec"
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

		if err = transcodeQueue.StartConsuming(config.Transcode.TranscodeCount*10, 100*time.Millisecond); err != nil {
			return err
		}

		var i int64
		for i = 0; i < config.Transcode.TranscodeCount; i++ {
			name := fmt.Sprintf("consumer %d", i)
			if _, err := transcodeQueue.AddConsumer(name, NewConsumer(i)); err != nil {
				panic(err)
			}
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

type Consumer struct {
	name   string
	count  int64
	before time.Time
}

func NewConsumer(tag int64) *Consumer {
	return &Consumer{
		name:   fmt.Sprintf("consumer%d", tag),
		count:  0,
		before: time.Now(),
	}
}

func (consumer *Consumer) Consume(delivery rmq.Delivery) {
	id := delivery.Payload()
	log.Printf("start transcode %s", id)
	db.QueryRow("UPDATE videos SET status = 'transcoding' WHERE id = ?", id)

	success := true
	origin := fmt.Sprintf("files/pending/vid_%s", id)
	target := fmt.Sprintf("files/videos/vid_%s.mp4", id)

	cmd := exec.Command(config.Transcode.FFmpegPath, "-i", origin, "-c:v", "libx264", "-crf", "18", "-c:a", "aac", "-b:a", "128k", "-ac", "2", "-f", "mp4", target)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		log.Printf("transcode %s failed: %s", id, errb.String())
		success = false
	}

	if success {
		db.QueryRow("UPDATE videos SET file_location = ?, status = 'transcode_success' WHERE id = ?", target, id)

		consumer.count++
		duration := time.Now().Sub(consumer.before)
		consumer.before = time.Now()
		log.Printf("%s consumed %d %s %d", consumer.name, consumer.count, id, duration)

		if err := delivery.Ack(); err != nil {
			log.Printf("failed to ack %s: %s", id, err)
		} else {
			log.Printf("acked %s", id)
		}
	} else { // reject one per batch
		db.QueryRow("UPDATE videos SET status = 'transcode_failed' WHERE id = ?", id)
		if err := delivery.Reject(); err != nil {
			log.Printf("failed to reject %s: %s", id, err)
		} else {
			log.Printf("rejected %s", id)
		}
	}
}
