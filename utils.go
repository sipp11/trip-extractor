package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type (
	// Handler to handle all route
	Handler struct {
		db              *sql.DB
		port            string
		rangeWithinStop float64
		verbose         bool
	}

	// Result for all input handlers
	Result struct {
		Success int    `json:"success"`
		Failed  int    `json:"failed"`
		Message string `json:"message"`
	}
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// RandString to generate random string
// reference: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang/
func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// LogPrint - easy to maintain verbose mode
func (h *Handler) LogPrint(msg string) {
	if h.verbose {
		fmt.Print(msg)
	}
}

// CheckError is a shorthanded func for a simple error validation
func CheckError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

// CheckDataCompleteness - to do initial check if data is good enough to process
func (h *Handler) CheckDataCompleteness() bool {
	stopCnt, _ := h.ItemCount("stops")
	stopAndRouteCnt, _ := h.ItemCount("stop_and_route")
	traceCnt, _ := h.ItemCount("traces")
	if stopCnt < 6 || stopAndRouteCnt < 6 {
		fmt.Printf("#stops = %d\n > which is NOT enough to do anything meaningful\n", stopCnt)
		return false
	}
	if traceCnt < 60 {
		fmt.Printf("#traces = %d\n > which is NOT enough to do anything meaningful\n", traceCnt)
		return false
	}
	return true
}
