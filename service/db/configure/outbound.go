package configure

import (
	"strings"

	"github.com/v2rayA/v2rayA/conf"
)

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
	typ := ObservatoryType(DefaultOutboundType)
	if envType := strings.ToLower(strings.TrimSpace(conf.GetEnvironmentConfig().OutboundType)); envType != "" {
		candidate := ObservatoryType(envType)
		if IsSupportedObservatoryType(candidate) {
			typ = candidate
		}
	}
	return OutboundSetting{
		ProbeURL:      DefaultProbeURL,
		ProbeInterval: DefaultProbeInterval,
		Type:          typ,
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
