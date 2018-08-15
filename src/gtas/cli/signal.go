package cli

import (
	"os"
	"os/signal"
	"syscall"
)

func signals() <-chan bool {
	quit := make(chan bool)

	go func() {
		signals := make(chan os.Signal)
		defer close(signals)

		signal.Notify(signals, syscall.SIGQUIT, syscall.SIGTERM, os.Interrupt, os.Kill)
		defer signalStop(signals)

		<-signals
		quit <- true
	}()

	return quit
}

func signalStop(c chan<- os.Signal) {
	signal.Stop(c)
}
