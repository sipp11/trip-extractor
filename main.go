package main

import (
	"database/sql"
	"os"

	"github.com/labstack/echo"
)

func main() {
	e := echo.New()

	connStr := os.Getenv("TPEX_PSQL")
	db, _ := sql.Open("postgres", connStr)
	h := &Handler{db: db}

	e.Static("/static", "assets")
	e.GET("/", h.indexHandler)
	e.Logger.Fatal(e.Start(":9090"))
}
