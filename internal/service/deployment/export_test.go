package deployment

import "time"

// SetPollIntervalForTest overrides the poll interval for unit tests.
// Must be called from TestMain before any tests run.
func SetPollIntervalForTest(d time.Duration) {
	pollInterval = d
}
