package main

import (
	"database/sql"
	"fmt"
	"strings"

	pq "github.com/lib/pq"
)

type (
	// Stop is to store all info about stop we need
	// to know to create trip
	Stop struct {
		ID         string  `json:"stop_id" validate:"required"`
		Name       string  `json:"stop_name"`
		Lat        float64 `json:"stop_lat" validate:"required"`
		Lon        float64 `json:"stop_lon" validate:"required"`
		Sequence   int     `json:"sequence" validate:"gte=0"`
		IsTerminal bool    `json:"is_terminal"`
	}

	// Trace is to keep all GPS data
	Trace struct {
		BoxID     string  `json:"box_id" db:"box_id" validate:"required"`
		Timestamp string  `json:"timestamp" db:"timestamp" validate:"required"`
		Lat       float64 `json:"lat" db:"lat" validate:"required"`
		Lon       float64 `json:"lon" db:"lon" validate:"required"`
	}
)

// ItemCount is a shorthand for counting item in asking table
func (h *Handler) ItemCount(tbl string) (int, string) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tbl)
	var count int
	err := h.db.QueryRow(query).Scan(&count)
	if err, ok := err.(*pq.Error); ok {
		if err.Code.Name() == "undefined_table" {
			// it is not yet created
			if tbl == "stops" {
				err := h.createStopTable()
				if err != nil {
					fmt.Println(err)
				}
			} else if tbl == "traces" {
				err := h.createTraceTable()
				if err != nil {
					fmt.Println(err)
				}
			}
			return -10, "undefined_table"
		}
		return -1, "db error"
	}
	return count, "ok"
}

// queryStops return stops in order mannerly
func (h *Handler) queryStops(where string, order string) (rows *sql.Rows, err error) {
	if order != "ASC" {
		order = "DESC"
	}
	if len(where) > 0 && strings.Index(where, "WHERE") == -1 {
		where = fmt.Sprintf("WHERE %s", where)
	}
	fieldOrder := `stop_id,sequence,'stop_name',stop_lat,stop_lon,is_terminal`
	query := fmt.Sprintf(`SELECT %s FROM stops %s ORDER BY sequence %s`, fieldOrder, where, order)
	return h.db.Query(query)
}

// queryTraces return stops in order mannerly
func (h *Handler) queryTraces(where string, order string) (rows *sql.Rows, err error) {
	if order != "DESC" {
		order = "ASC"
	}
	if len(where) > 0 && strings.Index(where, "WHERE") == -1 {
		where = fmt.Sprintf("WHERE %s", where)
	}
	query := fmt.Sprintf(`SELECT * FROM traces %s ORDER BY box_id ASC, timestamp %s`, where, order)
	return h.db.Query(query)
}
