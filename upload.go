package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type Video struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	FileLocation string `json:"file_location"`
	Status       string `json:"status"`
	UploadDate   string `json:"upload_date"`
}

func uploadView(c echo.Context) error {
	return c.String(http.StatusOK, "Upload page")
}

func uploadMetadata(c echo.Context) error {
	video := new(Video)
	if err := c.Bind(video); err != nil {
		return err
	}

	if video.ID == 0 {
		db.QueryRow("INSERT INTO videos (title, description, file_location, status, upload_date) VALUES (?, ?, ?, ?, ?)", video.Title, video.Description, video.FileLocation, video.Status, video.UploadDate)
		db.QueryRow("SELECT LAST_INSERT_ID()").Scan(&video.ID)
	} else {
		db.QueryRow("UPDATE videos SET title = ?, description = ?, file_location = ?, status = ?, upload_date = ? WHERE id = ?", video.Title, video.Description, video.FileLocation, video.Status, video.UploadDate, video.ID)
	}

	return c.JSON(http.StatusOK, video)
}

func uploadVideo(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	return c.String(http.StatusOK, strconv.Itoa(id))
}
