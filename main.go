package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

var (
	localAddr           = ":3000"
	remoteAddr          = "127.0.0.1:8031"
	dialTimeoutSec      = 5
	samplingIntervalSec = 1
	topSvcNum           = 10

	observer *Observer
)

func init() {
	flag.StringVar(&localAddr, "lo", localAddr, "observer listen address")
	flag.StringVar(&remoteAddr, "rmt", remoteAddr, "skynet server dial address")
	flag.IntVar(&dialTimeoutSec, "ti", dialTimeoutSec, "dial timeout second")
	flag.IntVar(&samplingIntervalSec, "i", samplingIntervalSec, "sampling interval second")
	flag.IntVar(&topSvcNum, "top", topSvcNum, "display top n services")
}

func handleStart(w http.ResponseWriter, _ *http.Request) {
	if !observer.Start() {
		w.Write([]byte("observer start again!!"))
		return
	}
	w.Write([]byte("Sampling..."))
}

func handleStop(w http.ResponseWriter, _ *http.Request) {
	if !observer.Stop() {
		w.Write([]byte("observer not start!!"))
		return
	}
	observer.dumpToChart(w, topSvcNum)
}

func main() {
	flag.Parse()
	observer = &Observer{}
	observer.Init(remoteAddr, time.Duration(dialTimeoutSec)*time.Second)
	log.Println("observer init ok!")

	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/stop", handleStop)
	err := http.ListenAndServe(localAddr, nil)
	if err != nil {
		log.Fatalf("observer listen local address failed, %v\n", err)
	}
	log.Printf("observer listen local address %s ok!\n", localAddr)
}
