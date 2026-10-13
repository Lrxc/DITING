package core

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestDNS64_MapIPv4ToIPv6(t *testing.T) {
	cfg := DNS64Config{
		Enabled:  true,
		Prefixes: []string{WellKnownDNS64Prefix},
	}
	synth, err := NewDNS64Synthesizer(cfg)
	if err != nil {
		t.Fatalf("failed to create synthesizer: %v", err)
	}

	ipv4 := net.ParseIP("192.0.2.1")
	v6 := synth.MapIPv4ToIPv6(ipv4)
	if v6 == nil {
		t.Fatal("expected synthesized IPv6 address")
	}

	// 192.0.2.1 对应十六进制 c0.00.02.01
	expected := "64:ff9b::c000:201"
	if v6.String() != expected {
		t.Fatalf("expected %s, got %s", expected, v6.String())
	}
}

func TestDNS64_SynthesizeFromA(t *testing.T) {
	cfg := DNS64Config{
		Enabled:  true,
		Prefixes: []string{WellKnownDNS64Prefix},
	}
	synth, err := NewDNS64Synthesizer(cfg)
	if err != nil {
		t.Fatalf("failed to create synthesizer: %v", err)
	}

	aRecords := []*dns.A{
		{
			Hdr: dns.RR_Header{Name: "ipv4only.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("93.184.216.34"),
		},
	}

	aaaaRecords := synth.SynthesizeFromA(aRecords)
	if len(aaaaRecords) != 1 {
		t.Fatalf("expected 1 AAAA record, got %d", len(aaaaRecords))
	}

	aaaa := aaaaRecords[0]
	if aaaa.Hdr.Name != "ipv4only.example.com." || aaaa.Hdr.Rrtype != dns.TypeAAAA || aaaa.Hdr.Ttl != 300 {
		t.Fatalf("invalid synthesized AAAA header: %+v", aaaa.Hdr)
	}
	expected := "64:ff9b::5db8:d822"
	if aaaa.AAAA.String() != expected {
		t.Fatalf("expected %s, got %s", expected, aaaa.AAAA.String())
	}
}

func TestDNS64_ExclusionSet(t *testing.T) {
	cfg := DNS64Config{
		Enabled:  true,
		Prefixes: []string{WellKnownDNS64Prefix},
	}
	synth, err := NewDNS64Synthesizer(cfg)
	if err != nil {
		t.Fatalf("failed to create synthesizer: %v", err)
	}

	// Loopback ::1 must be excluded
	if !synth.IsExcluded(net.ParseIP("::1")) {
		t.Fatal("::1 should be excluded")
	}

	// Link-local fe80::1 must be excluded
	if !synth.IsExcluded(net.ParseIP("fe80::1")) {
		t.Fatal("fe80::1 should be excluded")
	}

	// Well-known prefix itself must be excluded to prevent loops
	if !synth.IsExcluded(net.ParseIP("64:ff9b::192.0.2.1")) {
		t.Fatal("DNS64 synthesized address should be in exclusion set")
	}

	// Normal public IPv6 should not be excluded
	if synth.IsExcluded(net.ParseIP("2606:4700:4700::1111")) {
		t.Fatal("normal IPv6 should not be excluded")
	}
}

func TestDNS64_InvalidPrefix(t *testing.T) {
	cfg := DNS64Config{
		Enabled:  true,
		Prefixes: []string{"2001:db8::/32"}, // not /96
	}
	_, err := NewDNS64Synthesizer(cfg)
	if err == nil {
		t.Fatal("expected error for non-/96 prefix")
	}
}
