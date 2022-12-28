package main

import (
	"fmt"
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
)

func main() {
	var err error
	db, err = sql.Open("mysql", "root:password@tcp(127.0.0.1:3307)/ytdemo")
	defer db.Close()

	if err != nil {
		fmt.Println("MYSQL Connect Failed: ", err)
		os.Exit(0)
	}

	var version string
	db.QueryRow("SELECT VERSION()").Scan(&version)
	fmt.Println("Connected to:", version)

	e = echo.New()
	e.GET("/", index)

	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.Logger.Fatal(e.Start(":8080"))

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

forever:
	for {
		select {
		case <-sig:
			fmt.Println("signal received, stopping")
			break forever
		}
	}
}

func index(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
