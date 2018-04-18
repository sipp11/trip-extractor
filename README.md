# Trip Extractor

It's a tool to process GPS traces and give back a trip summary.

# Dependency

* postreSQL + postgis server

# How to get it works

Create config file as `my.ini` with content below


    [app]
    port = 9090
    range_km_within_stop = 0.025

    [db]
    path = postgres://user:password@localhost/table_name?sslmode=disable


# Input

Planned to have input via REST interface

* stops
    fields:
        stop_id     string
        stop_name   string
        stop_lat    float32
        stop_lon    float32
        is_terminal bool
    2 terminals required to find a trip

* trace
    fields:
        box_id      string
        timestamp   string (ISO Datetime)
        lat         float32
        lon         float32


# Output

* trip summary
* trip list
* schedule
    * first to last for both direction
* schedule at each stop (arrived & departed)
* avg trip duration
    * whole trip
    * each stop pair
