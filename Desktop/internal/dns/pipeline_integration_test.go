package dns

import (
	"context"
	"net"
	"testing"

	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/miekg/dns"
)

type pipelineMockResolver struct {
	answers map[string][]dns.RR
}

func (m *pipelineMockResolver) Resolve(ctx context.Context, rawQuery []byte) ([]byte, error) {
	return nil, nil
}

func (m *pipelineMockResolver) Exchange(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	resp := new(dns.Msg)
	resp.SetReply(req)
	if len(req.Question) > 0 {
		key := req.Question[0].Name
		if rrs, ok := m.answers[key]; ok {
			for _, rr := range rrs {
				if rr.Header().Rrtype == req.Question[0].Qtype {
					resp.Answer = append(resp.Answer, rr)
				}
			}
		}
	}
	return resp, nil
}

func (m *pipelineMockResolver) Configure(cfg core.ResolverConfig) error { return nil }
func (m *pipelineMockResolver) Shutdown() error                         { return nil }

func TestPipeline_FullIntegration(t *testing.T) {
	// 1. 初始化各模块
	res := &pipelineMockResolver{
		answers: map[string][]dns.RR{
			"public.example.com.": {
				&dns.A{Hdr: dns.RR_Header{Name: "public.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.ParseIP("93.184.216.34")},
			},
			"rebind.example.com.": {
				&dns.A{Hdr: dns.RR_Header{Name: "rebind.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.ParseIP("192.168.1.1")},
			},
			"ipv4only.example.com.": {
				&dns.A{Hdr: dns.RR_Header{Name: "ipv4only.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.ParseIP("8.8.8.8")},
			},
			"rebind-ipv4only.example.com.": {
				&dns.A{Hdr: dns.RR_Header{Name: "rebind-ipv4only.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.ParseIP("10.0.0.1")},
			},
		},
	}

	ac, err := core.NewAccessControl(core.AccessControlConfig{
		BlockedClients: []string{"192.168.100.99"},
	})
	if err != nil {
		t.Fatalf("failed to init AccessControl: %v", err)
	}

	limiter := core.NewRateLimiter(context.Background(), core.RateLimitConfig{
		Enabled:    true,
		Rate:       10.0,
		Burst:      2,
		IPv4Prefix: 32,
		IPv6Prefix: 128,
		ExemptIPs:  []string{"127.0.0.1"},
	})
	defer limiter.Close()

	rebindMgr := NewRebindProtectionManager(core.RebindConfig{
		Enabled:        true,
		AllowedDomains: []string{"allowed.local"},
	})

	dns64Synth, err := core.NewDNS64Synthesizer(core.DNS64Config{
		Enabled:  true,
		Prefixes: []string{core.WellKnownDNS64Prefix},
	})
	if err != nil {
		t.Fatalf("failed to init DNS64: %v", err)
	}

	edeMw := NewEDEMiddleware(true)
	accessMw := NewAccessMiddleware(ac)
	rateLimitMw := NewRateLimitMiddleware(limiter)
	unqualifiedMw := NewUnqualifiedMiddleware(true)
	rebindMw := NewRebindMiddleware(rebindMgr, nil)
	dns64Mw := NewDNS64Middleware(dns64Synth, res)
	forwardMw := NewForwardMiddleware(res)

	pipeline := NewPipeline(edeMw, accessMw, rateLimitMw, unqualifiedMw, rebindMw, dns64Mw, forwardMw)

	// Case 1: 客户端访问控制阻断 + RFC 8914 EDE 验证
	req1 := new(dns.Msg)
	req1.SetQuestion("public.example.com.", dns.TypeA)
	req1.SetEdns0(4096, false)
	ctx1 := NewDNSContext(context.Background(), req1, &net.UDPAddr{IP: net.ParseIP("192.168.100.99"), Port: 1234}, "udp")

	if err := pipeline.Execute(ctx1); err != nil {
		t.Fatalf("pipeline execution error: %v", err)
	}
	if ctx1.Resp == nil || ctx1.Resp.Rcode != dns.RcodeRefused {
		t.Fatalf("expected REFUSED for blocked client, got %v", ctx1.Resp)
	}
	ede1 := GetEdns0Option[*dns.EDNS0_EDE](ctx1.Resp)
	if ede1 == nil || ede1.InfoCode != dns.ExtendedErrorCodeProhibited {
		t.Fatalf("expected EDE Prohibited (code 18), got %v", ede1)
	}

	// Case 2: 单标签不完整域名外泄阻断 + EDE 验证
	req2 := new(dns.Msg)
	req2.SetQuestion("printer.", dns.TypeA)
	req2.SetEdns0(4096, false)
	ctx2 := NewDNSContext(context.Background(), req2, &net.UDPAddr{IP: net.ParseIP("192.168.1.50"), Port: 1234}, "udp")

	if err := pipeline.Execute(ctx2); err != nil {
		t.Fatalf("pipeline execution error: %v", err)
	}
	if ctx2.Resp == nil || ctx2.Resp.Rcode != dns.RcodeNameError {
		t.Fatalf("expected NXDOMAIN for unqualified query, got %v", ctx2.Resp)
	}
	ede2 := GetEdns0Option[*dns.EDNS0_EDE](ctx2.Resp)
	if ede2 == nil || ede2.InfoCode != dns.ExtendedErrorCodeBlocked {
		t.Fatalf("expected EDE Blocked (code 15), got %v", ede2)
	}

	// Case 3: DNS64 正常合成 AAAA 记录
	req3 := new(dns.Msg)
	req3.SetQuestion("ipv4only.example.com.", dns.TypeAAAA)
	ctx3 := NewDNSContext(context.Background(), req3, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}, "udp")

	if err := pipeline.Execute(ctx3); err != nil {
		t.Fatalf("pipeline execution error: %v", err)
	}
	if ctx3.Resp == nil || len(ctx3.Resp.Answer) != 1 {
		t.Fatalf("expected 1 synthesized AAAA record, got %d", len(ctx3.Resp.Answer))
	}
	aaaa3, ok := ctx3.Resp.Answer[0].(*dns.AAAA)
	if !ok || aaaa3.AAAA.String() != "64:ff9b::808:808" {
		t.Fatalf("expected 64:ff9b::808:808, got %v", aaaa3)
	}

	// Case 4: DNS Rebinding 防御拦截 DNS64 合成的内网私有地址
	req4 := new(dns.Msg)
	req4.SetQuestion("rebind-ipv4only.example.com.", dns.TypeAAAA)
	req4.SetEdns0(4096, false)
	ctx4 := NewDNSContext(context.Background(), req4, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}, "udp")

	if err := pipeline.Execute(ctx4); err != nil {
		t.Fatalf("pipeline execution error: %v", err)
	}
	if ctx4.Resp == nil || len(ctx4.Resp.Answer) != 0 {
		t.Fatalf("expected rebind protection to strip private IP, got %v", ctx4.Resp)
	}
	if val, ok := ctx4.Get("rebinding_blocked"); !ok || val != true {
		t.Fatalf("expected rebinding_blocked attribute to be set")
	}
	ede4 := GetEdns0Option[*dns.EDNS0_EDE](ctx4.Resp)
	if ede4 == nil || ede4.InfoCode != dns.ExtendedErrorCodeProhibited {
		t.Fatalf("expected EDE Prohibited for rebinding attempt, got %v", ede4)
	}

	// Case 5: 速率限制限流测试
	clientIP := net.ParseIP("192.168.1.222")
	for i := 0; i < 2; i++ {
		req := new(dns.Msg)
		req.SetQuestion("public.example.com.", dns.TypeA)
		ctx := NewDNSContext(context.Background(), req, &net.UDPAddr{IP: clientIP, Port: 1234}, "udp")
		_ = pipeline.Execute(ctx)
		if ctx.Resp.Rcode != dns.RcodeSuccess {
			t.Fatalf("query within burst should succeed")
		}
	}
	// 超额请求触发限流
	reqBurst := new(dns.Msg)
	reqBurst.SetQuestion("public.example.com.", dns.TypeA)
	reqBurst.SetEdns0(4096, false)
	ctxBurst := NewDNSContext(context.Background(), reqBurst, &net.UDPAddr{IP: clientIP, Port: 1234}, "udp")
	_ = pipeline.Execute(ctxBurst)
	if ctxBurst.Resp == nil || ctxBurst.Resp.Rcode != dns.RcodeRefused {
		t.Fatalf("expected rate-limited query to return REFUSED")
	}
	edeBurst := GetEdns0Option[*dns.EDNS0_EDE](ctxBurst.Resp)
	if edeBurst == nil || edeBurst.InfoCode != dns.ExtendedErrorCodeProhibited {
		t.Fatalf("expected EDE Prohibited for rate limit")
	}
}

func TestPipeline_RewriteClientAndECSIntegration(t *testing.T) {
	mockRes := &pipelineMockResolver{
		answers: map[string][]dns.RR{
			"upstream.example.com.": {
				&dns.A{Hdr: dns.RR_Header{Name: "upstream.example.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}, A: net.ParseIP("1.1.1.1")},
			},
		},
	}

	rewriteEngine := core.NewRewriteEngine(core.RewriteConfig{
		Enabled: true,
		Rules: []core.RewriteRule{
			{Domain: "*.corp.lan", IPs: []string{"10.10.10.10"}},
		},
	})
	clientResolver := core.NewClientResolver(core.ClientConfig{
		Enabled: true,
		Aliases: []core.ClientAlias{
			{IP: "192.168.1.77", Name: "Office-Printer"},
		},
	}, rewriteEngine)
	ecsMgr := NewECSManager(core.ECSConfig{
		Enabled:  true,
		StripECS: true,
	})

	clientMw := NewClientMiddleware(clientResolver)
	rewriteMw := NewRewriteMiddleware(rewriteEngine)
	ecsMw := NewECSMiddleware(ecsMgr)
	forwardMw := NewForwardMiddleware(mockRes)

	pipeline := NewPipeline(clientMw, rewriteMw, ecsMw, forwardMw)

	// Case 1: 客户端识别与重写命中
	req1 := new(dns.Msg)
	req1.SetQuestion("portal.corp.lan.", dns.TypeA)
	ctx1 := NewDNSContext(context.Background(), req1, &net.UDPAddr{IP: net.ParseIP("192.168.1.77"), Port: 34567}, "udp")

	if err := pipeline.Execute(ctx1); err != nil {
		t.Fatalf("pipeline error: %v", err)
	}
	if ctx1.ClientName != "Office-Printer" {
		t.Fatalf("expected ClientName Office-Printer, got %s", ctx1.ClientName)
	}
	if ctx1.Resp == nil || len(ctx1.Resp.Answer) != 1 {
		t.Fatalf("expected rewrite answer, got %v", ctx1.Resp)
	}
	aRec := ctx1.Resp.Answer[0].(*dns.A)
	if !aRec.A.Equal(net.ParseIP("10.10.10.10")) {
		t.Fatalf("rewrite IP mismatch: %v", aRec.A)
	}

	// Case 2: 上游转发请求时自动剥离 ECS 隐私子网
	req2 := new(dns.Msg)
	req2.SetQuestion("upstream.example.com.", dns.TypeA)
	_ = SetECS(req2, "192.0.2.0/24")

	ctx2 := NewDNSContext(context.Background(), req2, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 34567}, "udp")
	if err := pipeline.Execute(ctx2); err != nil {
		t.Fatalf("pipeline error: %v", err)
	}
	if ctx2.ClientName != "localhost" {
		t.Fatalf("expected loopback to be localhost, got %s", ctx2.ClientName)
	}
	// 验证请求报文上的 ECS 已被顺利剥离
	if opt := GetEdns0Option[*dns.EDNS0_SUBNET](req2); opt != nil {
		t.Fatalf("expected ECS option to be stripped before upstream forwarding")
	}
}

