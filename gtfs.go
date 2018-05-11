package main

import (
	"encoding/csv"
	"fmt"
	"os"
	s "strings"
	"time"
)

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
// * routes.txt - now we have route
func (h *Handler) GTFSExporter(route string, routeRev string) {

	stops := h.getStops("", "ASC", false)
	h.StopExporter(stops)
	h.RouteExporter()
	trips := h.StopTimesExporter(route, routeRev)
	h.TripExporter(trips)
	// TODO: add route to trip somehow
}

// StopExporter will give stops.txt
func (h *Handler) StopExporter(stops []Stop) {
	file, err := os.Create("stops.txt")
	CheckError("cannot create file", err)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// header
	headerRow := []string{
		"stop_id", "stop_code", "stop_name", "stop_desc", "stop_lat",
		"stop_lon", "zone_id", "stop_url", "location_type", "parent_station",
		"direction", "position"}
	err = writer.Write(headerRow)
	CheckError("Cannot write to file [se0] ", err)

	for _, stop := range stops {
		row := make([]string, len(headerRow))
		row[0] = s.TrimSpace(stop.ID)
		row[1] = s.TrimSpace(stop.ID)
		row[2] = s.TrimSpace(stop.Name)
		row[4] = fmt.Sprintf("%f", stop.Lat)
		row[5] = fmt.Sprintf("%f", stop.Lon)
		err = writer.Write(row)
		CheckError("Cannot write to file [se1] ", err)
	}
}

// RouteExporter will give routes.txt
func (h *Handler) RouteExporter() {
	file, err := os.Create("routes.txt")
	CheckError("cannot create file", err)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// header
	headerRow := []string{
		"route_id", "agency_id", "route_short_name", "route_long_name",
		"route_desc", "route_type", "route_url", "route_color",
		"route_text_color"}
	err = writer.Write(headerRow)
	CheckError("Cannot write to file [re0] ", err)

	routes := h.getDistinctDirection()
	for _, route := range routes {
		row := make([]string, len(headerRow))
		row[0] = s.TrimSpace(route)
		err = writer.Write(row)
		CheckError("Cannot write to file [re1] ", err)
	}
}

// TripExporter will give trips.txt
func (h *Handler) TripExporter(trips []string) {
	// route_id,service_id,trip_id,direction_id,block_id,shape_id,trip_type
	file, err := os.Create("trips.txt")
	CheckError("cannot create file", err)
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// header
	headerRow := []string{
		"route_id", "service_id", "trip_id", "direction_id",
		"block_id", "shape_id", "trip_type"}
	err = writer.Write(headerRow)
	CheckError("Cannot write to file [te0] ", err)

	for _, trip := range trips {
		row := make([]string, len(headerRow))
		ss := s.Split(trip, "__")
		row[0] = ss[0]
		row[2] = trip
		err = writer.Write(row)
		CheckError("Cannot write to file [te1] ", err)
	}
}

// StopTimesExporter - to export all data generated for gtfs feed
// return generated trips (tripID)
func (h *Handler) StopTimesExporter(r string, rrv string) []string {
	// file columns
	// trip_id,arrival_time,departure_time,stop_id,stop_sequence,
	// stop_headsign,pickup_type,drop_off_type,shape_dist_traveled,
	// timepoint,continuous_drop_off,continuous_pickup
	tripIDs := []string{}
	stopTimeRaws := h.ExtractTripWithRoute(r, rrv)

	fmt.Printf("exporting: stop_times\n")

	file, err := os.Create("stop_times.txt")
	CheckError("cannot create file", err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// header
	headerRow := []string{
		"trip_id", "arrival_time", "departure_time", "stop_id", "stop_sequence",
		"stop_headsign", "pickup_type", "drop_off_type", "shape_dist_traveled",
		"timepoint", "continuous_drop_off", "continuous_pickup"}
	err = writer.Write(headerRow)
	CheckError("Cannot write to file", err)

	hhmm := "15:04:05"
	currTripID := ""
	for _, ele := range stopTimeRaws {
		if currTripID != ele.TripID {
			currTripID = ele.TripID
			tripIDs = append(tripIDs, currTripID)
		}
		one := make([]string, len(headerRow))
		t1, _ := time.Parse(time.RFC3339, ele.Arrival)
		t2, _ := time.Parse(time.RFC3339, ele.Departure)
		one[0] = fmt.Sprintf("%+v", ele.TripID)
		one[1] = t1.Format(hhmm)
		one[2] = t2.Format(hhmm)
		one[3] = s.TrimSpace(ele.StopID)
		one[4] = fmt.Sprintf("%d", ele.Sequence+1)
		err = writer.Write(one)
		CheckError("Cannot write to file", err)
	}
	return tripIDs
}
