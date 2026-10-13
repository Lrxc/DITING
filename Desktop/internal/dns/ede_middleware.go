package dns

import (
	"fmt"

	"github.com/miekg/dns"
)

// NewEDEMiddleware 创建遵循 RFC 8914 规范的 Extended DNS Errors (EDE) 响应增强中间件。
// 借鉴自 blocky (ede_resolver.go) 与 AdGuard Home，为客户端提供详尽的标准 DNS 错误与拦截原因。
func NewEDEMiddleware(enabled bool) Middleware {
	return func(ctx *DNSContext, next func() error) error {
		err := next()
		// RFC 8914 Section 3: 响应中不得包含 EDE 选项，除非客户端查询本身携带了 OPT pseudo-RR (EDNS0)
		if !enabled || ctx.Resp == nil || ctx.Req == nil || ctx.Req.IsEdns0() == nil {
			return err
		}

		// 1. DNS Rebinding 防御拦截
		if val, ok := ctx.Get("rebinding_blocked"); ok && val == true {
			reason := "DITING: Blocked by DNS rebinding protection"
			if rVal, ok := ctx.Get("filter_reason"); ok {
				if rStr, ok := rVal.(string); ok && rStr != "" {
					reason = fmt.Sprintf("DITING: %s", rStr)
				}
			}
			SetEDE(ctx.Resp, dns.ExtendedErrorCodeProhibited, reason)
			return err
		}

		// 2. 规则过滤拦截 (AdGuard / 黑名单)
		if val, ok := ctx.Get("filter_blocked"); ok && val == true {
			reason := "DITING: Blocked by filter rule"
			if rVal, ok := ctx.Get("filter_reason"); ok {
				if rStr, ok := rVal.(string); ok && rStr != "" {
					reason = fmt.Sprintf("DITING: %s", rStr)
				}
			}
			SetEDE(ctx.Resp, dns.ExtendedErrorCodeBlocked, reason)
			return err
		}

		// 3. 上游解析故障 (SERVFAIL / Network Error)
		if ctx.Err != nil || ctx.Resp.Rcode == dns.RcodeServerFailure {
			errMsg := "DITING: Upstream resolution failed"
			if ctx.Err != nil {
				errMsg = fmt.Sprintf("DITING: %v", ctx.Err)
			}
			SetEDE(ctx.Resp, dns.ExtendedErrorCodeNetworkError, errMsg)
			return err
		}

		// 4. 过期容灾缓存 (Stale-While-Revalidate)
		if val, ok := ctx.Get("cache_hit"); ok && val == "stale" {
			SetEDE(ctx.Resp, dns.ExtendedErrorCodeCachedError, "DITING: Stale cache response (network degraded)")
			return err
		}

		return err
	}
}
