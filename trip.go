package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kellydunn/golang-geo"
)

type Trip struct {
	Start string
	End   string
	BoxID string
}

type StopTime struct {
	TripID    string
	StopID    string
	Arrival   string
	Departure string
	BoxID     string
	sequence  int
}

func (h *Handler) getStops(where string, order string) []Stop {
	var stops []Stop
	rows, err := h.queryStops(where, order)
	if err != nil {
		fmt.Printf("getStop err: %+v", err)
		os.Exit(1)
	}
	for rows.Next() {
		var stop Stop
		rows.Scan(&stop.ID, &stop.Name, &stop.Lat, &stop.Lon, &stop.Sequence, &stop.IsTerminal)
		stops = append(stops, stop)
	}
	return stops
}

// TripExtractor meant to get info for GTFS's `stop_times.txt`
func (h *Handler) TripExtractor() []StopTime {
	fmt.Printf("start Trip Extractor\n")
	stops := h.getStops("", "ASC")
	terminals := h.getStops("is_terminal=true", "ASC")
	fmt.Printf(" #stop %d  #terminal %d", len(stops), len(terminals))
	// traces := h.getTraces()
	trips := h.FindTripStartEnd(terminals)

	for _, ele := range trips {
		fmt.Printf(" >> trip: [%s] %+v - %+v \n", strings.TrimSpace(ele.BoxID), ele.Start, ele.End)
	}
	return nil
}

// FindTripStartEnd to get array of trips from all traces
func (h *Handler) FindTripStartEnd(terminals []Stop) []Trip {
	var trips []Trip
	first := terminals[0]
	second := terminals[1]
	t1 := geo.NewPoint(float64(first.Lat), float64(first.Lon))
	t2 := geo.NewPoint(float64(second.Lat), float64(second.Lon))
	fmt.Printf("t1: %+v\n", t1)
	fmt.Printf("t2: %+v\n", t2)

	rows, err := h.queryTraces("", "")
	if err != nil {
		fmt.Printf("queryTraces err: %+v", err)
		os.Exit(1)
	}
	var trip Trip
	var trace Trace
	closestT1 := 1.10
	closestT2 := 1.10
	for rows.Next() {
		var lat float32
		var lon float32
		rows.Scan(&trace.BoxID, &trace.Timestamp, &lat, &lon)
		// fmt.Printf(" trace (%f, %f)   <-->  (%f, %f)\n", trace.Lat, trace.Lon, lat, lon)
		if trace.Lat == lat && trace.Lon == lon {
			// skip processing idling one
			continue
		}
		trace.Lat = lat
		trace.Lon = lon
		// fmt.Printf("trace: %+v\n", trace)
		pnt := geo.NewPoint(float64(trace.Lat), float64(trace.Lon))
		// result in km
		distT1 := pnt.GreatCircleDistance(t1)
		distT2 := pnt.GreatCircleDistance(t2)
		if distT1 < closestT1 {
			closestT1 = distT1
		}
		if distT2 < closestT2 {
			closestT2 = distT2
		}
		if distT1 < 0.02 || distT2 < 0.02 {
			fmt.Printf("tmsp: %+v   distance: <t1> %0.05f <t2> %0.05f \n", trace.Timestamp, distT1, distT2)
		}
		if distT2 < 0.4 && trip.BoxID == "" {
			continue
		}
		if trip.BoxID == "" {
			trip.BoxID = trace.BoxID
		} else if trip.BoxID != trace.BoxID {
			trip.End = "-"
			trips = append(trips, trip)
			trip = Trip{BoxID: trace.BoxID}
		}
		if trip.Start == "" && distT1 < 0.4 {
			trip.Start = trace.Timestamp
		}
		if distT2 < 0.4 {
			trip.End = trace.Timestamp
			trips = append(trips, trip)
			trip = Trip{}
		}
	}
	fmt.Printf("closest t1: %+v   t2: %+v", closestT1, closestT2)
	return trips
}

// FindTripTimetable to get detail of trip and stop along the way
func (h *Handler) FindTripTimetable(trip Trip) []StopTime {
	return nil
}
