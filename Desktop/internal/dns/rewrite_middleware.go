package dns

import (
	"strings"

	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/miekg/dns"
)

// NewRewriteMiddleware 创建自定义 DNS 重写与伪装拦截中间件
// 深度借鉴自 AdGuard Home (dnsrewrite) 与 Blocky (custom_dns_resolver)
func NewRewriteMiddleware(engine *core.RewriteEngine) Middleware {
	return func(ctx *DNSContext, next func() error) error {
		if engine == nil || !engine.IsEnabled() || ctx.Req == nil || len(ctx.Req.Question) == 0 {
			return next()
		}

		q := ctx.Req.Question[0]
		domain := strings.Trim(q.Name, ".")
		if domain == "" {
			return next()
		}

		var answers []dns.RR
		var matched bool

		if q.Qtype == dns.TypePTR {
			answers, matched = engine.LookupPTR(q.Name)
		} else {
			answers, matched = engine.Lookup(domain, q.Qtype)
		}

		if matched {
			resp := new(dns.Msg)
			resp.SetReply(ctx.Req)
			resp.Authoritative = true
			resp.RecursionAvailable = true
			resp.Rcode = dns.RcodeSuccess
			resp.Answer = answers

			ctx.Resp = resp
			ctx.Set("source", "rewrite")
			if len(answers) > 0 {
				ctx.Set("cache_hit", "rewrite")
			} else {
				// 权威 NODATA 命中（如仅配置 A 记录但请求 AAAA/MX），阻断外泄
				ctx.Set("cache_hit", "rewrite_nodata")
			}
			return nil
		}

		return next()
	}
}
