package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/flosch/pongo2"
	"github.com/labstack/echo"
	"github.com/lib/pq"
)

type (
	// Handler to handle all route
	Handler struct {
		db *sql.DB
	}
	// Stop is to store all info about stop we need
	// to know to create trip
	Stop struct {
		StopID     string  `json:"stop_id" validate:"required"`
		StopName   string  `json:"stop_name"`
		StopLat    float32 `json:"stop_lat" validate:"required"`
		StopLon    float32 `json:"stop_lon" validate:"required"`
		IsTerminal bool    `json:"is_terminal"`
	}

	// Trace is to keep all GPS data
	Trace struct {
		BoxID     string  `json:"box_id" validate:"required"`
		Timestamp string  `json:"timestamp" validate:"required"`
		Lat       float32 `json:"lat" validate:"required"`
		Lon       float32 `json:"lon" validate:"required"`
	}

	// Result for all input handlers
	Result struct {
		Success int    `json:"success"`
		Failed  int    `json:"failed"`
		Message string `json:"message"`
	}
)

// StopInputHandler to accept stop via REST interface
func (h *Handler) StopInputHandler(c echo.Context) (err error) {
	stops := new([]Stop)
	if err := c.Bind(stops); err != nil {
		return err
	}
	result := Result{Success: 0, Failed: 0}
	var buffer bytes.Buffer
	for _, ele := range *stops {
		if err := c.Validate(ele); err != nil {
			result.Failed++
			if buffer.Len() > 0 {
				buffer.WriteString(",")
			}
			buffer.WriteString(err.Error())
		} else {
			result.Success++
			// insert
			_, err := h.db.Exec(fmt.Sprintf("INSERT INTO stops (stop_id, stop_name, stop_lat, stop_lon, is_terminal) VALUES ('%s', '%s', %f, %f, %v)", ele.StopID, ele.StopName, ele.StopLat, ele.StopLon, ele.IsTerminal))
			if err, ok := err.(*pq.Error); ok {
				// Here err is of type *pq.Error, you may inspect all its fields, e.g.:
				fmt.Println("pq error:", err.Code.Name())
			}
		}
	}
	if buffer.Len() > 0 {
		result.Message = buffer.String()
	}
	return c.JSON(http.StatusOK, result)
}

// TraceInputHandler to accept trace via REST interface
func (h *Handler) TraceInputHandler(c echo.Context) error {
	traces := new([]Trace)
	if err := c.Bind(traces); err != nil {
		return err
	}
	result := Result{Success: 0, Failed: 0}
	var buffer bytes.Buffer
	for _, ele := range *traces {
		if err := c.Validate(ele); err != nil {
			result.Failed++
			if buffer.Len() > 0 {
				buffer.WriteString(",")
			}
			buffer.WriteString(err.Error())
		} else {
			result.Success++
			// insert
			_, err := h.db.Exec(fmt.Sprintf("INSERT INTO traces (box_id, timestamp, lat, lon) VALUES ('%s', '%s', %f, %f)", ele.BoxID, ele.Timestamp, ele.Lat, ele.Lon))
			if err, ok := err.(*pq.Error); ok {
				// Here err is of type *pq.Error, you may inspect all its fields, e.g.:
				fmt.Println("pq error:", err.Error())
			}
		}
	}
	if buffer.Len() > 0 {
		result.Message = buffer.String()
	}
	return c.JSON(http.StatusOK, result)
}

// IndexHandler is the front page to check everything
func (h *Handler) IndexHandler(c echo.Context) error {
	var indexTmpl = pongo2.Must(pongo2.FromFile("html/index.html"))
	stopCnt := 0
	traceCnt := 0
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM stops").Scan(&count)
	if err, ok := err.(*pq.Error); ok {
		if err.Code.Name() == "undefined_table" {
			// it is not yet created
			if err := h.createStopTable(); err != nil {
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
			if err := h.createTraceTable(); err != nil {
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

func (h *Handler) resetData(c echo.Context) error {
	if err := h.truncateStopAndTrace(); err != nil {
		return c.HTML(http.StatusBadRequest, fmt.Sprintf("Something went wrong: %v", err))
	}
	return c.HTML(http.StatusOK, "Successfuly reset")
}

func (h *Handler) truncateStopAndTrace() error {
	sq := `TRUNCATE TABLE stops;
		TRUNCATE TABLE traces;`
	_, err := h.db.Exec(sq)
	return err
}

func (h *Handler) createStopTable() error {
	sq := `CREATE TABLE stops (
		stop_id char(150),
		stop_name char(250),
		stop_lat numeric,
		stop_lon numeric,
		is_terminal bool)
		`
	_, err := h.db.Exec(sq)
	return err
}

func (h *Handler) createTraceTable() error {
	cq := `CREATE TABLE traces (
		box_id char(150),
		timestamp timestamp,
		lat	numeric,
		lon numeric)
		`
	_, err := h.db.Exec(cq)
	return err
}
