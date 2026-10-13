package dns

import (
	"context"

	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/miekg/dns"
)

// NewDNS64Middleware 创建 DNS64 IPv6-only 过渡合成洋葱中间件。
// 借鉴自 blocky (dns64_resolver.go)、AdGuard Home (dns64.go) 与 RFC 6147 规范。
// 当 IPv6 客户端查询 AAAA 记录且上游无 IPv6 响应时，自动查询 A 记录并按 RFC 6052 合成 AAAA 响应。
func NewDNS64Middleware(synth *core.DNS64Synthesizer, resolver core.Resolver) Middleware {
	return func(ctx *DNSContext, next func() error) error {
		if synth == nil || !synth.IsEnabled() || resolver == nil || ctx.Req == nil || len(ctx.Req.Question) == 0 {
			return next()
		}

		q := ctx.Req.Question[0]
		if q.Qtype != dns.TypeAAAA {
			return next()
		}

		// 优先执行下游解析链（查本地缓存与上游）
		if err := next(); err != nil {
			return err
		}

		if ctx.Resp == nil {
			return nil
		}

		// RFC 6147 Section 5.1.1: 若应答中已包含权威/上游给出的 AAAA 记录，则不得进行合成
		hasAAAA := false
		for _, rr := range ctx.Resp.Answer {
			if _, ok := rr.(*dns.AAAA); ok {
				hasAAAA = true
				break
			}
		}
		if hasAAAA {
			return nil
		}

		// 发起同名 A 记录解析请求
		reqCtx := ctx.Context
		if reqCtx == nil {
			reqCtx = context.Background()
		}

		aReq := new(dns.Msg)
		aReq.SetQuestion(q.Name, dns.TypeA)
		aReq.RecursionDesired = true

		aResp, err := resolver.Exchange(reqCtx, aReq)
		if err != nil || aResp == nil || len(aResp.Answer) == 0 {
			return nil
		}

		var aRecords []*dns.A
		for _, rr := range aResp.Answer {
			if a, ok := rr.(*dns.A); ok {
				aRecords = append(aRecords, a)
			}
		}

		if len(aRecords) == 0 {
			return nil
		}

		aaaaRecords := synth.SynthesizeFromA(aRecords)
		if len(aaaaRecords) == 0 {
			return nil
		}

		// 将合成的 AAAA 记录注入响应应答区
		for _, aaaa := range aaaaRecords {
			ctx.Resp.Answer = append(ctx.Resp.Answer, aaaa)
		}
		ctx.Resp.Rcode = dns.RcodeSuccess
		ctx.Set("dns64_synthesized", true)
		return nil
	}
}
