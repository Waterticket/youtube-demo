package main

import (
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Video struct {
	ID           int64  `json:"id"`
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
		res, err := db.Exec("INSERT INTO videos (title, description, file_location, status) VALUES (?, ?, ?, ?)", video.Title, video.Description, video.FileLocation, video.Status)
		if err != nil {
			return SendMessage(c, http.StatusInternalServerError, 500, err.Error(), nil)
		}

		video.ID, _ = res.LastInsertId()
		if err != nil {
			return SendMessage(c, http.StatusInternalServerError, 500, err.Error(), nil)
		}
	} else {
		db.QueryRow("UPDATE videos SET title = ?, description = ?, file_location = ?, status = ?, upload_date = ? WHERE id = ?", video.Title, video.Description, video.FileLocation, video.Status, video.UploadDate, video.ID)
	}

	return SendMessage(c, http.StatusOK, 200, "success", video)
}

func uploadVideo(c echo.Context) error {
	id := c.Param("id")

	// get range
	rangeHeader := c.Request().Header.Get("Content-Range")
	if rangeHeader == "" {
		return SendMessage(c, http.StatusBadRequest, 400, "Content-Range header is missing", nil)
	}

	// parse range
	rangeParts := strings.Split(rangeHeader, " ")
	if len(rangeParts) != 2 {
		return SendMessage(c, http.StatusBadRequest, 400, "Content-Range header is invalid", nil)
	}

	rangeParts = strings.Split(rangeParts[1], "/")
	if len(rangeParts) != 2 {
		return SendMessage(c, http.StatusBadRequest, 400, "Content-Range header is invalid", nil)
	}

	maxBytes, _ := strconv.Atoi(rangeParts[1])
	if maxBytes == 4*1024*1024*1024 {
		return SendMessage(c, http.StatusBadRequest, 400, "File size should be less than 4GB", nil)
	}

	rangeParts = strings.Split(rangeParts[0], "-")
	if len(rangeParts) != 2 {
		return SendMessage(c, http.StatusBadRequest, 400, "Content-Range header is invalid", nil)
	}

	startByte, _ := strconv.Atoi(rangeParts[0])
	endByte, _ := strconv.Atoi(rangeParts[1])

	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	targetLocation := "files/pending/vid_" + id
	if err = writeFile(id, startByte, endByte, maxBytes, src, targetLocation); err != nil {
		log.Println(err)
		return SendMessage(c, http.StatusInternalServerError, 500, err.Error(), nil)
	}

	if endByte == maxBytes {
		db.QueryRow("UPDATE videos SET status = 'transcode_pending' WHERE id = ?", id)
		transcodeQueuePush(id)
	}

	return SendMessage(c, http.StatusOK, 200, "success", nil)
}
