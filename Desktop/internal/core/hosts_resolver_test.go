package core

import (
	"strings"
	"testing"

	"github.com/miekg/dns"
)

func TestHostsResolver_LoadReaderAndLookup(t *testing.T) {
	content := `
# System hosts mock
127.0.0.1 localhost myapp.local
192.168.1.100 nas.home storage.home # lan nas
::1 localhost ip6-localhost
2001:db8::1 ipv6.test.local
`
	hr, err := NewHostsResolver(HostsConfig{Enabled: false})
	if err != nil {
		t.Fatalf("failed to create HostsResolver: %v", err)
	}

	hr.LoadReader(strings.NewReader(content))

	// Test IPv4 lookup
	answersA := hr.Lookup("myapp.local", dns.TypeA)
	if len(answersA) != 1 {
		t.Fatalf("expected 1 A record for myapp.local, got %d", len(answersA))
	}
	aRecord, ok := answersA[0].(*dns.A)
	if !ok || aRecord.A.String() != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %v", answersA[0])
	}

	// Test IPv6 lookup
	answersAAAA := hr.Lookup("ipv6.test.local", dns.TypeAAAA)
	if len(answersAAAA) != 1 {
		t.Fatalf("expected 1 AAAA record for ipv6.test.local, got %d", len(answersAAAA))
	}
	aaaaRecord, ok := answersAAAA[0].(*dns.AAAA)
	if !ok || aaaaRecord.AAAA.String() != "2001:db8::1" {
		t.Fatalf("expected 2001:db8::1, got %v", answersAAAA[0])
	}

	// Test not found
	notFound := hr.Lookup("notfound.com", dns.TypeA)
	if len(notFound) != 0 {
		t.Fatalf("expected 0 records for non-existent domain, got %d", len(notFound))
	}

	// Test Reverse PTR lookup
	ptrAnswers := hr.LookupPTR("100.1.168.192.in-addr.arpa.")
	if len(ptrAnswers) != 1 {
		t.Fatalf("expected 1 PTR record for 192.168.1.100, got %d", len(ptrAnswers))
	}
	ptrRecord, ok := ptrAnswers[0].(*dns.PTR)
	if !ok || ptrRecord.Ptr != "nas.home." {
		t.Fatalf("expected nas.home., got %v", ptrAnswers[0])
	}
}

func TestParseIPFromArpa(t *testing.T) {
	// Valid IPv4
	ip4 := ParseIPFromArpa("1.1.168.192.in-addr.arpa.")
	if ip4 == nil || ip4.String() != "192.168.1.1" {
		t.Fatalf("expected 192.168.1.1, got %v", ip4)
	}

	// Invalid
	bad := ParseIPFromArpa("invalid.in-addr.arpa")
	if bad != nil {
		t.Fatalf("expected nil for invalid arpa, got %v", bad)
	}
}
