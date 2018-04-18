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
		db              *sql.DB
		port            string
		rangeWithinStop float64
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
			_, err := h.db.Exec(fmt.Sprintf("INSERT INTO stops (stop_id, stop_name, stop_lat, stop_lon, is_terminal, sequence) VALUES ('%s', '%s', %f, %f, %v, %d)", ele.ID, ele.Name, ele.Lat, ele.Lon, ele.IsTerminal, ele.Sequence))
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
	stopCnt, _ := h.ItemCount("stops")
	traceCnt, _ := h.ItemCount("traces")
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
		timestamp timestamp,
		lat	numeric,
		lon numeric,
		UNIQUE(box_id, timestamp)
		)`
	_, err := h.db.Exec(cq)
	return err
}
