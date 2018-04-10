package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/labstack/echo"
	validator "gopkg.in/go-playground/validator.v9"
	ini "gopkg.in/ini.v1"
)

// CustomValidator no idea what this is
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func main() {
	layout := "2006-01-02T15:04:05-0700"
	t, _ := time.Parse(layout, "2014-11-17T23:02:03+0000")
	t2, _ := time.Parse(layout, "2014-11-18T06:02:03+0700")
	fmt.Println("time: ", t, t.Unix())
	fmt.Println("time: ", t2, t2.Unix())
	cfg, err := ini.Load("my.ini")
	if err != nil {
		fmt.Printf("Fail to read my.ini: %v", err)
		os.Exit(1)
	}
	port := cfg.Section("app").Key("port").String()
	dbConn := cfg.Section("db").Key("path").String()

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		fmt.Printf("Fail to connect to db server: %v", err)
		os.Exit(1)
	}
	h := &Handler{db: db}

	e.Static("/static", "assets")
	e.GET("/", h.IndexHandler)
	e.POST("/input/reset", h.resetData)
	e.POST("/input/stop", h.StopInputHandler)
	e.POST("/input/trace", h.TraceInputHandler)
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", port)))
}
