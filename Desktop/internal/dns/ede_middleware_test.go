package dns

import (
	"context"
	"errors"
	"testing"

	"github.com/miekg/dns"
)

func TestEDEMiddleware_BlockedByFilter(t *testing.T) {
	mw := NewEDEMiddleware(true)
	req := new(dns.Msg)
	req.SetEdns0(4096, false)
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     req,
		Resp:    new(dns.Msg),
	}
	ctx.Set("filter_blocked", true)
	ctx.Set("filter_reason", "Malware domain blocked")

	err := mw(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ede := GetEdns0Option[*dns.EDNS0_EDE](ctx.Resp)
	if ede == nil {
		t.Fatal("expected EDE option in response")
	}
	if ede.InfoCode != dns.ExtendedErrorCodeBlocked {
		t.Fatalf("expected code %d, got %d", dns.ExtendedErrorCodeBlocked, ede.InfoCode)
	}
	if ede.ExtraText != "DITING: Malware domain blocked" {
		t.Fatalf("unexpected extra text: %s", ede.ExtraText)
	}
}

func TestEDEMiddleware_BlockedByRebind(t *testing.T) {
	mw := NewEDEMiddleware(true)
	req := new(dns.Msg)
	req.SetEdns0(4096, false)
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     req,
		Resp:    new(dns.Msg),
	}
	ctx.Set("rebinding_blocked", true)

	err := mw(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ede := GetEdns0Option[*dns.EDNS0_EDE](ctx.Resp)
	if ede == nil {
		t.Fatal("expected EDE option in response")
	}
	if ede.InfoCode != dns.ExtendedErrorCodeProhibited {
		t.Fatalf("expected code %d, got %d", dns.ExtendedErrorCodeProhibited, ede.InfoCode)
	}
}

func TestEDEMiddleware_UpstreamFailure(t *testing.T) {
	mw := NewEDEMiddleware(true)
	req := new(dns.Msg)
	req.SetEdns0(4096, false)
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     req,
		Resp:    new(dns.Msg),
		Err:     errors.New("connection timeout"),
	}
	ctx.Resp.Rcode = dns.RcodeServerFailure

	_ = mw(ctx, func() error {
		return ctx.Err
	})

	ede := GetEdns0Option[*dns.EDNS0_EDE](ctx.Resp)
	if ede == nil {
		t.Fatal("expected EDE option in response")
	}
	if ede.InfoCode != dns.ExtendedErrorCodeNetworkError {
		t.Fatalf("expected code %d, got %d", dns.ExtendedErrorCodeNetworkError, ede.InfoCode)
	}
}

func TestEDEMiddleware_NoClientEDNS0(t *testing.T) {
	// 验证 RFC 8914 Section 3: 客户端请求未提供 OPT RR 时，响应严禁附带 EDE
	mw := NewEDEMiddleware(true)
	req := new(dns.Msg) // 未调用 SetEdns0
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     req,
		Resp:    new(dns.Msg),
	}
	ctx.Set("filter_blocked", true)

	_ = mw(ctx, func() error { return nil })
	if GetEdns0Option[*dns.EDNS0_EDE](ctx.Resp) != nil {
		t.Fatal("RFC 8914 violation: EDE option must not be attached when client request lacks EDNS0")
	}
}

func TestEDEMiddleware_Disabled(t *testing.T) {
	mw := NewEDEMiddleware(false)
	req := new(dns.Msg)
	req.SetEdns0(4096, false)
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     req,
		Resp:    new(dns.Msg),
	}
	ctx.Set("filter_blocked", true)

	_ = mw(ctx, func() error { return nil })
	if GetEdns0Option[*dns.EDNS0_EDE](ctx.Resp) != nil {
		t.Fatal("expected no EDE option when disabled")
	}
}
