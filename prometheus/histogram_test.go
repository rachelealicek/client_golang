package prometheus

import (
	"sync"
	"testing"
	"time"
)

func TestHistogramVecDeleteRace(t *testing.T) {
	hVec := NewHistogramVec(HistogramOpts{
		Name: "test_histogram",
		Help: "helper",
	}, []string{"label"})

	labels := []string{"value"}
	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Goroutine 1: Constantly observing
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				hVec.WithLabelValues(labels...).Observe(1.0)
			}
		}
	}()

	// Goroutine 2: Constantly deleting
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				hVec.DeleteLabelValues(labels...)
			}
		}
	}()

	// Run the stress test for a short duration
	time.Sleep(2 * time.Second)
	close(stop)
	wg.Wait()
}
