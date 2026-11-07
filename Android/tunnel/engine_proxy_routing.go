// engine_proxy_routing.go implements per-application outbound proxy routing.
//
// Behavior:
// - Only the UIDs selected in the outbound-proxy app list are dialed through
//   the configured SOCKS5 / HTTP CONNECT proxy. Every other UID keeps using
//   the direct outbound, so unchecked apps never touch the proxy.
// - Kotlin resolves package names to UIDs and pushes the CSV through
//   SetProxyAppUIDs, which also allows runtime updates without restarting
//   the tunnel.
// - DNS upstream queries stay on the direct outbound on purpose: the resolver
//   answers for every app, so proxying it would route unselected apps' DNS
//   through the proxy as well.

package tunnel

import (
	"sort"
	"strconv"
	"strings"
)

// SetProxyAppUIDs selects which application UIDs are routed through the
// outbound proxy. An empty string disables proxying for every app.
func (e *Engine) SetProxyAppUIDs(uidsCsv string) {
	next := make(map[int]struct{})
	uids := make([]int, 0, 4)
	for _, raw := range strings.Split(uidsCsv, ",") {
		uid, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || uid <= 0 {
			continue
		}
		if _, exists := next[uid]; exists {
			continue
		}
		next[uid] = struct{}{}
		uids = append(uids, uid)
	}
	sort.Ints(uids)

	e.proxyUIDsMu.Lock()
	e.proxyUIDs = next
	e.proxyUIDsMu.Unlock()

	logf("SetProxyAppUIDs: %d app UID(s) routed via outbound proxy %v", len(next), uids)
}

// isProxyUID reports whether flows owned by uid must use the outbound proxy.
func (e *Engine) isProxyUID(uid int) bool {
	if uid <= 0 || uid == UIDUnknown {
		return false
	}
	e.proxyUIDsMu.RLock()
	_, proxied := e.proxyUIDs[uid]
	e.proxyUIDsMu.RUnlock()
	return proxied
}

// outboundForUID picks the transport used to dial on behalf of uid: the
// configured proxy for selected apps, the direct outbound for everything else.
func (e *Engine) outboundForUID(uid int) flowOutbound {
	if e.isProxyUID(uid) && e.flowOutbound != nil {
		return e.flowOutbound
	}
	if e.directOutbound != nil {
		return e.directOutbound
	}
	return newFlowOutbound(outboundProxyConfig{}, e.protectFn, nil)
}
