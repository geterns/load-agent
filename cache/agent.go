package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	sc "syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	config "github.com/geterns/load-agent/config"
	dummy "github.com/geterns/load-agent/dummy"
	logadpt "github.com/geterns/logadpt"
)

var (
	// config instance
	cfgIns config.Config

	jobs = make(chan int64, 64)
	done = make(chan bool)
)

func signalHandler() {
	listener := make(chan os.Signal)
	signal.Notify(listener, sc.SIGINT, sc.SIGABRT, sc.SIGKILL, sc.SIGTERM)
	caught := <-listener
	log.Infoln("Caught a signal:", caught)

	os.Exit(0)
}

func init() {
	// parse command line arguments
	confFile := flag.String("conf", "../conf/config.json", "config file")
	logFile := flag.String("log", "../logs/cache-agent.log", "log file")
	flag.Parse()
	// open log file
	log.SetOutput(&logadpt.FileRotator{
		FileName:    *logFile,
		MaxSize:     100 << 20,
		MaxDuration: 1 * time.Hour,
	})
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&logadpt.ClassicFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldsDelimiter: ", ",
	})
	log.Infoln("Opened log file")
	// load config file
	if err := cfgIns.LoadConfig(*confFile); err != nil {
		log.Errorln("Failed to load config file:", err.Error())
		os.Exit(-1)
	}
	log.Infoln("Loaded config file")
	// set seed
	rand.Seed(time.Now().UnixNano())
	// multiple processes setting
	runtime.GOMAXPROCS(runtime.NumCPU())
	// install signal handler
	go signalHandler()
}

func worker(id int32) {
	client := &http.Client{}
	for {
		fileSizeTenMegaByte, ok := <-jobs

		if !ok {
			log.WithFields(log.Fields{
				"worker": id,
			}).Infoln("Worker terminate")
		}

		// Set file
		url := fmt.Sprintf("%s/%dM?%s", cfgIns.UrlRoot, fileSizeTenMegaByte*10, cfgIns.UrlPara)

		for attempt := int32(1); attempt <= cfgIns.CacheAgentMaxAttempt; attempt++ {
			req, reqErr := http.NewRequest("GET", url, nil)
			if reqErr != nil {
				log.WithFields(log.Fields{
					"worker": id,
					"url":    url,
					"error":  reqErr.Error(),
				}).Errorln("Failed to make HTTP request")
				continue
			}

			// Send request
			startTime := time.Now()
			resp, respErr := client.Do(req)
			if respErr != nil {
				log.WithFields(log.Fields{
					"worker":  id,
					"url":     url,
					"attempt": attempt,
					"error":   respErr.Error(),
				}).Errorln("Failed to do HTTP request")
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				log.WithFields(log.Fields{
					"worker":      id,
					"url":         url,
					"attempt":     attempt,
					"status_code": resp.StatusCode,
				}).Errorln("Request failed")
				continue
			}

			// Read response, calculate duration
			requestCompleteTime := time.Now()
			out := dummy.DummyWriter{new(bool), new(time.Time), new(time.Time), new(time.Duration)}

			if n, ioErr := io.Copy(out, resp.Body); ioErr != nil {
				log.WithFields(log.Fields{
					"worker":  id,
					"url":     url,
					"attempt": attempt,
					"error":   ioErr.Error(),
				}).Errorln("Failed to write data file")
			} else {
				requestTimeMSecond :=
					float64(requestCompleteTime.Sub(startTime).Nanoseconds()) / 1000000.0
				firstByteArrivalTimeMSecond :=
					float64(out.FirstByteArrivalTime.Sub(startTime).Nanoseconds()) / 1000000.0
				maxWaitTimeMSecond :=
					float64(out.MaxWaitDuration.Nanoseconds()) / 1000000.0
				timeUsedSecond :=
					float64(time.Since(requestCompleteTime).Nanoseconds()) / 1000000000.0
				log.WithFields(log.Fields{
					"worker":                          id,
					"url":                             url,
					"attempt":                         attempt,
					"write_size":                      n,
					"requestTimeMSecond":              requestTimeMSecond,
					"first_byte_arrival_time_msecond": firstByteArrivalTimeMSecond,
					"max_wait_duration_msecond":       maxWaitTimeMSecond,
					"time_used_second":                timeUsedSecond,
					"average_speed":                   fmt.Sprintf("%.2f KiB/s", float64(n)/timeUsedSecond/1024.0),
				}).Debugln("Done")

				break
			}
		}
	}

	done <- true
}

func main() {
	log.Infoln("Start cache-agent, config = ", cfgIns)

	for id := int32(0); id < cfgIns.CacheAgentWorkerNumber; id++ {
		go worker(id)
	}

	for job := cfgIns.MinFileSizeTenMegaByte; job <= cfgIns.MaxFileSizeTenMegaByte; job++ {
		jobs <- job
	}

	close(jobs)

	for id := int32(0); id < cfgIns.CacheAgentWorkerNumber; id++ {
		<-done
	}
}
