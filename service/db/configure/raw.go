package configure

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
	"github.com/v2rayA/v2rayA/core/serverObj"
	"github.com/v2rayA/v2rayA/pkg/util/log"
)

type ServerRaw struct {
	ServerObj serverObj.ServerObj `json:"serverObj"`
	Latency   string              `json:"latency"`
}

type SubscriptionRaw struct {
	Remarks    string      `json:"remarks,omitempty"`
	Address    string      `json:"address"`
	Status     string      `json:"status"` //update time, error info, etc.
	Servers    []ServerRaw `json:"servers"`
	Info       string      `json:"info"` // maybe include some info from provider
	AutoSelect bool        `json:"autoSelect"`
}

func Bytes2SubscriptionRaw(b []byte) (*SubscriptionRaw, error) {
	var s struct {
		Remarks    string `json:"remarks,omitempty"`
		Address    string `json:"address"`
		Status     string `json:"status"`
		Info       string `json:"info"`
		AutoSelect bool   `json:"autoSelect"`
	}
	if err := jsoniter.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	sub := &SubscriptionRaw{
		Remarks:    s.Remarks,
		Address:    s.Address,
		Status:     s.Status,
		Info:       s.Info,
		AutoSelect: s.AutoSelect,
	}
	rawList := gjson.GetBytes(b, "servers").Array()
	sub.Servers = make([]ServerRaw, 0, len(rawList))
	for _, raw := range rawList {
		rawBytes := []byte(raw.Raw)
		if wrapped := raw.Get("Raw").String(); wrapped != "" {
			rawBytes = []byte(wrapped)
		}
		sr, err := Bytes2ServerRaw(rawBytes)
		if err != nil {
			return nil, err
		}
		sub.Servers = append(sub.Servers, *sr)
	}
	return sub, nil
}

func Bytes2ServerRaw(b []byte) (*ServerRaw, error) {
	var s ServerRaw
	var obj serverObj.ServerObj
	protocol := gjson.GetBytes(b, "serverObj.protocol").String()
	if protocol == "" {
		log.Warn("empty protocol, fallback to vmess: %v", gjson.GetBytes(b, "serverObj.ps").String())
		protocol = "vmess"
	}
	obj, err := serverObj.New(protocol)
	if err != nil {
		return nil, err
	}
	s.ServerObj = obj
	if err := jsoniter.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
