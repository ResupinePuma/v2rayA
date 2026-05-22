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
