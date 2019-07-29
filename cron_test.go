package cron

import (
	"fmt"
	"github.com/pubgo/errors"
	"sync"
	"testing"
	"time"
)

// Many tests schedule a job for every second, and then wait at most a second
// for it to run.  This amount is just slightly larger than 1 second to
// compensate for a few milliseconds of runtime.
const ONE_SECOND = 1*time.Second + 10*time.Millisecond

// Start and stop cron with no entries.
func TestNoEntries(t *testing.T) {
	defer errors.Assert()

	cron := New()
	cron.Start()

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-stop(cron):
	}
}

// Start, stop, then add an entry. Verify entry doesn't run.
func TestStopCausesJobsToNotRun(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.Start()
	cron.Stop()
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) {
		wg.Done()
	}, "test1")

	select {
	case <-time.After(ONE_SECOND):
		// No job ran!
	case <-wait(wg):
		t.FailNow()
	}
}

// Add a job, start cron, expect it runs.
func TestAddBeforeRunning(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test2")
	cron.Start()
	defer cron.Stop()

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

// Start cron, add a job, expect it runs.
func TestAddWhileRunning(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.Start()
	defer cron.Stop()
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test3")

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

// Test timing with Entries.
func TestSnapshotEntries(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddFunc("@every 2s", func(t time.Time, name string) {
		fmt.Println(t, name)
		wg.Done()
	}, "test4")
	cron.Start()
	defer cron.Stop()

	// Cron should fire in 2 seconds. After 1 second, call Entries.
	select {
	case <-time.After(ONE_SECOND):
		cron.Entries()
	}

	// Even though Entries was called, the cron should fire at the 2 second mark.
	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}

}

// Test that the entries are correctly sorted.
// Add a bunch of long-in-the-future entries, and an immediate entry, and ensure
// that the immediate entry runs immediately.
// Also: Test that multiple jobs run in the same instant.
func TestMultipleEntries(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	cron := New()
	cron.AddFunc("0 0 0 1 1 ?", func(t time.Time, name string) {}, "test5")
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test6")
	cron.AddFunc("0 0 0 31 12 ?", func(t time.Time, name string) {}, "test7")
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test8")

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

// Test running the same job twice.
func TestRunningJobTwice(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	cron := New()
	cron.AddFunc("0 0 0 1 1 ?", func(t time.Time, name string) {}, "test9")
	cron.AddFunc("0 0 0 31 12 ?", func(t time.Time, name string) {}, "test10")
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test11")

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(2 * ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

func TestRunningMultipleSchedules(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	cron := New()
	cron.AddFunc("0 0 0 1 1 ?", func(t time.Time, name string) {}, "test12")
	cron.AddFunc("0 0 0 31 12 ?", func(t time.Time, name string) {}, "test13")
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test14")
	cron.Schedule(Every(time.Minute), FuncJob(func(t time.Time, name string) {}), "test15")
	cron.Schedule(Every(time.Second), FuncJob(func(t time.Time, name string) { wg.Done() }), "test16")
	cron.Schedule(Every(time.Hour), FuncJob(func(t time.Time, name string) {}), "test17")

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(2 * ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

// Test that the cron is run in the local time zone (as opposed to UTC).
func TestLocalTimezone(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	now := time.Now().Local()
	spec := fmt.Sprintf("%d %d %d %d %d ?",
		now.Second()+1, now.Minute(), now.Hour(), now.Day(), now.Month())

	cron := New()
	cron.AddFunc(spec, func(t time.Time, name string) { wg.Done() }, "test18")
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

type testJob struct {
	wg   *sync.WaitGroup
	name string
}

func (t testJob) Run(tt time.Time, name string) {
	t.wg.Done()
}

// Simple test using Runnables.
func TestJob(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddJob("0 0 0 30 Feb ?", testJob{wg, "job0"}, "test19")
	cron.AddJob("0 0 0 1 1 ?", testJob{wg, "job1"}, "test20")
	cron.AddJob("* * * * * ?", testJob{wg, "job2"}, "test21")
	cron.AddJob("1 0 0 1 1 ?", testJob{wg, "job3"}, "test22")
	cron.Schedule(Every(5*time.Second+5*time.Nanosecond), testJob{wg, "job4"}, "test23")
	cron.Schedule(Every(5*time.Minute), testJob{wg, "job5"}, "test24")

	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}

	// Ensure the entries are in the right order.
	expecteds := []string{"job2", "job4", "job5", "job1", "job3", "job0"}

	var actuals []string
	for _, entry := range cron.Entries() {
		actuals = append(actuals, entry.Job.(testJob).name)
	}

	for i, expected := range expecteds {
		if actuals[i] != expected {
			t.Errorf("Jobs not in the right order.  (expected) %s != %s (actual)", expecteds, actuals)
			t.FailNow()
		}
	}
}

// Add a job, start cron, remove the job, expect it to have not run
func TestAddBeforeRunningThenRemoveWhileRunning(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test25")
	cron.Start()
	defer cron.Stop()
	cron.RemoveJob("test25")

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(ONE_SECOND):
	case <-wait(wg):
		t.FailNow()
	}
}

// Add a job, remove the job, start cron, expect it to have not run
func TestAddBeforeRunningThenRemoveBeforeRunning(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddFunc("* * * * * ?", func(t time.Time, name string) { wg.Done() }, "test26")
	cron.RemoveJob("test26")
	cron.Start()
	defer cron.Stop()

	// Give cron 2 seconds to run our job (which is always activated).
	select {
	case <-time.After(ONE_SECOND):
	case <-wait(wg):
		t.FailNow()
	}
}

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}

func stop(cron *Cron) chan bool {
	ch := make(chan bool)
	go func() {
		cron.Stop()
		ch <- true
	}()
	return ch
}

func TestMuti(t *testing.T) {
	defer errors.Assert()

	wg := &sync.WaitGroup{}
	cron := New()
	_func := func(t time.Time, name string) {
		go func() {
			fmt.Println(t.UnixNano(), name)
			wg.Done()
		}()
	}

	for i := 0; i < 100000; i++ {
		fmt.Println(i)
		wg.Add(1)
		cron.AddFunc(fmt.Sprintf("test%d", i), "*/2 * * * * *", _func)
	}

	cron.Start()

	cron.Each(func(e *Entry) {
		fmt.Println(e.Name)
	})

	defer cron.Stop()

	wg.Wait()
}
