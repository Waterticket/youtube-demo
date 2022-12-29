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
		message.Status = 500
		message.Message = err.Error()
		messageJson, _ := json.Marshal(message)
		return c.String(http.StatusInternalServerError, string(messageJson))
	}
	defer rows.Close()

	videos := make([]Video, 0)

	for rows.Next() {
		var video Video

		err = rows.Scan(&video.ID, &video.Title, &video.Description, &video.FileLocation, &video.Status, &video.UploadDate)
		if err != nil {
			message.Status = 500
			message.Message = err.Error()
			messageJson, _ := json.Marshal(message)
			return c.String(http.StatusInternalServerError, string(messageJson))
		}

		videos = append(videos, video)
	}

	message.Status = 200
	message.Message = "success"
	message.Data = videos
	messageJson, _ := json.Marshal(message)
	return c.String(http.StatusOK, string(messageJson))
}

func videoView(c echo.Context) error {
	message := new(Message)
	id := c.Param("id")

	var video Video
	err := db.QueryRow("SELECT * FROM videos WHERE id = ?", id).Scan(&video.ID, &video.Title, &video.Description, &video.FileLocation, &video.Status, &video.UploadDate)
	if err != nil {
		message.Status = 500
		message.Message = err.Error()
		messageJson, _ := json.Marshal(message)
		return c.String(http.StatusInternalServerError, string(messageJson))
	}

	message.Status = 200
	message.Message = "success"
	message.Data = video
	messageJson, _ := json.Marshal(message)
	return c.String(http.StatusOK, string(messageJson))
}
