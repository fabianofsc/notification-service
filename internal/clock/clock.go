package clock

import "time"

type Real struct{}

func (Real) Now() time.Time { return time.Now() }

type Fake struct {
	now time.Time
}

func NewFake(start time.Time) *Fake {
	return &Fake{now: start}
}

func (c *Fake) Now() time.Time { return c.now }

func (c *Fake) Set(t time.Time) { c.now = t }

func (c *Fake) Advance(d time.Duration) { c.now = c.now.Add(d) }