package touch

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/v2rayA/v2rayA/db/configure"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

/*
Touch是树型结构的前后端通信形式，其结构设计和前端统一。
*/
type SubscriptionStatus string
type Touch struct {
	Servers          []Server           `json:"servers"`
	Subscriptions    []Subscription     `json:"subscriptions"`
	ConnectedServers []*configure.Which `json:"connectedServer"` //冗余一个信息，方便查找
}
type Server struct {
	ID          int                 `json:"id"`
	TYPE        configure.TouchType `json:"_type"`
	Name        string              `json:"name"`
	Address     string              `json:"address"`
	Net         string              `json:"net"`
	PingLatency string              `json:"pingLatency"`
	IsDead      bool                `json:"isDead"`
}
type Subscription struct {
	Remarks    string              `json:"remarks,omitempty"`
	ID         int                 `json:"id"`
	TYPE       configure.TouchType `json:"_type"`
	Host       string              `json:"host"`
	Address    string              `json:"address"`
	Status     SubscriptionStatus  `json:"status"`
	Info       string              `json:"info"`
	Servers    []Server            `json:"servers"`
	AutoSelect bool                `json:"autoSelect"`
	Outbounds  []string            `json:"outbounds"`
}

func NewUpdateStatus() SubscriptionStatus {
	return SubscriptionStatus(time.Now().Local().Format("2006-1-2 15:04:05"))
}

/* Mapping []TouchServerRaw to []Server */
func serverRawsToServers(rss []configure.ServerRaw) (ts []Server) {
	ts = make([]Server, len(rss))
	for i, v := range rss {
		var address string
		if v.ServerObj.GetPort() == 0 {
			address = v.ServerObj.GetHostname()
		} else {
			address = net.JoinHostPort(v.ServerObj.GetHostname(), strconv.Itoa(v.ServerObj.GetPort()))
		}
		ts[i] = Server{
			ID:          i + 1,
			Name:        v.ServerObj.GetName(),
			Address:     address,
			Net:         v.ServerObj.ProtoToShow(),
			PingLatency: v.Latency,
			IsDead:      isDeadLatency(v.Latency),
		}
	}
	return
}

func isDeadLatency(latency string) bool {
	if latency == "" {
		return false
	}
	if strings.HasSuffix(latency, "ms") {
		return false
	}
	return true
}

func parseSubscriptionHost(address string) string {
	if strings.TrimSpace(address) == "" {
		return ""
	}
	u, err := url.Parse(address)
	if err == nil && u != nil && u.Host != "" {
		return u.Host
	}
	// it may be OOCv1
	tmp := make(map[string]string)
	if err = jsoniter.Unmarshal([]byte(address), &tmp); err == nil {
		if baseURL := strings.TrimSpace(tmp["baseUrl"]); baseURL != "" {
			u, err = url.Parse(baseURL)
			if err == nil && u != nil {
				return u.Host
			}
		}
	}
	// Some subscription "address" values can be proxy links (not URL host-style),
	// keep Touch generation resilient and avoid noisy warnings here.
	return ""
}

func normalizeSubscriptionOutbounds(outbounds []string) []string {
	seen := make(map[string]struct{})
	normalized := make([]string, 0, len(outbounds))
	for _, outbound := range outbounds {
		outbound = strings.TrimSpace(outbound)
		if outbound == "" {
			continue
		}
		if _, ok := seen[outbound]; ok {
			continue
		}
		seen[outbound] = struct{}{}
		normalized = append(normalized, outbound)
	}
	if len(normalized) == 0 {
		normalized = append(normalized, "proxy")
	}
	return normalized
}

// GenerateTouch generates a touch from database
func GenerateTouch() (t Touch) {
	t.Servers = serverRawsToServers(configure.GetServers())
	subscriptions := configure.GetSubscriptions()
	t.Subscriptions = make([]Subscription, len(subscriptions))
	for i, v := range subscriptions {
		t.Subscriptions[i] = Subscription{
			Remarks:    v.Remarks,
			ID:         i + 1,
			Host:       parseSubscriptionHost(v.Address),
			Address:    v.Address,
			Status:     SubscriptionStatus(v.Status),
			Servers:    serverRawsToServers(v.Servers),
			Info:       v.Info,
			AutoSelect: v.AutoSelect,
			Outbounds:  normalizeSubscriptionOutbounds(v.Outbounds),
		}
	}
	t.ConnectedServers = configure.GetConnectedServers().Get()
	//补充TYPE
	for i := range t.Subscriptions {
		t.Subscriptions[i].TYPE = configure.SubscriptionType
		for j := range t.Subscriptions[i].Servers {
			t.Subscriptions[i].Servers[j].TYPE = configure.SubscriptionServerType
		}
	}
	for i := range t.Servers {
		t.Servers[i].TYPE = configure.ServerType
	}
	return
}
