package dns

import (
	"context"
	"errors"
	"testing"

	"github.com/miekg/dns"
)

func TestEDEMiddleware_BlockedByFilter(t *testing.T) {
	mw := NewEDEMiddleware(true)
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     new(dns.Msg),
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
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     new(dns.Msg),
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
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     new(dns.Msg),
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

func TestEDEMiddleware_Disabled(t *testing.T) {
	mw := NewEDEMiddleware(false)
	ctx := &DNSContext{
		Context: context.Background(),
		Req:     new(dns.Msg),
		Resp:    new(dns.Msg),
	}
	ctx.Set("filter_blocked", true)

	_ = mw(ctx, func() error { return nil })
	if GetEdns0Option[*dns.EDNS0_EDE](ctx.Resp) != nil {
		t.Fatal("expected no EDE option when disabled")
	}
}
