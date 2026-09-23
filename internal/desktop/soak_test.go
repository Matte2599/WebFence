package desktop

import (
	"testing"
	"time"
)

func TestSoakDurationBounds(t *testing.T) {
	for _, value := range []string{"", "-30m", "0s", "9s", "61m", "NaN", "999999999999999999h"} {
		if _, ok := parseSoakArgs([]string{"webfence", "--soak-test=" + value}); ok {
			t.Fatalf("accepted invalid trial duration %q", value)
		}
	}
	for _, duration := range []time.Duration{10 * time.Second, 30 * time.Minute, time.Hour} {
		got, ok := parseSoakArgs([]string{"webfence", "--soak-test=" + duration.String()})
		if !ok || got != duration {
			t.Fatalf("duration %s: got %s, %t", duration, got, ok)
		}
	}
}
