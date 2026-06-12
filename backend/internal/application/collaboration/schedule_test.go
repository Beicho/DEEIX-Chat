package collaboration

import (
	"testing"
	"time"
)

func TestComputeNextRunAtForDailySchedule(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC)
	next, err := computeNextRunAt("daily", "08:15", 0, "", now)
	if err != nil {
		t.Fatalf("computeNextRunAt() error = %v", err)
	}
	want := time.Date(2026, 6, 13, 8, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestComputeNextRunAtForWeeklySchedule(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC) // Friday
	next, err := computeNextRunAt("weekly", "08:15", int(time.Monday), "", now)
	if err != nil {
		t.Fatalf("computeNextRunAt() error = %v", err)
	}
	want := time.Date(2026, 6, 15, 8, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestComputeNextRunAtForCronSchedule(t *testing.T) {
	now := time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC)
	next, err := computeNextRunAt("cron", "", 0, "15 8 * * *", now)
	if err != nil {
		t.Fatalf("computeNextRunAt() error = %v", err)
	}
	want := time.Date(2026, 6, 13, 8, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %s, want %s", next, want)
	}
}
