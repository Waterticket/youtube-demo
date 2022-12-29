package main

import (
	"fmt"
	"github.com/adjust/rmq/v5"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"net/http"
	"os"
	"os/signal"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB
	e  *echo.Echo

	connection rmq.Connection
)

func main() {
	var err error
	// mysql
	db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3307)/ytdemo")
	defer db.Close()

	if err != nil {
		fmt.Println("MYSQL Connect Failed: ", err)
		os.Exit(0)
	}

	var version string
	db.QueryRow("SELECT VERSION()").Scan(&version)
	fmt.Println("Connected to:", version)

	// rmq
	errChan := make(chan error, 1)
	connection, err = rmq.OpenConnection("rmq", "tcp", "localhost:6379", 1, errChan)
	if err != nil {
		panic(err)
	}

	// transcode queue
	if err = transcodeQueueInit(connection); err != nil {
		panic(err)
	}

	// echo
	e = echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))
	e.GET("/", index)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.GET("/upload", uploadView)
	e.POST("/upload/metadata", uploadMetadata)
	e.POST("/upload/video/:id", uploadVideo)
	e.PUT("/upload/video/:id", uploadVideo)

	e.Logger.Fatal(e.Start(":8080"))

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			fmt.Println("signal received, stopping")
			break forever

		case err := <-errChan:
			fmt.Println("error received, stopping", err)
			break forever
		}
	}
}

func index(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
