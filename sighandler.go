package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//sigHandle will trap/handle the signal and set global flag
func sigHandle() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGKILL, syscall.SIGTERM)
	go func() {
		for s := range sigc {
			pStillRunning = false
			log.Println("Oops, signal -> ", s)
			time.Sleep(time.Second * 30)
			log.Println("Gracefully exit now!")
			os.Exit(0)
		}
	}()
}
