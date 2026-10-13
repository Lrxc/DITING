package dns

import (
	"github.com/haoze-evolluling/diting/desktop/internal/core"
)

// NewClientMiddleware 创建客户端主机名识别与活动记录中间件
// 深度借鉴自 Blocky (client_names_resolver.go) 与 AdGuard Home (internal/rdns)
func NewClientMiddleware(resolver *core.ClientResolver) Middleware {
	return func(ctx *DNSContext, next func() error) error {
		if resolver != nil && ctx.ClientIP != nil {
			name := resolver.IdentifyClient(ctx.ClientIP)
			ctx.ClientName = name
			if name != "" {
				ctx.Set("client_name", name)
			}
		}
		return next()
	}
}
