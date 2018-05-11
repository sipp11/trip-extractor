package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/flosch/pongo2"
	"github.com/labstack/echo"
	"github.com/lib/pq"
	validator "gopkg.in/go-playground/validator.v9"
)

// CustomValidator no idea what this is
type CustomValidator struct {
	validator *validator.Validate
}

// Validate is to validate input data
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func (h *Handler) serveWebInterface() {

	layout := "2006-01-02T15:04:05-0700"
	t, _ := time.Parse(layout, "2014-11-17T23:02:03+0000")
	t2, _ := time.Parse(layout, "2014-11-18T06:02:03+0700")
	fmt.Println("time: ", t, t.Unix())
	fmt.Println("time: ", t2, t2.Unix())
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	e.Static("/static", "assets")
	e.GET("/", h.IndexHandler)
	e.POST("/input/reset", h.resetData)
	e.POST("/input/stop", h.StopInputHandler)
	e.POST("/input/trace", h.TraceInputHandler)
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", h.port)))
}

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
	stopAndRouteCnt, _ := h.ItemCount("stop_and_route")
	traceCnt, _ := h.ItemCount("traces")
	stopTimeCnt, _ := h.ItemCount("stop_times")
	directions := h.getDistinctDirection()
	summary := make(map[string][][]string)

	for _, direction := range directions {
		stops := h.getStops(direction, "ASC", false)
		summary[direction] = make([][]string, len(stops))
		for ind, stop := range stops {
			where := fmt.Sprintf("direction='%s' AND stop_id='%s'", direction, stop.ID)
			fieldOrder := `box_id,stop_id,direction,sequence,to_char(arrival, 'HH24:MI') as hhmm,stop_duration`
			query := fmt.Sprintf(`SELECT %s from stop_times WHERE %s order by hhmm ASC;`, fieldOrder, where)
			rows, err := h.db.Query(query)
			if err != nil {
				log.Fatal("getStopTime err: ", err)
			}
			summary[direction][ind] = append(summary[direction][ind], stop.ID)
			for rows.Next() {
				var stopTime StopTime
				rows.Scan(&stopTime.BoxID, &stopTime.StopID, &stopTime.Direction,
					&stopTime.Sequence, &stopTime.Arrival, &stopTime.StopDuration)
				oneST := fmt.Sprintf("%s (%d s)", stopTime.Arrival, stopTime.StopDuration)
				summary[direction][ind] = append(summary[direction][ind], oneST)
			}
		}
	}

	out, err := indexTmpl.Execute(pongo2.Context{
		"stops":          stopCnt,
		"stop_and_route": stopAndRouteCnt,
		"traces":         traceCnt,
		"stop_times":     stopTimeCnt,
		"summary":        summary,
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
