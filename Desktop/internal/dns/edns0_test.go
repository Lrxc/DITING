package dns

import (
	"testing"

	"github.com/miekg/dns"
)

func TestEDNS0_SetAndGetOption(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	if SetEdns0Option(nil, new(dns.EDNS0_EDE)) {
		t.Fatal("expected false for nil msg")
	}
	if SetEdns0Option(msg, nil) {
		t.Fatal("expected false for nil opt")
	}

	ede := &dns.EDNS0_EDE{
		InfoCode:  dns.ExtendedErrorCodeBlocked,
		ExtraText: "DITING: Blocked by test rule",
	}

	ok := SetEdns0Option(msg, ede)
	if !ok {
		t.Fatal("failed to set EDNS0 option")
	}

	opt := msg.IsEdns0()
	if opt == nil {
		t.Fatal("expected OPT record in msg")
	}

	retrieved := GetEdns0Option[*dns.EDNS0_EDE](msg)
	if retrieved == nil {
		t.Fatal("expected to retrieve EDE option")
	}

	if retrieved.InfoCode != dns.ExtendedErrorCodeBlocked {
		t.Fatalf("expected info code %d, got %d", dns.ExtendedErrorCodeBlocked, retrieved.InfoCode)
	}
	if retrieved.ExtraText != "DITING: Blocked by test rule" {
		t.Fatalf("unexpected extra text: %s", retrieved.ExtraText)
	}

	// Replacement of same option code
	newEde := &dns.EDNS0_EDE{
		InfoCode:  dns.ExtendedErrorCodeProhibited,
		ExtraText: "Replaced",
	}
	SetEdns0Option(msg, newEde)
	opt = msg.IsEdns0()
	if len(opt.Option) != 1 {
		t.Fatalf("expected 1 option after replacement, got %d", len(opt.Option))
	}
	retrieved = GetEdns0Option[*dns.EDNS0_EDE](msg)
	if retrieved.ExtraText != "Replaced" {
		t.Fatalf("expected Replaced, got %s", retrieved.ExtraText)
	}
}

func TestSetEDE_HelperAndTruncate(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion("example.com.", dns.TypeA)

	longText := ""
	for i := 0; i < 300; i++ {
		longText += "a"
	}
	SetEDE(msg, dns.ExtendedErrorCodeNetworkError, longText)

	opt := GetEdns0Option[*dns.EDNS0_EDE](msg)
	if opt == nil {
		t.Fatal("expected EDE option")
	}
	if len(opt.ExtraText) != MaxEDETextLength {
		t.Fatalf("expected truncated length %d, got %d", MaxEDETextLength, len(opt.ExtraText))
	}

	// Test UTF-8 multi-byte truncate
	chineseText := "谛听DNS安全防御系统测试很长很长很长很长很长很长很长很长很长很长很长很长"
	truncated := TruncateText(chineseText, 10)
	if len(truncated) > 10 {
		t.Fatalf("truncated string exceeded 10 bytes: len=%d", len(truncated))
	}
}
