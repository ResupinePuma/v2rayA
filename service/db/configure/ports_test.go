package configure

import "testing"

func TestNewPortsDefaultToRulePorts(t *testing.T) {
	p := NewPorts()
	if p.Socks5 != 0 || p.Http != 0 || p.Socks5WithPac != 20170 || p.HttpWithPac != 20171 {
		t.Fatalf("unexpected default ports: %#v", p)
	}
}

func TestNormalizePortsMigratesLegacyDefaults(t *testing.T) {
	p := &Ports{Socks5: 20170, Http: 20171, Socks5WithPac: 0, HttpWithPac: 20172}
	NormalizePorts(p)
	if p.Socks5 != 0 || p.Http != 0 || p.Socks5WithPac != 20170 || p.HttpWithPac != 20171 {
		t.Fatalf("legacy ports were not migrated to rule ports: %#v", p)
	}
}
