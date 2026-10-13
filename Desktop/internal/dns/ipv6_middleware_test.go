package dns

import (
	"context"
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestIPv6Middleware(t *testing.T) {
	enabled := true
	mw := NewIPv6Middleware(func() bool { return enabled })

	// 1. AAAA 查询拦截并返回权威 NODATA
	req := new(dns.Msg)
	req.SetQuestion("test.example.com.", dns.TypeAAAA)
	dctx := NewDNSContext(context.Background(), req, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5353}, "udp")

	nextCalled := false
	err := mw(dctx, func() error {
		nextCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nextCalled {
		t.Fatal("expected next() not to be called for AAAA when BlockIPv6 is enabled")
	}
	if dctx.Resp == nil {
		t.Fatal("expected synthetic response")
	}
	if dctx.Resp.Rcode != dns.RcodeSuccess {
		t.Fatalf("expected NOERROR (0), got %d", dctx.Resp.Rcode)
	}
	if len(dctx.Resp.Answer) != 0 {
		t.Fatalf("expected empty Answer for NODATA, got %d records", len(dctx.Resp.Answer))
	}
	if len(dctx.Resp.Ns) == 0 {
		t.Fatal("expected SOA record in NS section")
	}
	if src, _ := dctx.Get("source"); src != "block_ipv6" {
		t.Fatalf("expected source=block_ipv6, got %v", src)
	}
	if blocked, _ := dctx.Get("ipv6_blocked"); blocked != true {
		t.Fatalf("expected ipv6_blocked=true, got %v", blocked)
	}

	// 2. A 记录查询不受影响，正常透传至下层
	reqA := new(dns.Msg)
	reqA.SetQuestion("test.example.com.", dns.TypeA)
	dctxA := NewDNSContext(context.Background(), reqA, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5353}, "udp")

	nextCalledA := false
	errA := mw(dctxA, func() error {
		nextCalledA = true
		return nil
	})
	if errA != nil || !nextCalledA {
		t.Fatalf("expected TypeA to pass through to next(), nextCalled=%v, err=%v", nextCalledA, errA)
	}

	// 3. 动态禁用后，AAAA 恢复透传
	enabled = false
	dctxDisabled := NewDNSContext(context.Background(), req, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5353}, "udp")
	nextCalledDisabled := false
	errDisabled := mw(dctxDisabled, func() error {
		nextCalledDisabled = true
		return nil
	})
	if errDisabled != nil || !nextCalledDisabled {
		t.Fatalf("expected AAAA to pass through when disabled, nextCalled=%v, err=%v", nextCalledDisabled, errDisabled)
	}
}
