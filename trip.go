package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kellydunn/golang-geo"

	_ "github.com/lib/pq"
)

// Trip stores concise trip information in order to find stop_times
type Trip struct {
	ID      string
	Start   string
	End     string
	BoxID   string
	BeginAt Stop
	EndAt   Stop
	Comment string
}

// StopTimeRaw is to keep all schedules
type StopTimeRaw struct {
	TripID    string
	StopID    string
	Arrival   string
	Departure string
	BoxID     string
	Sequence  int
	Direction string
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
		rows.Scan(&stop.ID, &stop.Sequence, &stop.Name, &stop.Lat, &stop.Lon, &stop.IsTerminal)
		stops = append(stops, stop)
	}
	return stops
}

// TripExtractor meant to get info for GTFS's `stop_times.txt`
func (h *Handler) TripExtractor() []StopTimeRaw {
	_ = h.truncateStopTimeTable()
	fmt.Printf("start Trip Extractor\n")
	stops := h.getStops("", "ASC")
	terminals := h.getStops("is_terminal=TRUE", "ASC")
	fmt.Printf("#stop %d  #terminal %d\n", len(stops), len(terminals))
	trips := h.FindTripStartEnd(terminals)

	fmt.Println("Trip Summery")
	for ind, ele := range trips {
		order := ind + 1
		tt2, _ := time.Parse(time.RFC3339, ele.End)
		tt1, _ := time.Parse(time.RFC3339, ele.Start)
		tripDuration := tt2.Sub(tt1)
		direction := fmt.Sprintf("%s..%s", strings.TrimSpace(ele.BeginAt.ID), strings.TrimSpace(ele.EndAt.ID))
		fmt.Printf("%d.`%s`[%s] (%s) %+v - %+v => %0.1f Min \n",
			order, ele.ID, strings.TrimSpace(ele.BoxID), direction,
			ele.Start, ele.End, tripDuration.Minutes())
		if len(ele.Comment) > 0 {
			fmt.Printf("    :: %s\n", ele.Comment)
		}
		if len(ele.Comment) == 0 {
			stopTimeTable := h.FindTripTimetable(ele, stops, direction)
			for ind2, stEle := range stopTimeTable {
				order := ind2 + 1
				t2, _ := time.Parse(time.RFC3339, stEle.Departure)
				t1, _ := time.Parse(time.RFC3339, stEle.Arrival)
				duration := t2.Sub(t1)
				fmt.Printf("   %d [%s] %+v -> %0.0f s \n",
					order,
					strings.TrimSpace(stEle.StopID),
					stEle.Arrival,
					duration.Seconds())
				h.insertStopTime(stEle)
			}
		}
	}
	return nil
}

// FindTripStartEnd to get array of trips from all traces
func (h *Handler) FindTripStartEnd(terminals []Stop) []Trip {
	rows, err := h.queryTraces("", "")
	if err != nil {
		log.Fatal("queryTraces err:", err)
	}
	var (
		trips         []Trip
		boxID         string
		trip          Trip
		trace         Trace
		firstTerminal Stop
	)
	isAtTerminalInd := -1

	for rows.Next() {
		rows.Scan(&trace.BoxID, &trace.Timestamp, &trace.Lat, &trace.Lon)
		pnt := geo.NewPoint(trace.Lat, trace.Lon)
		// if box changes -> end the old one, reset stuffs
		if boxID != "" && boxID != trace.BoxID {
			fmt.Println("found a new box: ", trace.BoxID, "   | old one: ", boxID)
			boxID = ""
			trip = Trip{}
			firstTerminal = Stop{}
			isAtTerminalInd = -1
		}
		if boxID == "" {
			boxID = trace.BoxID
		}
		// result in km
		terminalsDistance := make([]float64, len(terminals))
		for ind, ele := range terminals {
			tGeoPoint := geo.NewPoint(ele.Lat, ele.Lon)
			terminalsDistance[ind] = pnt.GreatCircleDistance(tGeoPoint)
		}
		// init trip
		if firstTerminal == (Stop{}) {
			gotFirst := false
			for ind, dist := range terminalsDistance {
				if dist < h.rangeWithinStop {
					firstTerminal = terminals[ind]
					gotFirst = true
					isAtTerminalInd = ind
					break
				}
			}
			if gotFirst == false {
				// if we haven't found the first terminal,
				// prior to this data is no good anyway
				continue
			}
			// init trip
			trip.BeginAt = terminals[isAtTerminalInd]
			trip.Start = trace.Timestamp
			trip.BoxID = boxID
		}
		// skip anything if is at the same terminal
		if isAtTerminalInd != -1 {
			// check if it's at the same terminal? then continue
			if terminalsDistance[isAtTerminalInd] < h.rangeWithinStop {
				// trip start when it departs the terminal,
				// not when it first arrives
				trip.Start = trace.Timestamp
				continue
			} else {
				isAtTerminalInd = -1
			}
		}
		for ind, dist := range terminalsDistance {
			if dist < h.rangeWithinStop {
				// this is what it shouldn't be... maybe GPS sucks
				if isAtTerminalInd != -1 {
					trip.Comment = "[GPSSucks] jumping from terminal to terminal, huh?"
				} else if terminals[ind] == trip.BeginAt {
					trip.Comment = "[IncompletedTrip] round trip w/o hitting another terminal"
				}
				trip.End = trace.Timestamp
				trip.EndAt = terminals[ind]
				trip.ID = RandString(6)
				trips = append(trips, trip)
				trip = Trip{}
				isAtTerminalInd = -1
				firstTerminal = Stop{}
			}
		}
	}
	return trips
}

// FindTripTimetable to get detail of trip and stop along the way
// and interpolate if there is no data stopping at the stop
func (h *Handler) FindTripTimetable(trip Trip, stops []Stop, direction string) []StopTimeRaw {
	q := fmt.Sprintf("box_id = '%s' AND timestamp >= '%s' AND timestamp <= '%s'",
		trip.BoxID, trip.Start, trip.End)
	// fmt.Println(q)
	rows, err := h.queryTraces(q, "ASC")
	if err != nil {
		log.Fatal(err)
	}
	var (
		trace    Trace
		boxID    string
		stopTime StopTimeRaw
	)
	results := make([]StopTimeRaw, len(stops))
	for rows.Next() {
		rows.Scan(&trace.BoxID, &trace.Timestamp, &trace.Lat, &trace.Lon)
		pnt := geo.NewPoint(trace.Lat, trace.Lon)
		// if box changes -> end the old one (or save prev if applicant)
		if boxID != "" && boxID != trace.BoxID && stopTime != (StopTimeRaw{}) {
			results[stopTime.Sequence] = stopTime
			stopTime = StopTimeRaw{}
		}
		if boxID == "" {
			boxID = trace.BoxID
		}

		atTheStop := -1
		for ind, ele := range stops {
			tGeoPoint := geo.NewPoint(ele.Lat, ele.Lon)
			distance := pnt.GreatCircleDistance(tGeoPoint)
			if distance < h.rangeWithinStop {
				atTheStop = ind
				if stopTime == (StopTimeRaw{}) {
					// init stopTime
					stopTime.BoxID = trace.BoxID
					stopTime.TripID = trip.ID
					stopTime.Arrival = trace.Timestamp
					stopTime.Departure = trace.Timestamp
					stopTime.StopID = ele.ID
					stopTime.Direction = direction
					stopTime.Sequence = ind
				} else {
					// check if it's still at the same stop
					//   or close the prev one and start the new one
					if stopTime.StopID == ele.ID {
						stopTime.Departure = trace.Timestamp
						continue
					} else {
						// close the old one & init the one one
						results[stopTime.Sequence] = stopTime
						stopTime = StopTimeRaw{
							BoxID:     trace.BoxID,
							TripID:    trip.ID,
							Arrival:   trace.Timestamp,
							Departure: trace.Timestamp,
							StopID:    ele.ID,
							Direction: direction,
							Sequence:  ind,
						}
					}

				}
			}
		}
		if atTheStop == -1 {
			// mean it's not at any stop
			if stopTime != (StopTimeRaw{}) {
				// close it
				results[stopTime.Sequence] = stopTime
				stopTime = StopTimeRaw{}
			}
		}
	}
	if results[stopTime.Sequence] == (StopTimeRaw{}) {
		results[stopTime.Sequence] = stopTime
	}
	results = FillupMissingStopTime(results, stops, direction)
	return results
}

// FillupMissingStopTime by interpolating
func FillupMissingStopTime(l []StopTimeRaw, stops []Stop, direction string) []StopTimeRaw {
	for ind, ele := range l {
		if ele != (StopTimeRaw{}) {
			continue
		}
		fmt.Printf("[%s] This stop needs attention.\n", strings.TrimSpace(stops[ind].ID))
		// no way we can miss first and last since we won't have a Trip to begin with
		if ind == 0 || ind == len(l) {
			continue
		}
		// NOTE: this doesn't handle anything but missing just one StopTime
		interpolated := simpleInterpolation(l[ind-1], l[ind+1], stops, ind)
		interpolated.Direction = direction
		l[ind] = interpolated
	}
	return l
}

// Linear StopTime interpolation, simple and effective
func simpleInterpolation(before StopTimeRaw, after StopTimeRaw, stops []Stop, targetInd int) StopTimeRaw {
	result := StopTimeRaw{
		TripID:   before.TripID,
		StopID:   stops[targetInd].ID,
		BoxID:    before.BoxID,
		Sequence: before.Sequence + 1,
	}
	t2, _ := time.Parse(time.RFC3339, after.Arrival)
	t1, _ := time.Parse(time.RFC3339, before.Departure)
	overlapPeriod := t2.Sub(t1)
	pnt2 := geo.NewPoint(stops[targetInd+1].Lat, stops[targetInd+1].Lon)
	pnt1 := geo.NewPoint(stops[targetInd-1].Lat, stops[targetInd-1].Lon)
	pntTarget := geo.NewPoint(stops[targetInd].Lat, stops[targetInd].Lon)

	targetToNext := pntTarget.GreatCircleDistance(pnt2)
	prevToTarget := pnt1.GreatCircleDistance(pntTarget)

	targetPeriod := prevToTarget / (prevToTarget + targetToNext) * overlapPeriod.Seconds()

	targetArrivalTime := t1.Add(time.Duration(targetPeriod) * time.Second)
	result.Arrival = targetArrivalTime.Format(time.RFC3339)
	result.Departure = result.Arrival
	return result
}
