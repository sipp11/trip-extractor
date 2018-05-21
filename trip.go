package main

import (
	"fmt"
	s "strings"
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

func (h *Handler) getStops(route string, order string, onlyTerminal bool) []Stop {
	var stops []Stop
	rows, err := h.queryStops(route, order, onlyTerminal)
	CheckError("getStop", err)
	for rows.Next() {
		var stop Stop
		rows.Scan(&stop.ID, &stop.Sequence, &stop.Name, &stop.Lat, &stop.Lon, &stop.IsTerminal, &stop.LocationType, &stop.ParentStation)
		stops = append(stops, stop)
	}
	return stops
}

// TripExtractor meant to get info for GTFS's `stop_times.txt`
func (h *Handler) TripExtractor(route string, routeRev string) []StopTimeRaw {
	_ = h.truncateStopTimeTable()
	fmt.Printf("start Trip Extractor\n")
	return h.ExtractTripWithRoute(route, routeRev)
}

// ExtractTripWithRoute - has a limit that stop at the end has to be
// the same name otherwise, it would not work
func (h *Handler) ExtractTripWithRoute(route string, routeRev string) []StopTimeRaw {
	bkk, _ := time.LoadLocation("Asia/Bangkok")
	allTrips := []StopTimeRaw{}
	// Route for each direction
	stopDirection := make(map[string][]Stop, 2)
	stopDirection[route] = h.getStops(route, "ASC", false)
	if len(routeRev) > 0 {
		stopDirection[routeRev] = h.getStops(routeRev, "ASC", false)
	} else {
		// make reverse stops/route manually
		routeRev = fmt.Sprintf("%s-rev", route)
		stopDirection[routeRev] = h.getStops(route, "DESC", false)
		for ind, stop := range stopDirection[routeRev] {
			stop.Sequence = ind + 1
		}
	}
	fwdTrip := h.findOneWayTripPeriod(
		stopDirection[route][0],
		stopDirection[route][len(stopDirection[route])-1],
		route)
	revTrip := h.findOneWayTripPeriod(
		stopDirection[routeRev][0],
		stopDirection[routeRev][len(stopDirection[routeRev])-1],
		routeRev)

	fmt.Printf("\n%s\n", route)
	// for _, ele := range stopDirection[route] {
	// 	fmt.Print(ele.Sequence, ". ", s.TrimSpace(ele.ID), " -> ")
	// }
	hhmm := "15:04:05"
	for ind, trip := range fwdTrip {
		tt2, _ := time.Parse(time.RFC3339, trip.End)
		tt1, _ := time.Parse(time.RFC3339, trip.Start)
		day := tt1.In(bkk).Format("Mon")
		if h.day != "" && day != h.day {
			continue
		}
		tripDuration := tt2.Sub(tt1)
		fmt.Printf("%d. %.0f min: [%s] %s -> %s  /%s/\n",
			ind+1, tripDuration.Minutes(),
			day,
			tt1.In(bkk).Format(hhmm), tt2.In(bkk).Format(hhmm),
			s.TrimSpace(trip.BoxID))
		h.LogPrint(fmt.Sprintf("     %s\n", s.TrimSpace(trip.BoxID)))
		stopTimeRaws := h.FindTripTimeTable(trip, stopDirection[route], route)
		allTrips = append(allTrips, stopTimeRaws...)
		h.printAndInsertTimeTable(stopTimeRaws)
	}
	fmt.Printf("\n%s\n", routeRev)
	for ind, trip := range revTrip {
		tt2, _ := time.Parse(time.RFC3339, trip.End)
		tt1, _ := time.Parse(time.RFC3339, trip.Start)
		day := tt1.In(bkk).Format("Mon")
		if h.day != "" && day != h.day {
			continue
		}
		tripDuration := tt2.Sub(tt1)
		fmt.Printf("%d. %.0f min: [%s] %s -> %s  /%s/\n",
			ind+1, tripDuration.Minutes(),
			tt1.In(bkk).Format("Mon"),
			tt1.In(bkk).Format(hhmm), tt2.In(bkk).Format(hhmm),
			s.TrimSpace(trip.BoxID))
		h.LogPrint(fmt.Sprintf("     %s\n", s.TrimSpace(trip.BoxID)))
		stopTimeRaws := h.FindTripTimeTable(trip, stopDirection[routeRev], routeRev)
		allTrips = append(allTrips, stopTimeRaws...)
		h.printAndInsertTimeTable(stopTimeRaws)
	}
	return allTrips
}

func (h *Handler) printAndInsertTimeTable(stt []StopTimeRaw) {
	bkk, _ := time.LoadLocation("Asia/Bangkok")

	for _, stEle := range stt {
		t2, _ := time.Parse(time.RFC3339, stEle.Departure)
		t1, _ := time.Parse(time.RFC3339, stEle.Arrival)

		duration := t2.Sub(t1)
		h.LogPrint(fmt.Sprintf("   %d /%s/ [%s] %+v -> %0.0f s \n",
			stEle.Sequence+1,
			s.TrimSpace(stEle.Direction),
			s.TrimSpace(stEle.StopID),
			t1.In(bkk).Format(time.RFC1123Z),
			duration.Seconds()))
		h.insertStopTime(stEle)
	}
}

func (h *Handler) findOneWayTripPeriod(beginAt Stop, endAt Stop, tripPrefix string) []Trip {

	whereArr := make([]string, 2)
	// filter trace for only what inside this sphere (50 m radius)
	// both terminals -- so we don't have to process traces in between
	distance := int(h.rangeWithinStop * 1000)
	tmpl := "ST_DistanceSphere(geom, ST_MakePoint(%f,%f)) <= %d"
	whereArr[0] = fmt.Sprintf(tmpl, beginAt.Lon, beginAt.Lat, distance)
	whereArr[1] = fmt.Sprintf(tmpl, endAt.Lon, endAt.Lat, distance)
	whereClause := s.Join(whereArr, " OR ")
	rows, err := h.queryTraces(whereClause, "ASC")
	CheckError("Find traces inside terminals", err)

	var (
		trips []Trip
		boxID string
		trip  Trip
		trace Trace
	)
	beginPoint := geo.NewPoint(beginAt.Lat, beginAt.Lon)
	endPoint := geo.NewPoint(endAt.Lat, endAt.Lon)
	tripCounter := 1
	for rows.Next() {
		rows.Scan(&trace.BoxID, &trace.Timestamp, &trace.Lat, &trace.Lon)
		pnt := geo.NewPoint(trace.Lat, trace.Lon)

		// reset anything if BoxID changes
		if boxID != trace.BoxID {
			trip = Trip{}
		}

		// start checking if it's at the first terminal
		bDistance := beginPoint.GreatCircleDistance(pnt)
		if bDistance < h.rangeWithinStop {
			if trip.Start == "" {
				// init this trip
				boxID = trace.BoxID
				trip = Trip{
					BeginAt: beginAt,
					Start:   trace.Timestamp,
					BoxID:   trace.BoxID,
				}
			} else {
				// if it's still at the first terminal, set new start time
				trip.Start = trace.Timestamp
			}
			continue
		}
		// if trip is initialized yet, no point checking the rest
		if trip.Start == "" {
			continue
		}
		// checking if it's at the second terminal
		eDistance := endPoint.GreatCircleDistance(pnt)
		if eDistance < h.rangeWithinStop {
			if trip.Start != "" {
				t2, _ := time.Parse(time.RFC3339, trace.Timestamp)
				t1, _ := time.Parse(time.RFC3339, trip.Start)
				diff := t2.Sub(t1)
				if diff.Hours() < 3.0 {
					// end this trip
					trip.End = trace.Timestamp
					trip.EndAt = endAt
					trip.ID = fmt.Sprintf("%s__%d", tripPrefix, tripCounter)
					trips = append(trips, trip)
					trip = Trip{}
					tripCounter++
				} else {
					// reset when trip is way too long, should be bad one
					trip = Trip{}
				}
			}
		}
	}
	return trips
}

// FindTripTimeTable to get detail of trip and stop along the way
// and interpolate if there is no data stopping at the stop
func (h *Handler) FindTripTimeTable(t Trip, stops []Stop, d string) []StopTimeRaw {
	q := fmt.Sprintf("box_id = '%s' AND timestamp >= '%s' AND timestamp <= '%s'",
		t.BoxID, t.Start, t.End)
	rows, err := h.queryTraces(q, "ASC")
	CheckError("findTripTimeTable 00", err)
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
					stopTime.TripID = t.ID
					stopTime.Arrival = trace.Timestamp
					stopTime.Departure = trace.Timestamp
					stopTime.StopID = ele.ID
					stopTime.Direction = d
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
							TripID:    t.ID,
							Arrival:   trace.Timestamp,
							Departure: trace.Timestamp,
							StopID:    ele.ID,
							Direction: d,
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
	results = FillupMissingStopTime(results, stops, d)
	return results
}

// FillupMissingStopTime by interpolating
func FillupMissingStopTime(l []StopTimeRaw, stops []Stop, direction string) []StopTimeRaw {
	i := -1
	j := -1
	for ind, ele := range l {
		if ele == (StopTimeRaw{}) {
			continue
		}
		if i == -1 {
			i = ind
			continue
		}
		// fill up missing one
		j = ind
		orig := l[i]
		origTime, _ := time.Parse(time.RFC3339, orig.Departure)
		distanceIJ := distanceBetween(stops[i], stops[j])
		durationIJ := durationBetween(l[i].Departure, l[j].Arrival)
		for k := i + 1; k < j; k++ {
			distanceIK := distanceBetween(stops[i], stops[k])
			durationIK := distanceIK / distanceIJ * float64(durationIJ)

			arrivedAt := origTime.Add(time.Duration(durationIK))
			l[k] = StopTimeRaw{
				TripID:    orig.TripID,
				StopID:    stops[k].ID,
				Arrival:   arrivedAt.Format(time.RFC3339),
				Departure: arrivedAt.Format(time.RFC3339),
				BoxID:     orig.BoxID,
				Sequence:  k,
				Direction: orig.Direction,
			}
		}
		i = ind
		j = -1
	}
	return l
}

func distanceBetween(a Stop, b Stop) float64 {
	aPoint := geo.NewPoint(a.Lat, a.Lon)
	bPoint := geo.NewPoint(b.Lat, b.Lon)
	return aPoint.GreatCircleDistance(bPoint)
}

func durationBetween(a string, b string) time.Duration {
	t2, _ := time.Parse(time.RFC3339, b)
	t1, _ := time.Parse(time.RFC3339, a)
	return t2.Sub(t1)
}
