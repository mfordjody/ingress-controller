package taskgroup

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Group struct {
	wg sync.WaitGroup
}

func (g *Group) Go(fn func()) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		fn()
	}()
}

func (g *Group) Wait() {
	g.wg.Wait()
}

func SetupStopSignalChan() <-chan struct{} {
	stopChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	go func() {
		<-signalChan
		log.Println("Stop signal received, shutting down...")
		close(stopChan)
	}()

	return stopChan
}
