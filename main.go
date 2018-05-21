package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	ini "gopkg.in/ini.v1"
)

var (
	verbose   = flag.Bool("v", false, "Verbose mode")
	outputDir = flag.String("dir", "output", "GTFS output directory")
	day       = flag.String("day", "", "Filtered day (Mon, Tue, ...)")
	route     = flag.String("rt", "", "route_id")
	routeRev  = flag.String("rtrv", "", "route_id for reverse (use the same route if not specified)")
	radius    = flag.Int("radius", 50, "Radius in meter for checking stop")
)

var usage = `Usage: trip_extractor [options...] <cmd>
Options:
  -v        verbosely
  -dir      GTFS output directory
  -day      Filtered day (Mon, Tue, ...) default: no filter
  -rt       route_id (1)
  -rtrv     [optional] route_id (2) for reverse
            (use the same route_id if not specified)
  -radius   Radius (m) for stop detection
            (50m as default)

Command:

  initdb      to create all necessary tables
  web         to serve web
  gen         to generate timetable
  gtfs        to generate GTFS feed: stop_times.txt
  flushdb     to drop all and create new tables
  geom_regen  to update all records with "geom type" from lat, lon fields
`

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()
	if flag.NArg() < 1 {
		usageAndExit("")
	}
	cfg, err := ini.Load("my.ini")
	CheckError("Fail to read my.ini", err)

	port := cfg.Section("app").Key("port").String()
	dbConn := cfg.Section("db").Key("path").String()
	rangeWithinStop, err := cfg.Section("app").Key("range_km_within_stop").Float64()
	if err != nil {
		log.Fatal("range_km_within_stop cannot be casted to float64")
	}
	db, err := sql.Open("postgres", dbConn)
	defer db.Close()
	CheckError("Fail to connect to db server", err)

	h := &Handler{
		db:              db,
		port:            port,
		rangeWithinStop: rangeWithinStop,
		verbose:         *verbose,
		outputDir:       *outputDir,
		day:             *day,
	}
	args := flag.Args()

	switch args[0] {

	case "web":
		h.serveWebInterface()

	case "initdb":

		fmt.Printf("Database Initialization? ")
		err := h.InitDB()
		if err != nil {
			fmt.Print("yes")
		} else {
			fmt.Printf("no because %+v", err)
		}

	case "flushdb":
		err := h.FlushDatabase()
		CheckError("Database flush error: ", err)
		fmt.Println("Database flushed")

	case "geom_regen":
		err := h.GeomRegenerate()
		CheckError("GEOM regeneration error: ", err)
		fmt.Println("GEOM updated")

	case "gen":
		if len(*route) == 0 {
			usageAndExit("No route_id specified")
		}
		// fmt.Printf("checking if data is good? ")
		// isGood := h.CheckDataCompleteness()
		// if !isGood {
		// 	fmt.Printf("WARNING: Not enough data to work on\n")
		// 	os.Exit(1)
		// }
		// fmt.Printf(" yes\n")
		h.TripExtractor(*route, *routeRev)

	case "gtfs":
		if len(*route) == 0 {
			usageAndExit("No route_id specified")
		}
		// fmt.Println("GTFS is a work in progress...")
		// fmt.Printf("checking if data is good? ")
		// isGood := h.CheckDataCompleteness()
		// if !isGood {
		// 	fmt.Printf("WARNING: Not enough data to work on\n")
		// 	os.Exit(1)
		// }
		// fmt.Printf(" yes\n")
		h.GTFSExporter(*route, *routeRev)

	default:
		usageAndExit("")
	}
}
