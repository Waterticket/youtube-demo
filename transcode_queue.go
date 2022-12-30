package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/adjust/rmq/v5"
	"log"
	"math"
	"os"
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

type VideoInfo struct {
	Stream []struct {
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		FrameRate string `json:"r_frame_rate"`
	} `json:"streams"`

	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func (v *VideoInfo) getFrameRate() int {
	var frameRate int
	var sec int
	if _, err := fmt.Sscanf(v.Stream[0].FrameRate, "%d/%d", &frameRate, &sec); err != nil {
		log.Println(err)
	}

	return frameRate / sec
}

func (v *VideoInfo) getDuration() float64 {
	var duration float64
	if _, err := fmt.Sscanf(v.Format.Duration, "%f", &duration); err != nil {
		log.Println(err)
	}

	return duration
}

func (v *VideoInfo) getDisplaySize() (int, int) {
	return v.Stream[0].Width, v.Stream[0].Height
}

func (v *VideoInfo) String() string {
	return fmt.Sprintf("width: %d, height: %d, frameRate: %d, duration: %.2f", v.Stream[0].Width, v.Stream[0].Height, v.getFrameRate(), v.getDuration())
}

func (consumer *Consumer) Consume(delivery rmq.Delivery) {
	id := delivery.Payload()
	log.Printf("start transcode %s", id)
	db.QueryRow("UPDATE videos SET status = 'transcoding' WHERE id = ?", id)

	origin := fmt.Sprintf("files/pending/vid_%s", id)
	directory := fmt.Sprintf("files/videos/%s", id)
	target := directory + "/stream_%v.m3u8"
	tsFileName := directory + "/vid_%v_%03d.ts"
	masterFileName := "vid.m3u8"

	success := make(chan bool, 0)
	failed := make(chan bool, 0)
	go func() {
		select {
		case <-success:
			db.QueryRow("UPDATE videos SET file_location = ?, status = 'transcode_success' WHERE id = ?", directory+"/"+masterFileName, id)

			consumer.count++
			duration := time.Now().Sub(consumer.before)
			consumer.before = time.Now()
			log.Printf("%s consumed %d %s %d", consumer.name, consumer.count, id, duration)

			if err := delivery.Ack(); err != nil {
				log.Printf("failed to ack %s: %s", id, err)
			} else {
				log.Printf("acked %s", id)
			}

		case <-failed:
			db.QueryRow("UPDATE videos SET status = 'transcode_failed' WHERE id = ?", id)
			if err := delivery.Reject(); err != nil {
				log.Printf("failed to reject %s: %s", id, err)
			} else {
				log.Printf("rejected %s", id)
			}
		}
	}()

	if _, err := os.Stat(directory); os.IsNotExist(err) {
		if err = os.Mkdir(directory, 0777); err != nil {
			log.Printf("Failed to Generate Folder %s : %s", directory, err)
			failed <- true
			return
		}
	}

	log.Printf("directory %s created", directory)

	var outb, errb bytes.Buffer
	cmd := exec.Command("ffprobe", "-i", origin, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height,r_frame_rate", "-show_entries", "format=duration", "-print_format", "json")
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		log.Printf("get video info %s failed: %s", id, errb.String())
		failed <- true
		return
	}

	log.Printf("get video info %s success: %s", id, outb.String())

	var videoInfo VideoInfo
	if err := json.Unmarshal(outb.Bytes(), &videoInfo); err != nil {
		log.Printf("parse video info %s failed: %s", id, err)
		failed <- true
		return
	}

	log.Printf("video info %s: %s", id, videoInfo.String())
	width, height := videoInfo.getDisplaySize()
	db.QueryRow("UPDATE videos SET width = ?, height = ?, duration = ? WHERE id = ?", width, height, int(math.Round(videoInfo.getDuration())), id)
	outb.Reset()
	errb.Reset()

	var ffmpegArgs []string
	var filter string
	cnt := 0

	var frameBitrate map[int]string
	switch frame := videoInfo.getFrameRate(); {
	case frame <= 30:
		frameBitrate = map[int]string{
			360:  "750K",
			480:  "1M",
			720:  "3M",
			1080: "5M",
			1440: "16M",
			2160: "40M",
		}

	case frame <= 60:
		frameBitrate = map[int]string{
			360:  "1M",
			480:  "2M",
			720:  "4M",
			1080: "10M",
			1440: "24M",
			2160: "60M",
		}
	}

	log.Printf("frame bitrate %s: %v", id, frameBitrate)
	if height < 360 {
		log.Printf("doesn't support under 360p video. %s", id)
		failed <- true
		return
	}

	switch {
	case height >= 2160:
		cnt += 1
		filter += fmt.Sprintf("[v%d]scale=w=-2:h=2160[v%dout];", cnt, cnt)
		ffmpegArgs = append(ffmpegArgs, getFFmpegArgs(cnt, frameBitrate[2160], "128K")...)
		fallthrough

	case height >= 1440:
		cnt += 1
		filter += fmt.Sprintf("[v%d]scale=w=-2:h=1440[v%dout];", cnt, cnt)
		ffmpegArgs = append(ffmpegArgs, getFFmpegArgs(cnt, frameBitrate[1440], "128K")...)
		fallthrough

	case height >= 1080:
		cnt += 1
		filter += fmt.Sprintf("[v%d]scale=w=-2:h=1080[v%dout];", cnt, cnt)
		ffmpegArgs = append(ffmpegArgs, getFFmpegArgs(cnt, frameBitrate[1080], "128K")...)
		fallthrough

	case height >= 720:
		cnt += 1
		filter += fmt.Sprintf("[v%d]scale=w=-2:h=720[v%dout];", cnt, cnt)
		ffmpegArgs = append(ffmpegArgs, getFFmpegArgs(cnt, frameBitrate[720], "96K")...)
		fallthrough

	case height >= 480:
		cnt += 1
		filter += fmt.Sprintf("[v%d]scale=w=-2:h=480[v%dout];", cnt, cnt)
		ffmpegArgs = append(ffmpegArgs, getFFmpegArgs(cnt, frameBitrate[480], "96K")...)
		fallthrough

	case height >= 360:
		cnt += 1
		filter += fmt.Sprintf("[v%d]scale=w=-2:h=360[v%dout];", cnt, cnt)
		ffmpegArgs = append(ffmpegArgs, getFFmpegArgs(cnt, frameBitrate[360], "48K")...)
	}

	filter = filter[:len(filter)-1] // remove last semicolon

	var streamMap string
	preFilter := fmt.Sprintf("[0:v]split=%d", cnt)
	for i := 1; i <= cnt; i++ {
		streamMap += fmt.Sprintf("v:%d,a:%d ", i-1, i-1)
		preFilter += fmt.Sprintf("[v%d]", i)
	}
	filter = preFilter + ";" + filter
	streamMap = streamMap[:len(streamMap)-1]

	ffmpegArgs = append([]string{"-i", origin, "-filter_complex", filter}, ffmpegArgs...)
	ffmpegArgs = append(ffmpegArgs, "-f", "hls", "-hls_time", "2", "-hls_playlist_type", "vod", "-hls_flags", "independent_segments", "-hls_segment_type", "mpegts", "-hls_segment_filename", tsFileName, "-master_pl_name", masterFileName, "-var_stream_map", streamMap, target)

	cmd = exec.Command(config.Transcode.FFmpegPath, ffmpegArgs...)
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	log.Printf("command line: %s", cmd.String())

	if err := cmd.Run(); err != nil {
		log.Printf("transcode %s failed: %s", id, errb.String())
		failed <- true
		return
	}

	log.Printf("transcode %s success", id)
	success <- true
	return
}

func getFFmpegArgs(n int, videoBitrate string, audioBitrate string) []string {
	args := []string{
		"-map", fmt.Sprintf("[v%dout]", n), fmt.Sprintf("-c:v:%d", n-1), "libx264", "-x264-params", "nal-hrd=cbr:force-cfr=1",
		fmt.Sprintf("-b:v:%d", n-1), videoBitrate, fmt.Sprintf("-maxrate:v:%d", n-1), videoBitrate, fmt.Sprintf("-minrate:v:%d", n-1), videoBitrate, fmt.Sprintf("-bufsize:v:%d", n-1), videoBitrate,
		"-preset", "slow", "-g", "48", "-sc_threshold", "0", "-keyint_min", "48",

		"-map", "a:0", fmt.Sprintf("-c:a:%d", n-1), "aac", fmt.Sprintf("-b:a:%d", n-1), audioBitrate, "-ac", "2",
	}

	return args
}
