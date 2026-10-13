package dns

import (
	"strings"

	"github.com/miekg/dns"
)

// NewIPv6Middleware 创建 IPv6 AAAA 查询过滤与极速 NODATA 响应中间件
// 深度借鉴自 dnscrypt-proxy (plugin_block_ipv6.go)、AdGuard Home (block_ipv6) 与 Pi-hole (block-ipv6)
// 在部分家庭或企业宽带中，IPv6 路由质量低下或配置异常会导致双栈握手严重超时（Happy Eyeballs 卡顿）。
// 开启此功能后，本地即刻秒级应答权威 NODATA (NOERROR + SOA)，强制终端立即回退至极速 IPv4 连接。
func NewIPv6Middleware(enabledFunc func() bool) Middleware {
	return func(ctx *DNSContext, next func() error) error {
		if ctx.Req == nil || len(ctx.Req.Question) == 0 {
			return next()
		}

		isEnabled := false
		if enabledFunc != nil {
			isEnabled = enabledFunc()
		}
		if !isEnabled {
			return next()
		}

		q := ctx.Req.Question[0]
		if q.Qtype != dns.TypeAAAA {
			return next()
		}

		resp := new(dns.Msg)
		resp.SetReply(ctx.Req)
		resp.Authoritative = true
		resp.RecursionAvailable = true
		resp.Rcode = dns.RcodeSuccess

		soaZone := q.Name
		if !strings.HasSuffix(soaZone, ".") {
			soaZone += "."
		}

		// 附加 RFC 2308 负缓存授权区 SOA 记录
		resp.Ns = append(resp.Ns, &dns.SOA{
			Hdr: dns.RR_Header{
				Name:   soaZone,
				Rrtype: dns.TypeSOA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			Ns:      "localhost.",
			Mbox:    "nobody.invalid.",
			Serial:  1,
			Refresh: 10000,
			Retry:   2400,
			Expire:  604800,
			Minttl:  300,
		})

		ctx.Resp = resp
		ctx.Set("source", "block_ipv6")
		ctx.Set("ipv6_blocked", true)
		return nil
	}
}
