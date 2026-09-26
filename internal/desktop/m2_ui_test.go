package desktop

import (
	"testing"
	"time"
)

func TestNVDWindowAllowsCurrentUTCDayWithoutFutureRequests(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 30, 0, 0, time.UTC)
	start, end, err := nvdWindow("2026-09-25", "2026-09-26", now)
	if err != nil || start.Day() != 25 || !end.Equal(now) {
		t.Fatalf("current day: %s %s %v", start, end, err)
	}
	if _, _, err := nvdWindow("2026-09-25", "2026-09-27", now); err == nil {
		t.Fatal("future source window accepted")
	}
}
