package cron

import (
	"github.com/pubgo/g/errors"
	"time"
)

// ConstantDelaySchedule represents a simple recurring duty cycle, e.g. "Every 5 minutes".
// It does not support jobs more frequent than once a second.
type ConstantDelaySchedule struct {
	Delay time.Duration
}

// Every returns a crontab Schedule that activates once every duration.
// Delays of less than a second are not supported (will panic).
// Any fields less than a Second are truncated.
func Every(duration time.Duration) ConstantDelaySchedule {
	errors.PanicTT(duration < time.Second, func(err *errors.Err) {
		err.Msg("cron/constantdelay: delays of less than a second are not supported: %s", duration.String())
	})
	return ConstantDelaySchedule{Delay: duration - time.Duration(duration.Nanoseconds())%time.Second}
}

// Next returns the next time this should be run.
// This rounds so that the next activation time will be on the second.
func (schedule ConstantDelaySchedule) Next(t time.Time) time.Time {
	return t.Add(schedule.Delay - time.Duration(t.Nanosecond())*time.Nanosecond)
}
