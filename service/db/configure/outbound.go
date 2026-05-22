package configure

type ObservatoryType string

func (t ObservatoryType) String() string {
	return string(t)
}

const (
	LeastPing  ObservatoryType = "leastping"
	Random     ObservatoryType = "random"
	Health     ObservatoryType = "health"
	RoundRobin ObservatoryType = "roundrobin"
	LeastLoad  ObservatoryType = "leastload"
)

type OutboundSetting struct {
	ProbeURL      string          `json:"probeURL"`
	ProbeInterval string          `json:"probeInterval"`
	Type          ObservatoryType `json:"type"`
}

// DefaultOutboundSetting returns an OutboundSetting with default values.
func DefaultOutboundSetting() OutboundSetting {
	return OutboundSetting{
		ProbeURL:      DefaultProbeURL,
		ProbeInterval: DefaultProbeInterval,
		Type:          ObservatoryType(DefaultOutboundType),
	}
}

func SupportedObservatoryTypes() []ObservatoryType {
	return []ObservatoryType{
		LeastPing,
		Random,
		RoundRobin,
		LeastLoad,
		Health, // alias of leastping for UI
	}
}

func IsSupportedObservatoryType(t ObservatoryType) bool {
	for _, v := range SupportedObservatoryTypes() {
		if v == t {
			return true
		}
	}
	return false
}
