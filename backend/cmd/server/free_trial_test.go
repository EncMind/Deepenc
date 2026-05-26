package main

import (
	"testing"
	"time"
)

func TestFreeTrialWindowEnsureWindow(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	ft := &FreeTrialWindow{
		Enabled: true,
	}

	if !ft.ensureWindow(now) {
		t.Fatalf("expected ensureWindow to modify the trial window")
	}

	if ft.Enabled {
		t.Fatalf("expected enabled to be cleared after seeding window")
	}

	if ft.StartAt == 0 || ft.EndAt == 0 {
		t.Fatalf("expected start/end timestamps to be set")
	}

	if got, want := ft.EndAt-ft.StartAt, int64(freeTrialDuration/time.Millisecond); got != want {
		t.Fatalf("expected duration %d, got %d", want, got)
	}
}

func TestFreeTrialWindowActive(t *testing.T) {
	now := time.Now()
	ft := FreeTrialWindow{
		StartAt: now.Add(-1 * time.Hour).UnixMilli(),
		EndAt:   now.Add(1 * time.Hour).UnixMilli(),
	}

	if !ft.Active(now) {
		t.Fatalf("expected trial to be active")
	}

	if ft.Active(now.Add(2 * time.Hour)) {
		t.Fatalf("expected trial to be inactive after end")
	}
}
