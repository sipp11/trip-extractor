package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

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

	// StopTime is a arrival time for each stopm including stop duration
	StopTime struct {
		BoxID        string `json:"box_id" db:"box_id" validate:"required"`
		StopID       string `json:"stop_id" validate:"required"`
		Direction    string `json:"direction" validate:"required"`
		Sequence     int    `json:"sequence" validate:"gte=0"`
		Arrival      string `json:"arrival" db:"arrival" validate:"required"`
		StopDuration int    `json:"stop_duration" db:"stop_duration" validate:"required"`
	}
)

func (h *Handler) truncateTables() error {
	sq := `TRUNCATE TABLE stops;
		TRUNCATE TABLE traces;`
	_, err := h.db.Exec(sq)
	if err != nil {
		return err
	}
	err = h.truncateStopTimeTable()
	return err
}

func (h *Handler) truncateStopTimeTable() error {
	sq := `TRUNCATE TABLE stop_times;`
	_, err := h.db.Exec(sq)
	return err
}

func (h *Handler) createStopTable() error {
	sq := `CREATE TABLE stops (
		stop_id char(150),
		sequence int,
		stop_name char(250),
		stop_lat numeric,
		stop_lon numeric,
		is_terminal bool,
		UNIQUE(stop_id)
		)`
	_, err := h.db.Exec(sq)
	return err
}

func (h *Handler) createTraceTable() error {
	cq := `CREATE TABLE traces (
		box_id char(150),
		timestamp timestamptz,
		lat	numeric,
		lon numeric,
		UNIQUE(box_id, timestamp)
		)`
	_, err := h.db.Exec(cq)
	return err
}

func (h *Handler) createStopTimeTable() error {
	cq := `CREATE TABLE stop_times (
		box_id char(150),
		stop_id char(150),
		direction char(30),
		sequence int,
		arrival timestamptz,
		stop_duration int,
		UNIQUE(box_id, stop_id, arrival)
		)`
	_, err := h.db.Exec(cq)
	return err
}

func (h *Handler) insertStopTime(st StopTimeRaw) error {
	insertQuery := `INSERT INTO stop_times
	(box_id, stop_id, direction, sequence, arrival, stop_duration)
	values ($1,$2,$3,$4,$5,$6);`

	stmt, err := h.db.Prepare(insertQuery)
	if err != nil {
		return err
	}
	t2, _ := time.Parse(time.RFC3339, st.Departure)
	t1, _ := time.Parse(time.RFC3339, st.Arrival)
	duration := t2.Sub(t1)
	_, err = stmt.Exec(st.BoxID, st.StopID, st.Direction, st.Sequence, st.Arrival, int(duration.Seconds()))
	return err

}

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
			} else if tbl == "stop_times" {
				err := h.createStopTimeTable()
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

func (h *Handler) queryStopTime(where string, order string) (rows *sql.Rows, err error) {
	if order != "DESC" {
		order = "ASC"
	}
	if len(where) > 0 && strings.Index(where, "WHERE") == -1 {
		where = fmt.Sprintf("WHERE %s", where)
	}
	fieldOrder := `box_id,stop_id,direction,sequence,arrival,stop_duration`
	query := fmt.Sprintf(`SELECT %s FROM stop_times %s ORDER BY direction ASC, arrival %s`, fieldOrder, where, order)
	return h.db.Query(query)
}

func (h *Handler) getDistinctDirection() []string {
	var directions []string
	query := fmt.Sprintf("SELECT DISTINCT(direction) FROM stop_times ORDER BY direction ASC")
	rows, err := h.db.Query(query)
	if err != nil {
		log.Fatal("get distinct direction error", err)
	}
	for rows.Next() {
		var direction string
		rows.Scan(&direction)
		directions = append(directions, direction)
	}
	return directions
}

func (h *Handler) getStopTimes(where string, order string) []StopTime {
	var result []StopTime
	return result
}
