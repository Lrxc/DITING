package dns

import (
	"context"
	"net"
	"testing"

	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/miekg/dns"
)

type mockDNS64Resolver struct {
	aIP net.IP
}

func (m *mockDNS64Resolver) Resolve(ctx context.Context, rawQuery []byte) ([]byte, error) {
	return nil, nil
}

func (m *mockDNS64Resolver) Exchange(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	resp := new(dns.Msg)
	resp.SetReply(req)
	if len(req.Question) > 0 && req.Question[0].Qtype == dns.TypeA {
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   m.aIP,
		})
	}
	return resp, nil
}

func (m *mockDNS64Resolver) Configure(cfg core.ResolverConfig) error { return nil }
func (m *mockDNS64Resolver) Shutdown() error                         { return nil }

func TestDNS64Middleware_Synthesis(t *testing.T) {
	synth, err := core.NewDNS64Synthesizer(core.DNS64Config{
		Enabled:  true,
		Prefixes: []string{core.WellKnownDNS64Prefix},
	})
	if err != nil {
		t.Fatalf("failed to create synth: %v", err)
	}

	mockRes := &mockDNS64Resolver{aIP: net.ParseIP("192.0.2.33")}
	mw := NewDNS64Middleware(synth, mockRes)

	// 1. AAAA 查询，下游返回 NODATA
	req := new(dns.Msg)
	req.SetQuestion("ipv4only.example.com.", dns.TypeAAAA)
	ctx := NewDNSContext(context.Background(), req, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5353}, "udp")

	err = mw(ctx, func() error {
		// 模拟下游返回 NODATA 空应答
		resp := new(dns.Msg)
		resp.SetReply(req)
		ctx.Resp = resp
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.Resp == nil || len(ctx.Resp.Answer) != 1 {
		t.Fatalf("expected 1 synthesized AAAA record, got %d", len(ctx.Resp.Answer))
	}

	aaaa, ok := ctx.Resp.Answer[0].(*dns.AAAA)
	if !ok {
		t.Fatalf("expected *dns.AAAA answer")
	}
	expectedIP := "64:ff9b::c000:221"
	if aaaa.AAAA.String() != expectedIP {
		t.Fatalf("expected %s, got %s", expectedIP, aaaa.AAAA.String())
	}
	if val, ok := ctx.Get("dns64_synthesized"); !ok || val != true {
		t.Fatal("expected dns64_synthesized attribute")
	}

	// 2. 原生已有 AAAA 记录时不得合成
	reqNative := new(dns.Msg)
	reqNative.SetQuestion("dualstack.example.com.", dns.TypeAAAA)
	ctxNative := NewDNSContext(context.Background(), reqNative, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5353}, "udp")

	err = mw(ctxNative, func() error {
		resp := new(dns.Msg)
		resp.SetReply(reqNative)
		resp.Answer = append(resp.Answer, &dns.AAAA{
			Hdr:  dns.RR_Header{Name: "dualstack.example.com.", Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
			AAAA: net.ParseIP("2001:db8::1"),
		})
		ctxNative.Resp = resp
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ctxNative.Resp.Answer) != 1 {
		t.Fatalf("expected unchanged native AAAA record count, got %d", len(ctxNative.Resp.Answer))
	}
	if _, ok := ctxNative.Get("dns64_synthesized"); ok {
		t.Fatal("dns64_synthesized should NOT be set when native AAAA exists")
	}
}
