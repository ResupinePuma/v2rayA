package configure

import "testing"

func TestAutoUpdateDurationMinutesPreferred(t *testing.T) {
	s := &Setting{GFWListAutoUpdateIntervalHour: 2, GFWListAutoUpdateIntervalMinute: 15, SubscriptionAutoUpdateIntervalHour: 3, SubscriptionAutoUpdateIntervalMinute: 20}
	if got := s.GFWListAutoUpdateDuration(); got != 15 {
		t.Fatalf("GFWListAutoUpdateDuration()=%d, want 15", got)
	}
	if got := s.SubscriptionAutoUpdateDuration(); got != 20 {
		t.Fatalf("SubscriptionAutoUpdateDuration()=%d, want 20", got)
	}
}

func TestAutoUpdateDurationFallbackHours(t *testing.T) {
	s := &Setting{GFWListAutoUpdateIntervalHour: 2, SubscriptionAutoUpdateIntervalHour: 3}
	if got := s.GFWListAutoUpdateDuration(); got != 120 {
		t.Fatalf("GFWListAutoUpdateDuration()=%d, want 120", got)
	}
	if got := s.SubscriptionAutoUpdateDuration(); got != 180 {
		t.Fatalf("SubscriptionAutoUpdateDuration()=%d, want 180", got)
	}
}
