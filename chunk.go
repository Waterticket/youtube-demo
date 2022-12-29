package main

import (
	"io"
	"os"
	"strconv"
)

var (
	fileHandler []*os.File
)

func writeFile(id string, startByte int, endByte int, maxBytes int, src io.Reader, targetLocation string) error {
	idseq, _ := strconv.Atoi(id)
	if err := checkDirectory(); err != nil {
		return err
	}

	filePath := "files/temp/" + id
	var file *os.File
	var err error

	if startByte == 0 {
		file, err = os.Create(filePath)
		if err != nil {
			return err
		}

		if err = preAllocateFile(file, int64(maxBytes)); err != nil {
			return err
		}

		fileHandler[idseq] = file
	}

	file = fileHandler[idseq]
	if _, err = file.Seek(int64(startByte), 0); err != nil {
		return err
	}

	if _, err = io.Copy(file, src); err != nil {
		return err
	}

	if endByte == maxBytes {
		// last chunk

		dest, err := os.OpenFile(targetLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer dest.Close()

		if _, err = file.Seek(0, 0); err != nil {
			return err
		}

		if _, err = io.Copy(dest, file); err != nil {
			return err
		}

		if err = file.Close(); err != nil {
			return err
		}

		if err = os.Remove(filePath); err != nil {
			return err
		}

		fileHandler[idseq] = nil
	}

	return nil
}

func checkDirectory() error {
	if _, err := os.Stat("files/temp"); os.IsNotExist(err) {
		err := os.Mkdir("files/temp", 0777)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("files/pending"); os.IsNotExist(err) {
		err := os.Mkdir("files/pending", 0777)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat("files/videos"); os.IsNotExist(err) {
		err := os.Mkdir("files/videos", 0777)
		if err != nil {
			return err
		}
	}

	return nil
}

func preAllocateFile(file *os.File, size int64) error {
	if err := file.Truncate(size); err != nil {
		return err
	}

	return nil
}
