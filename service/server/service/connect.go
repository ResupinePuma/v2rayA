package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/v2rayA/v2rayA/core/ipforward"
	"github.com/v2rayA/v2rayA/core/v2ray"
	"github.com/v2rayA/v2rayA/core/v2ray/asset"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/pkg/util/log"
)

var V2OnlyFeatureError = fmt.Errorf("v2fly/v2ray-core only feature")

func StopV2ray() (err error) {
	v2ray.ProcessManager.Stop(true)
	return nil
}
func StartV2ray() (err error) {
	if err = checkSupport(nil); err != nil {
		return err
	}
	//configure the ip forward
	setting := GetSetting()
	if setting.IpForward != ipforward.IsIpForwardOn() {
		e := ipforward.WriteIpForward(setting.IpForward)
		if e != nil {
			log.Warn("Connect: %v", e)
		}
	}
	if css := configure.GetConnectedServers(); css.Len() == 0 {
		return fmt.Errorf("failed: no server is selected. please select at least one server")
	} else {
		filtered := make([]configure.Which, 0, css.Len())
		for _, wt := range css.Get() {
			if !isValidWhichRange(*wt) {
				log.Warn("StartV2ray: auto-excluding out-of-range connection: type=%s id=%d sub=%d outbound=%s", wt.TYPE, wt.ID, wt.Sub, wt.Outbound)
				continue
			}
			supported, e := IsSupported(*wt)
			if e != nil {
				log.Warn("StartV2ray: auto-excluding invalid connection while support-checking: type=%s id=%d sub=%d outbound=%s err=%v", wt.TYPE, wt.ID, wt.Sub, wt.Outbound, e)
				continue
			}
			if !supported {
				log.Warn("StartV2ray: auto-excluding unsupported server: type=%s id=%d sub=%d outbound=%s", wt.TYPE, wt.ID, wt.Sub, wt.Outbound)
				continue
			}
			filtered = append(filtered, *wt)
		}
		if len(filtered) != css.Len() {
			for _, out := range configure.GetOutbounds() {
				if e := configure.ClearConnects(out); e != nil {
					return fmt.Errorf("failed to clear outbound %q while removing unsupported servers: %w", out, e)
				}
			}
			for _, wt := range filtered {
				if e := configure.AddConnect(wt); e != nil {
					return fmt.Errorf("failed to restore supported connection after filtering: %w", e)
				}
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("failed: all selected servers are unsupported")
		}
	}
	return v2ray.UpdateV2RayConfig()
}

func isValidWhichRange(wt configure.Which) bool {
	if wt.ID <= 0 {
		return false
	}
	switch wt.TYPE {
	case configure.ServerType:
		return wt.ID <= configure.GetLenServers()
	case configure.SubscriptionServerType:
		if wt.Sub < 0 || wt.Sub >= configure.GetLenSubscriptions() {
			return false
		}
		return wt.ID <= configure.GetLenSubscriptionServers(wt.Sub)
	default:
		return false
	}
}

func Disconnect(which configure.Which, clearOutbound bool) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to disconnect: %w", err)
		}
	}()
	lastConnected := configure.GetConnectedServersByOutbound(which.Outbound)
	if clearOutbound {
		err = configure.ClearConnects(which.Outbound)
	} else {
		err = configure.RemoveConnect(which)
	}
	if err != nil {
		return
	}
	//update the v2ray config and restart v2ray
	if v2ray.ProcessManager.Running() {
		defer func() {
			if err != nil && lastConnected != nil && v2ray.ProcessManager.Running() {
				_ = configure.OverwriteConnects(lastConnected)
				_ = v2ray.UpdateV2RayConfig()
			}
		}()
		if err = v2ray.UpdateV2RayConfig(); err != nil {
			return
		}
	}
	return
}

func checkAssetsExist(setting *configure.Setting) error {
	if !asset.DoesV2rayAssetExist("geoip.dat") || !asset.DoesV2rayAssetExist("geosite.dat") {
		return fmt.Errorf("geoip.dat or geosite.dat file does not exists")
	}
	if setting.RulePortMode == configure.GfwlistMode || setting.Transparent == configure.TransparentGfwlist {
		if !asset.DoesV2rayAssetExist("LoyalsoldierSite.dat") {
			return fmt.Errorf("GFWList file does not exists. Try updating GFWList please")
		}
	}
	return nil
}

func checkSupport(toAppend []*configure.Which) (err error) {
	setting := GetSetting()
	if err = checkAssetsExist(setting); err != nil {
		return err
	}
	// Both v2ray-core and xray-core support load balancing now
	// No need to restrict multiple servers for any core type
	return nil
}

func Connect(which *configure.Which) (err error) {
	log.Trace("Connect: begin")
	defer log.Trace("Connect: done")
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to connect: %w", err)
		}
	}()
	if which == nil {
		return fmt.Errorf("which can not be nil")
	}
	setting := GetSetting()
	if err = checkSupport([]*configure.Which{which}); err != nil {
		if !errors.Is(err, V2OnlyFeatureError) {
			return err
		}
		if err = configure.ClearConnects(which.Outbound); err != nil {
			return err
		}
	}
	//configure the ip forward
	if setting.IpForward != ipforward.IsIpForwardOn() {
		e := ipforward.WriteIpForward(setting.IpForward)
		if e != nil {
			log.Warn("Connect: %v", e)
		}
	}
	//locate server
	currentConnected := configure.GetConnectedServersByOutbound(which.Outbound)
	defer func() {
		// if error occurs, restore the result of connecting
		if err != nil && currentConnected != nil && v2ray.ProcessManager.Running() {
			_ = configure.OverwriteConnects(currentConnected)
			_ = v2ray.UpdateV2RayConfig()
		}
	}()
	//save the result of connecting to database
	if err = configure.AddConnect(*which); err != nil {
		return
	}
	//update the v2ray config and start/restart v2ray
	if v2ray.ProcessManager.Running() {
		if err = v2ray.UpdateV2RayConfig(); err != nil {
			return
		}
	}
	return
}

func isUnexpectedTransportErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unexpected transport type")
}

type removedWhichIndexes struct {
	servers map[int]struct{}
	subs    map[int]map[int]struct{}
}

func newRemovedWhichIndexes() removedWhichIndexes {
	return removedWhichIndexes{servers: make(map[int]struct{}), subs: make(map[int]map[int]struct{})}
}

func (r removedWhichIndexes) add(wt configure.Which) {
	switch wt.TYPE {
	case configure.ServerType:
		r.servers[wt.ID] = struct{}{}
	case configure.SubscriptionServerType:
		if _, ok := r.subs[wt.Sub]; !ok {
			r.subs[wt.Sub] = make(map[int]struct{})
		}
		r.subs[wt.Sub][wt.ID] = struct{}{}
	}
}

func (r removedWhichIndexes) contains(wt configure.Which) bool {
	switch wt.TYPE {
	case configure.ServerType:
		_, ok := r.servers[wt.ID]
		return ok
	case configure.SubscriptionServerType:
		ids, ok := r.subs[wt.Sub]
		if !ok {
			return false
		}
		_, ok = ids[wt.ID]
		return ok
	default:
		return false
	}
}

func (r removedWhichIndexes) adjust(wt configure.Which) (configure.Which, bool) {
	if r.contains(wt) {
		return wt, false
	}
	removedBefore := 0
	switch wt.TYPE {
	case configure.ServerType:
		for id := range r.servers {
			if id < wt.ID {
				removedBefore++
			}
		}
	case configure.SubscriptionServerType:
		for id := range r.subs[wt.Sub] {
			if id < wt.ID {
				removedBefore++
			}
		}
	}
	wt.ID -= removedBefore
	return wt, wt.ID > 0
}

func removeUnexpectedTransportNodes(toRemove []configure.Which) (removedWhichIndexes, error) {
	removed := newRemovedWhichIndexes()
	serverIDs := make(map[int]struct{})
	subIDs := make(map[int]map[int]struct{})
	for _, wt := range toRemove {
		removed.add(wt)
		switch wt.TYPE {
		case configure.ServerType:
			serverIDs[wt.ID-1] = struct{}{}
		case configure.SubscriptionServerType:
			if _, ok := subIDs[wt.Sub]; !ok {
				subIDs[wt.Sub] = make(map[int]struct{})
			}
			subIDs[wt.Sub][wt.ID-1] = struct{}{}
		}
	}
	if len(serverIDs) > 0 {
		indexes := make([]int, 0, len(serverIDs))
		for idx := range serverIDs {
			indexes = append(indexes, idx)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(indexes)))
		if err := configure.RemoveServers(indexes); err != nil {
			return removed, err
		}
	}
	for sub, ids := range subIDs {
		raw := configure.GetSubscription(sub)
		if raw == nil {
			continue
		}
		indexes := make([]int, 0, len(ids))
		for idx := range ids {
			indexes = append(indexes, idx)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(indexes)))
		for _, idx := range indexes {
			if idx < 0 || idx >= len(raw.Servers) {
				continue
			}
			raw.Servers = append(raw.Servers[:idx], raw.Servers[idx+1:]...)
		}
		if err := configure.SetSubscription(sub, raw); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

func pruneUnexpectedTransportConnections(touches []configure.Which) ([]configure.Which, int, error) {
	toRemove := make([]configure.Which, 0)
	seenBad := make(map[string]struct{})
	for _, wt := range touches {
		_, err := IsSupported(wt)
		if !isUnexpectedTransportErr(err) {
			continue
		}
		key := fmt.Sprintf("%s/%d/%d", wt.TYPE, wt.ID, wt.Sub)
		if _, ok := seenBad[key]; ok {
			continue
		}
		seenBad[key] = struct{}{}
		toRemove = append(toRemove, wt)
	}
	if len(toRemove) == 0 {
		return touches, 0, nil
	}
	removed, err := removeUnexpectedTransportNodes(toRemove)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]configure.Which, 0, len(touches)-len(toRemove))
	for _, wt := range touches {
		adjusted, ok := removed.adjust(wt)
		if !ok {
			continue
		}
		filtered = append(filtered, adjusted)
	}
	return filtered, len(toRemove), nil
}

// ReplaceOutboundConnections atomically replaces members of one outbound group.
// It updates v2ray config once after DB changes, and rolls back on failure.
func ReplaceOutboundConnections(outbound string, touches []configure.Which) (err error) {
	log.Trace("ReplaceOutboundConnections: begin")
	defer log.Trace("ReplaceOutboundConnections: done")
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to replace outbound connections: %w", err)
		}
	}()

	if outbound == "" {
		outbound = "proxy"
	}

	// Normalize outbound and deduplicate touches.
	normalized := make([]configure.Which, 0, len(touches))
	seen := make(map[string]struct{})
	for i, wt := range touches {
		if wt.ID <= 0 {
			return fmt.Errorf("invalid touch id at index %d: %d", i, wt.ID)
		}
		switch wt.TYPE {
		case configure.ServerType:
			wt.Sub = 0
			if wt.ID > configure.GetLenServers() {
				return fmt.Errorf("invalid server id at index %d: %d", i, wt.ID)
			}
		case configure.SubscriptionServerType:
			if wt.Sub < 0 {
				return fmt.Errorf("invalid subscription index at index %d: %d", i, wt.Sub)
			}
			if wt.Sub >= configure.GetLenSubscriptions() || wt.ID > configure.GetLenSubscriptionServers(wt.Sub) {
				return fmt.Errorf("invalid subscription server range at index %d: sub=%d id=%d", i, wt.Sub, wt.ID)
			}
		default:
			return fmt.Errorf("invalid touch type at index %d: %q", i, wt.TYPE)
		}
		wt.Outbound = outbound
		key := fmt.Sprintf("%s/%d/%d", wt.TYPE, wt.ID, wt.Sub)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, wt)
	}

	var removedUnsupported int
	normalized, removedUnsupported, err = pruneUnexpectedTransportConnections(normalized)
	if err != nil {
		return err
	}
	if removedUnsupported > 0 {
		log.Warn("ReplaceOutboundConnections: removed %d node(s) with unexpected transport type from outbound %s", removedUnsupported, outbound)
	}
	if len(normalized) == 0 {
		return fmt.Errorf("all selected servers were removed due to unexpected transport type")
	}

	backup := configure.GetConnectedServersByOutbound(outbound)
	restore := func() {
		if backup != nil {
			_ = configure.OverwriteConnects(backup)
		} else {
			_ = configure.ClearConnects(outbound)
		}
	}

	if err = configure.ReplaceConnects(outbound, normalized); err != nil {
		return err
	}

	if v2ray.ProcessManager.Running() {
		if err = v2ray.UpdateV2RayConfig(); err != nil {
			restore()
			_ = v2ray.UpdateV2RayConfig()
			return err
		}
	}

	return nil
}
