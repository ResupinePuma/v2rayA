package configure

import "testing"

func TestIsSupportedObservatoryType(t *testing.T) {
	for _, typ := range SupportedObservatoryTypes() {
		if !IsSupportedObservatoryType(typ) {
			t.Fatalf("expected supported type: %s", typ)
		}
	}
	if IsSupportedObservatoryType(ObservatoryType("unknown")) {
		t.Fatalf("unknown strategy should be unsupported")
	}
}
