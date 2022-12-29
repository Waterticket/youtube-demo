package main

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"net/http"
)

func videoList(c echo.Context) error {
	message := new(Message)
	rows, err := db.Query("SELECT * FROM videos")
	if err != nil {
		message.status = 500
		message.message = err.Error()
		messageJson, _ := json.Marshal(message)
		return c.String(http.StatusInternalServerError, string(messageJson))
	}

	videos := make([]Video, 0)

	for rows.Next() {
		var video Video

		err = rows.Scan(&video.ID, &video.Title, &video.Description, &video.FileLocation, &video.Status, &video.UploadDate)
		if err != nil {
			message.status = 500
			message.message = err.Error()
			messageJson, _ := json.Marshal(message)
			return c.String(http.StatusInternalServerError, string(messageJson))
		}

		videos = append(videos, video)
	}

	message.status = 200
	message.message = "success"
	message.data = videos
	messageJson, _ := json.Marshal(message)
	return c.String(http.StatusOK, string(messageJson))
}

func videoView(c echo.Context) error {
	message := new(Message)
	id := c.Param("id")

	var video Video
	err := db.QueryRow("SELECT * FROM videos WHERE id = ?", id).Scan(&video.ID, &video.Title, &video.Description, &video.FileLocation, &video.Status, &video.UploadDate)
	if err != nil {
		message.status = 500
		message.message = err.Error()
		messageJson, _ := json.Marshal(message)
		return c.String(http.StatusInternalServerError, string(messageJson))
	}

	message.status = 200
	message.message = "success"
	message.data = video
	messageJson, _ := json.Marshal(message)
	return c.String(http.StatusOK, string(messageJson))
}
