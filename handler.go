package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/flosch/pongo2"
	"github.com/labstack/echo"
	"github.com/lib/pq"
)

// Handler to handle all route
type Handler struct {
	db *sql.DB
}

func (h *Handler) indexHandler(c echo.Context) error {
	var indexTmpl = pongo2.Must(pongo2.FromFile("html/index.html"))

	stopCnt := 0
	traceCnt := 0
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM stops").Scan(&count)
	if err, ok := err.(*pq.Error); ok {
		if err.Code.Name() == "undefined_table" {
			// it is not yet created
			createStopTable := `CREATE TABLE stops (
				stop_id char(150),
				stop_name char(250),
				stop_lat numeric,
				stop_lon numeric,
				is_termimal bool)
				`
			_, err := h.db.Exec(createStopTable)
			if err != nil {
				fmt.Println(err)
			}
			stopCnt = -10
		} else {
			stopCnt = -1
		}
	} else {
		stopCnt = count
	}
	err = h.db.QueryRow("SELECT COUNT(*) FROM traces").Scan(&count)
	if err, ok := err.(*pq.Error); ok {
		if err.Code.Name() == "undefined_table" {
			// it is not yet created
			createTraceTable := `CREATE TABLE traces (
				box_id char(150),
				timestamp timestamp,
				lat	numeric,
				lon numeric)
				`
			_, err := h.db.Exec(createTraceTable)
			if err != nil {
				fmt.Println(err)
			}
			traceCnt = -10
		} else {
			traceCnt = -1
		}
	} else {
		traceCnt = count
	}
	out, err := indexTmpl.Execute(pongo2.Context{"stops": stopCnt, "traces": traceCnt})
	if err != nil {
		return err
	}
	return c.HTML(http.StatusOK, out)
}
