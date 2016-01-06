package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	sc "syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	logadpt "github.com/geterns/logadpt"
)

var (
	// config instance
	cfgIns config

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
	logFile := flag.String("log", "../logs/massive_down.log", "log file")
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
	if err := cfgIns.loadConfig(*confFile); err != nil {
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
	for pass := int32(0); pass < cfgIns.RequestPerRoutine; pass++ {
		fileIndex := rand.Int63n(
			cfgIns.MaxFileSizeTenMegaByte -
				cfgIns.MinFileSizeTenMegaByte + 1)
		fileSizeMegaByte := (cfgIns.MinFileSizeTenMegaByte + fileIndex) * 10
		url := fmt.Sprintf("%s/%dM?%s", cfgIns.UrlRoot, fileSizeMegaByte,
			cfgIns.UrlPara)

		req, reqErr := http.NewRequest("GET", url, nil)
		if reqErr != nil {
			log.WithFields(log.Fields{
				"worker": id,
				"pass":   pass,
				"url":    url,
				"error":  reqErr.Error(),
			}).Errorln("Failed to make HTTP request")
			continue
		}

		testBlockSizeMegaByte := rand.Int63n(
			cfgIns.MaxTestBlockSizeMegaByte-
				cfgIns.MinTestBlockSizeMegaByte+1) +
			cfgIns.MinTestBlockSizeMegaByte
		if testBlockSizeMegaByte > fileSizeMegaByte {
			testBlockSizeMegaByte = fileSizeMegaByte
		}

		testBlockStartPos := rand.Int63n(fileSizeMegaByte-
			testBlockSizeMegaByte+1) << 20
		testBlockEndPos := testBlockStartPos +
			(testBlockSizeMegaByte << 20) - 1

		req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d",
			testBlockStartPos, testBlockEndPos))
		startTime := time.Now()
		resp, respErr := client.Do(req)
		if respErr != nil {
			log.WithFields(log.Fields{
				"worker": id,
				"pass":   pass,
				"url":    url,
				"test_block_size_mega_byte": testBlockSizeMegaByte,
				"range":                     req.Header.Get("Range"),
				"error":                     respErr.Error(),
			}).Errorln("Failed to do HTTP request")
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 206 {
			log.WithFields(log.Fields{
				"worker": id,
				"pass":   pass,
				"url":    url,
				"test_block_size_mega_byte": testBlockSizeMegaByte,
				"range":                     req.Header.Get("Range"),
				"status_code":               resp.StatusCode,
			}).Errorln("Request failed")
			continue
		}

		fileName := filepath.Join(cfgIns.DataDir,
			fmt.Sprintf("%d_%d_%d_%d.dat", id, pass,
				fileSizeMegaByte<<20,
				testBlockSizeMegaByte<<20))
		out, outErr := os.Create(fileName)
		if outErr != nil {
			log.WithFields(log.Fields{
				"worker": id,
				"pass":   pass,
				"url":    url,
				"test_block_size_mega_byte": testBlockSizeMegaByte,
				"range":                     req.Header.Get("Range"),
				"error":                     outErr.Error(),
			}).Errorln("Failed to open data file")
			continue
		}
		defer out.Close()

		if n, ioErr := io.Copy(out, resp.Body); ioErr != nil {
			log.WithFields(log.Fields{
				"worker": id,
				"pass":   pass,
				"url":    url,
				"test_block_size_mega_byte": testBlockSizeMegaByte,
				"range":                     req.Header.Get("Range"),
				"error":                     ioErr.Error(),
			}).Errorln("Failed to write data file")
		} else {
			timeUsedSecond := float64(time.Since(startTime).Nanoseconds()) / 1000000000.0
			log.WithFields(log.Fields{
				"worker": id,
				"pass":   pass,
				"url":    url,
				"test_block_size_mega_byte": testBlockSizeMegaByte,
				"range":                     req.Header.Get("Range"),
				"write_size":                n,
				"time_used_second":          timeUsedSecond,
				"average_speed":             fmt.Sprintf("%.2f KiB/s", float64(n)/timeUsedSecond/1024.0),
			}).Debugln("Done")
		}
	}

	done <- true
}

func main() {
	log.Infoln("Start test, config = ", cfgIns)
	for id := int32(0); id < cfgIns.RoutineNumber; id++ {
		go worker(id)
	}
	for id := int32(0); id < cfgIns.RoutineNumber; id++ {
		<-done
	}
}
