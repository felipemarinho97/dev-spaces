package util

import (
	"time"

	"github.com/schollz/progressbar/v3"
)

type UnknownBar struct {
	Bar  *progressbar.ProgressBar
	stop bool
}

func NewUnknownBar(description string) *UnknownBar {
	b := progressbar.Default(-1, description)
	return &UnknownBar{
		Bar: b,
	}
}

func (u *UnknownBar) Start() {
	go func() {
		for {
			if u.stop {
				break
			}
			u.Bar.Add(1)
			time.Sleep(40 * time.Millisecond)
		}
	}()
}

func (u *UnknownBar) Stop() {
	u.stop = true
	u.Bar.Clear()
}
