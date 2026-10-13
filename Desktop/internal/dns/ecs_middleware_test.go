package dns

import (
	"context"
	"net"
	"testing"

	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/miekg/dns"
)

func TestECSMiddleware_StripAndCustom(t *testing.T) {
	// 1. 测试 StripECS
	mgr := NewECSManager(core.ECSConfig{
		Enabled:      true,
		StripECS:     true,
		CustomSubnet: "",
	})
	mw := NewECSMiddleware(mgr)

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	// 附加原始客户端子网
	_ = SetECS(req, "192.0.2.0/24")

	if opt := GetEdns0Option[*dns.EDNS0_SUBNET](req); opt == nil {
		t.Fatalf("expected EDNS0_SUBNET before middleware")
	}

	dctx := NewDNSContext(context.Background(), req, &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1234}, "udp")
	called := false
	err := mw(dctx, func() error {
		called = true
		return nil
	})
	if err != nil || !called {
		t.Fatalf("expected next to run")
	}

	if opt := GetEdns0Option[*dns.EDNS0_SUBNET](req); opt != nil {
		t.Fatalf("expected EDNS0_SUBNET to be stripped, but found: %v", opt)
	}

	// 2. 测试 CustomSubnet 注入
	mgr.Configure(core.ECSConfig{
		Enabled:      true,
		StripECS:     false,
		CustomSubnet: "203.0.113.0/24",
	})

	req2 := new(dns.Msg)
	req2.SetQuestion("example.com.", dns.TypeA)
	dctx2 := NewDNSContext(context.Background(), req2, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}, "udp")

	_ = mw(dctx2, func() error { return nil })

	opt := GetEdns0Option[*dns.EDNS0_SUBNET](req2)
	if opt == nil {
		t.Fatalf("expected EDNS0_SUBNET to be injected")
	}
	if opt.Family != 1 || opt.SourceNetmask != 24 || !opt.Address.Equal(net.ParseIP("203.0.113.0")) {
		t.Fatalf("unexpected injected subnet: %v", opt)
	}
}
