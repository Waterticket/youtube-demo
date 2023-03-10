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
	configInit()

	var err error
	// mysql
	db, err = sql.Open("mysql", config.Mysql.Username+":"+config.Mysql.Password+"@tcp("+config.Mysql.Addr+")/"+config.Mysql.Database+"?charset=utf8mb4&parseTime=True&loc=Local")
	defer db.Close()

	if err != nil {
		panic(err)
	}

	var version string
	db.QueryRow("SELECT VERSION()").Scan(&version)
	fmt.Println("Connected to:", version)

	// rmq
	errChan := make(chan error, 1)
	connection, err = rmq.OpenConnection("rmq", "tcp", config.Redis.Addr, 1, errChan)
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

	e.GET("/videos", videoList)
	e.GET("/video/:id", videoView)

	e.Logger.Fatal(e.Start(config.Server.Addr))

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

type Message struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func SendMessage(c echo.Context, httpStatus int, status int, message string, data interface{}) error {
	return c.JSON(httpStatus, Message{
		Status:  status,
		Message: message,
		Data:    data,
	})
}
