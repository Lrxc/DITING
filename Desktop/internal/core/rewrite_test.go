package core

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestRewriteEngine_ExactAndWildcard(t *testing.T) {
	cfg := RewriteConfig{
		Enabled: true,
		TTL:     120,
		Rules: []RewriteRule{
			{
				Domain: "nas.lan",
				IPs:    []string{"192.168.1.100", "fe80::1"},
			},
			{
				Domain: "*.test",
				IPs:    []string{"127.0.0.1"},
			},
			{
				Domain: "alias.local",
				CNAME:  "target.example.com",
			},
			{
				Domain: "meta.lan",
				TXT:    []string{"v=spf1 ~all", "status=ok"},
			},
		},
	}

	engine := NewRewriteEngine(cfg)

	// 1. 测试精准匹配 A 记录
	answers, matched := engine.Lookup("nas.lan.", dns.TypeA)
	if !matched || len(answers) != 1 {
		t.Fatalf("expected 1 A record for nas.lan, got %d, matched=%v", len(answers), matched)
	}
	aRecord, ok := answers[0].(*dns.A)
	if !ok || !aRecord.A.Equal(net.ParseIP("192.168.1.100")) {
		t.Fatalf("unexpected A record value: %v", answers[0])
	}
	if aRecord.Hdr.Ttl != 120 {
		t.Fatalf("expected TTL 120, got %d", aRecord.Hdr.Ttl)
	}

	// 2. 测试精准匹配 AAAA 记录
	answers, matched = engine.Lookup("nas.lan", dns.TypeAAAA)
	if !matched || len(answers) != 1 {
		t.Fatalf("expected 1 AAAA record for nas.lan, got %d", len(answers))
	}
	aaaaRecord, ok := answers[0].(*dns.AAAA)
	if !ok || !aaaaRecord.AAAA.Equal(net.ParseIP("fe80::1")) {
		t.Fatalf("unexpected AAAA record value: %v", answers[0])
	}

	// 3. 测试泛域名匹配
	subdomains := []string{"app.test", "api.v1.test", "my-site.test."}
	for _, sub := range subdomains {
		answers, matched = engine.Lookup(sub, dns.TypeA)
		if !matched || len(answers) != 1 {
			t.Fatalf("expected wildcard match for %s, got len=%d matched=%v", sub, len(answers), matched)
		}
		aRec := answers[0].(*dns.A)
		if !aRec.A.Equal(net.ParseIP("127.0.0.1")) {
			t.Fatalf("wildcard IP mismatch: %v", aRec.A)
		}
	}

	// 4. 测试 CNAME 记录重写 (支持 TypeA 和 TypeCNAME 查询)
	answers, matched = engine.Lookup("alias.local", dns.TypeA)
	if !matched || len(answers) != 1 {
		t.Fatalf("expected CNAME for alias.local A query, got %d", len(answers))
	}
	cnameRec, ok := answers[0].(*dns.CNAME)
	if !ok || cnameRec.Target != "target.example.com." {
		t.Fatalf("unexpected CNAME target: %v", answers[0])
	}

	// 5. 测试 TXT 记录重写
	answers, matched = engine.Lookup("meta.lan", dns.TypeTXT)
	if !matched || len(answers) != 1 {
		t.Fatalf("expected TXT record for meta.lan, got %d", len(answers))
	}
	txtRec := answers[0].(*dns.TXT)
	if len(txtRec.Txt) != 2 || txtRec.Txt[0] != "v=spf1 ~all" {
		t.Fatalf("unexpected TXT record values: %v", txtRec.Txt)
	}

	// 6. 测试权威 NODATA (匹配了域名但未匹配查询类型，如 MX)
	answers, matched = engine.Lookup("nas.lan", dns.TypeMX)
	if !matched {
		t.Fatalf("expected matched=true for NODATA detection")
	}
	if len(answers) != 0 {
		t.Fatalf("expected empty answers for NODATA, got %d", len(answers))
	}

	// 7. 测试自动反向 PTR 解析
	ptrQuery, err := dns.ReverseAddr("192.168.1.100")
	if err != nil {
		t.Fatalf("reverseAddr failed: %v", err)
	}
	ptrAns, ptrMatched := engine.LookupPTR(ptrQuery)
	if !ptrMatched || len(ptrAns) != 1 {
		t.Fatalf("expected PTR match for 192.168.1.100, got len=%d matched=%v", len(ptrAns), ptrMatched)
	}
	ptrRecord, ok := ptrAns[0].(*dns.PTR)
	if !ok || ptrRecord.Ptr != "nas.lan." {
		t.Fatalf("unexpected PTR target: %v", ptrAns[0])
	}
}

func TestRewriteEngine_DynamicRules(t *testing.T) {
	engine := NewRewriteEngine(DefaultRewriteConfig())

	// 初始状态无规则
	if _, matched := engine.Lookup("dynamic.dev", dns.TypeA); matched {
		t.Fatalf("should not match initially")
	}

	// 动态添加
	err := engine.AddRule(RewriteRule{
		Domain: "dynamic.dev",
		IPs:    []string{"10.10.10.10"},
	})
	if err != nil {
		t.Fatalf("add rule error: %v", err)
	}

	ans, matched := engine.Lookup("dynamic.dev", dns.TypeA)
	if !matched || len(ans) != 1 {
		t.Fatalf("expected dynamic.dev to match after AddRule")
	}

	// 动态删除
	deleted := engine.DeleteRule("dynamic.dev")
	if !deleted {
		t.Fatalf("expected DeleteRule to return true")
	}

	_, matched = engine.Lookup("dynamic.dev", dns.TypeA)
	if matched {
		t.Fatalf("should not match after DeleteRule")
	}
}
