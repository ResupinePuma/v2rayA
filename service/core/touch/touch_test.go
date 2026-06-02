package touch

import "testing"

func TestIsDeadLatency(t *testing.T) {
	cases := []struct {
		latency string
		dead    bool
	}{
		{"", false},
		{"120ms", false},
		{"TIMEOUT", true},
		{"NOT STABLE", true},
	}
	for _, c := range cases {
		if got := isDeadLatency(c.latency); got != c.dead {
			t.Fatalf("isDeadLatency(%q)=%v, want %v", c.latency, got, c.dead)
		}
	}
}

func TestNormalizeSubscriptionOutbounds(t *testing.T) {
	got := normalizeSubscriptionOutbounds([]string{"", "proxy", "video", "proxy"})
	if len(got) != 2 || got[0] != "proxy" || got[1] != "video" {
		t.Fatalf("normalizeSubscriptionOutbounds()=%#v", got)
	}
	got = normalizeSubscriptionOutbounds(nil)
	if len(got) != 1 || got[0] != "proxy" {
		t.Fatalf("normalizeSubscriptionOutbounds(nil)=%#v", got)
	}
}
