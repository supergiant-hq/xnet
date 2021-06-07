// Custom Ticker
package util

import (
	"time"
)

// Ticker is a wrapper over the native time.Ticker implementation
type Ticker struct {
	f      func()
	d      time.Duration
	ticker *time.Ticker
	exit   chan bool
}

type TickerFunction func()

// Creates a new ticker which runs every d time.Duration and executes the provided f TickerFunction
func NewTicker(d time.Duration, f TickerFunction) *Ticker {
	return &Ticker{
		f: f,
		d: d,
	}
}

// Start the ticker
func (t *Ticker) Start() {
	t.ticker = time.NewTicker(t.d)
	t.exit = make(chan bool, 1)

	go func() {
		defer func() {
			recover()
		}()
		defer t.ticker.Stop()
		for {
			select {
			case <-t.ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Stop()
						}
					}()

					t.f()
				}()
			case <-t.exit:
				return
			}
		}
	}()
}

// Stop the ticker
func (t *Ticker) Stop() {
	defer func() {
		recover()
	}()

	if t.ticker == nil {
		return
	}

	select {
	case t.exit <- true:
	default:
	}

	t.ticker.Stop()
	t.ticker = nil
}
