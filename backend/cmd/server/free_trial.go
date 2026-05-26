package main

import "time"

const freeTrialDuration = 30 * 24 * time.Hour

// FreeTrialWindow holds metadata about a complimentary access window.
type FreeTrialWindow struct {
	Enabled bool  `json:"enabled"`
	StartAt int64 `json:"startAt,omitempty"`
	EndAt   int64 `json:"endAt,omitempty"`
}

func (ft *FreeTrialWindow) ensureWindow(now time.Time) bool {
	if ft == nil || !ft.Enabled {
		return false
	}

	durationMillis := int64(freeTrialDuration / time.Millisecond)
	ft.StartAt = now.UnixMilli()
	ft.EndAt = ft.StartAt + durationMillis
	ft.Enabled = false
	return true
}

func (ft *FreeTrialWindow) clearWindow() bool {
	if ft == nil {
		return false
	}

	if ft.Enabled || ft.StartAt != 0 || ft.EndAt != 0 {
		ft.Enabled = false
		ft.StartAt = 0
		ft.EndAt = 0
		return true
	}
	return false
}

// Active reports whether the free trial window covers the supplied time.
func (ft FreeTrialWindow) Active(now time.Time) bool {
	if ft.StartAt == 0 || ft.EndAt == 0 {
		return false
	}
	nowMillis := now.UnixMilli()
	return nowMillis >= ft.StartAt && nowMillis < ft.EndAt
}
