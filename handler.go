package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

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
	stopTimeCnt, _ := h.ItemCount("stop_times")

	stops := h.getStops("", "ASC")

	directions := h.getDistinctDirection()
	// summary := make([][]string, len(stops))
	summary := make(map[string][][]string)
	for _, direction := range directions {
		summary[direction] = make([][]string, len(stops))
		for ind, stop := range stops {
			where := fmt.Sprintf("direction='%s' AND stop_id='%s'", direction, stop.ID)
			rows, err := h.queryStopTime(where, "ASC")
			if err != nil {
				log.Fatal("getStopTime err: ", err)
			}
			summary[direction][ind] = append(summary[direction][ind], stop.ID)
			for rows.Next() {
				var stopTime StopTime
				rows.Scan(&stopTime.BoxID, &stopTime.StopID, &stopTime.Direction,
					&stopTime.Sequence, &stopTime.Arrival, &stopTime.StopDuration)
				tmsp, _ := time.Parse(time.RFC3339, stopTime.Arrival)
				tmspStr := tmsp.Format("15:04:05")
				oneST := fmt.Sprintf("%s (%d s)", tmspStr, stopTime.StopDuration)
				summary[direction][ind] = append(summary[direction][ind], oneST)
			}
		}
	}

	out, err := indexTmpl.Execute(pongo2.Context{
		"stops":      stopCnt,
		"traces":     traceCnt,
		"stop_times": stopTimeCnt,
		"stop_items": stops,
		"summary":    summary,
	})
	if err != nil {
		return err
	}
	return c.HTML(http.StatusOK, out)
}

func (h *Handler) resetData(c echo.Context) error {
	if err := h.truncateTables(); err != nil {
		return c.HTML(http.StatusBadRequest, fmt.Sprintf("Something went wrong: %v", err))
	}
	return c.HTML(http.StatusOK, "Successfuly reset")
}
