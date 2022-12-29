package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func videoList(c echo.Context) error {
	rows, err := db.Query("SELECT * FROM videos")
	if err != nil {
		return SendMessage(c, http.StatusInternalServerError, 500, err.Error(), nil)
	}
	defer rows.Close()

	videos := make([]Video, 0)

	for rows.Next() {
		var video Video

		err = rows.Scan(&video.ID, &video.Title, &video.Description, &video.FileLocation, &video.Status, &video.UploadDate)
		if err != nil {
			return SendMessage(c, http.StatusInternalServerError, 500, err.Error(), nil)
		}

		videos = append(videos, video)
	}

	return SendMessage(c, http.StatusOK, 200, "success", videos)
}

func videoView(c echo.Context) error {
	id := c.Param("id")

	var video Video
	err := db.QueryRow("SELECT * FROM videos WHERE id = ?", id).Scan(&video.ID, &video.Title, &video.Description, &video.FileLocation, &video.Status, &video.UploadDate)
	if err != nil {
		return SendMessage(c, http.StatusInternalServerError, 500, err.Error(), nil)
	}

	return SendMessage(c, http.StatusOK, 200, "success", video)
}
