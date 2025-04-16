package wgroup

import (
	"sync"
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
