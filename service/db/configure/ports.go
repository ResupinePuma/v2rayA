package configure

type Ports struct {
	Socks5        int     `json:"socks5"`
	Http          int     `json:"http"`
	Socks5WithPac int     `json:"socks5WithPac"`
	HttpWithPac   int     `json:"httpWithPac"`
	Vmess         int     `json:"vmess"`
	Api           ApiPort `json:"api"`
}

type ApiPort struct {
	Port     int      `json:"port"`
	Services []string `json:"services"`
}

func NewPorts() Ports {
	return Ports{
		Socks5:        0,
		Http:          0,
		Socks5WithPac: 20170,
		HttpWithPac:   20171,
		Vmess:         0,
		Api:           ApiPort{Port: 0},
	}
}

func NormalizePorts(p *Ports) {
	if p == nil {
		return
	}
	// Migrate the historical defaults to rule-aware ports on startup:
	// 20170 is SOCKS with rule, 20171 is HTTP with rule.
	if p.Socks5 == 20170 && p.Http == 20171 && p.Socks5WithPac == 0 && p.HttpWithPac == 20172 && p.Vmess == 0 && p.Api.Port == 0 {
		*p = NewPorts()
	}
}
