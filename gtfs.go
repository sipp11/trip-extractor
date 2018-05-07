package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

func makeRange(size int, ascending bool) []int {
	a := make([]int, size)
	for i := range a {
		if ascending == true {
			a[i] = i
		} else {
			a[i] = size - i - 1
		}
	}
	return a
}

// GTFSExporter - export what we can
// * stop_times.txt
// * stops.txt
// * trips.txt (route_id, service_id, trip_id) only trip_id; the rest is dummy
func (h *Handler) GTFSExporter() {

	stops := h.getStops("", "ASC")
	h.StopExporter(stops)
	trips := h.StopTimesExporter(stops)
	h.TripExporter(trips)
}

// StopExporter will give stops.txt
func (h *Handler) StopExporter(stops []Stop) {
	file, err := os.Create("stops.txt")
	checkError("cannot create file", err)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// header
	headerRow := []string{
		"stop_id", "stop_code", "stop_name", "stop_desc", "stop_lat",
		"stop_lon", "zone_id", "stop_url", "location_type", "parent_station",
		"direction", "position"}
	err = writer.Write(headerRow)
	checkError("Cannot write to file [se0] ", err)

	for _, stop := range stops {
		row := make([]string, len(headerRow))
		row[0] = strings.TrimSpace(stop.ID)
		row[1] = strings.TrimSpace(stop.ID)
		row[2] = strings.TrimSpace(stop.Name)
		row[4] = fmt.Sprintf("%f", stop.Lat)
		row[5] = fmt.Sprintf("%f", stop.Lon)
		err = writer.Write(row)
		checkError("Cannot write to file [se1] ", err)
	}
}

// TripExporter will give trips.txt
func (h *Handler) TripExporter(trips []string) {
	// route_id,service_id,trip_id,direction_id,block_id,shape_id,trip_type
	file, err := os.Create("trips.txt")
	checkError("cannot create file", err)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// header
	headerRow := []string{
		"route_id", "service_id", "trip_id", "direction_id",
		"block_id", "shape_id", "trip_type"}
	err = writer.Write(headerRow)
	checkError("Cannot write to file [te0] ", err)

	for _, trip := range trips {
		row := make([]string, len(headerRow))
		row[2] = trip
		err = writer.Write(row)
		checkError("Cannot write to file [te1] ", err)
	}
}

// StopTimesExporter - to export all data generated for gtfs feed
// return generated trips (tripID)
func (h *Handler) StopTimesExporter(stops []Stop) []string {
	// file columns
	// trip_id,arrival_time,departure_time,stop_id,stop_sequence,
	// stop_headsign,pickup_type,drop_off_type,shape_dist_traveled,
	// timepoint,continuous_drop_off,continuous_pickup
	fmt.Printf("exporting: stop_times\n")
	terminals := h.getStops("is_terminal=TRUE", "ASC")
	trips := h.FindTripStartEnd(terminals)
	var tripIDs []string

	file, err := os.Create("stop_times.txt")
	checkError("cannot create file", err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// header
	headerRow := []string{
		"trip_id", "arrival_time", "departure_time", "stop_id", "stop_sequence",
		"stop_headsign", "pickup_type", "drop_off_type", "shape_dist_traveled",
		"timepoint", "continuous_drop_off", "continuous_pickup"}
	err = writer.Write(headerRow)
	checkError("Cannot write to file", err)

	tripTotal := 0
	for _, ele := range trips {
		direction := fmt.Sprintf("%s..%s", strings.TrimSpace(ele.BeginAt.ID), strings.TrimSpace(ele.EndAt.ID))
		tripID := fmt.Sprintf("%s-%d", strings.Replace(direction, "..", "", -1), tripTotal)
		tripIDs = append(tripIDs, tripID)
		tripTotal++
		if len(ele.Comment) > 0 {
			continue
		}
		stopTimeTable := h.FindTripTimetable(ele, stops, direction)
		begID := fmt.Sprintf("%s..", strings.TrimSpace(stopTimeTable[0].StopID))
		isFwdDirection := strings.Index(direction, begID)
		var stOrder []int
		if isFwdDirection != 0 {
			stOrder = makeRange(len(stopTimeTable), false)
		} else {
			stOrder = makeRange(len(stopTimeTable), true)
		}
		for ind, targetInd := range stOrder {
			stEle := stopTimeTable[targetInd]
			order := ind + 1
			t1, _ := time.Parse(time.RFC3339, stEle.Arrival)
			t2, _ := time.Parse(time.RFC3339, stEle.Departure)

			arrivalStr := t1.Format("15:04:05")
			departureStr := t2.Format("15:04:05")

			one := make([]string, len(headerRow))
			one[0] = fmt.Sprintf("%+v", tripID)
			one[1] = arrivalStr
			one[2] = departureStr
			one[3] = strings.TrimSpace(stEle.StopID)
			one[4] = fmt.Sprintf("%+v", order)
			err = writer.Write(one)
			checkError("Cannot write to file", err)

			h.insertStopTime(stEle)
		}
	}
	return tripIDs
}
