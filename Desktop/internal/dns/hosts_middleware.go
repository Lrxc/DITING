package dns

import (
	"strings"

	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/miekg/dns"
)

// NewHostsMiddleware 创建系统与自定义 Hosts 本地优先解析中间件
// 借鉴自 blocky (hosts_file_resolver.go) 与 AdGuard Home (dnsforward/dnsforward.go)
func NewHostsMiddleware(resolver *core.HostsResolver) Middleware {
	return func(ctx *DNSContext, next func() error) error {
		if resolver == nil || ctx.Req == nil || len(ctx.Req.Question) == 0 {
			return next()
		}

		q := ctx.Req.Question[0]
		domain := strings.Trim(q.Name, ".")
		if domain == "" {
			return next()
		}

		var answers []dns.RR
		if q.Qtype == dns.TypePTR {
			answers = resolver.LookupPTR(q.Name)
		} else if q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA {
			answers = resolver.Lookup(domain, q.Qtype)
		}

		if len(answers) > 0 {
			resp := new(dns.Msg)
			resp.SetReply(ctx.Req)
			resp.Authoritative = true
			resp.RecursionAvailable = true
			resp.Rcode = dns.RcodeSuccess
			resp.Answer = answers
			ctx.Resp = resp
			ctx.Set("source", "hosts")
			ctx.Set("cache_hit", "hosts")
			return nil
		}

		return next()
	}
}
