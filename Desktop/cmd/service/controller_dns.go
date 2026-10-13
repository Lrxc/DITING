package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/haoze-evolluling/diting/desktop/internal/core"
	ditingdns "github.com/haoze-evolluling/diting/desktop/internal/dns"
	"github.com/haoze-evolluling/diting/desktop/internal/ipc"
	"github.com/haoze-evolluling/diting/desktop/internal/platform"
	"github.com/haoze-evolluling/diting/desktop/internal/querylog"
	miekgdns "github.com/miekg/dns"
)

// StartDNS 实现 ServiceController 接口
func (p *program) StartDNS(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.dnsRunning {
		return nil
	}

	// 提前捕获引用：查询日志记录位于 DNS 热路径，回调内不得再取 p.mu
	ql := p.queryLog

	metricsMw := ditingdns.NewMetricsMiddleware(func(dctx *ditingdns.DNSContext, duration time.Duration, err error) {
		qName := ""
		qType := ""
		if dctx.Req != nil && len(dctx.Req.Question) > 0 {
			qName = dctx.Req.Question[0].Name
			qType = miekgdns.TypeToString[dctx.Req.Question[0].Qtype]
		}
		success := (err == nil)
		p.metrics.RecordQuery(duration, success)

		cacheHitStr := ""
		if val, ok := dctx.Get("cache_hit"); ok {
			if s, ok := val.(string); ok {
				cacheHitStr = s
			}
		}
		rcodeStr := ""
		if dctx.Resp != nil {
			rcodeStr = miekgdns.RcodeToString[dctx.Resp.Rcode]
		}

		filterBlocked := false
		filterRule := ""
		filterReason := ""
		if val, ok := dctx.Get("filter_blocked"); ok {
			if b, ok := val.(bool); ok {
				filterBlocked = b
			}
		}
		if val, ok := dctx.Get("rebinding_blocked"); ok {
			if b, ok := val.(bool); ok && b {
				filterBlocked = true
				if filterRule == "" {
					filterRule = "REBIND_PROTECTION"
				}
			}
		}
		if val, ok := dctx.Get("access_blocked"); ok {
			if b, ok := val.(bool); ok && b {
				filterBlocked = true
				if filterRule == "" {
					filterRule = "CLIENT_ACCESS_CONTROL"
				}
			}
		}
		if val, ok := dctx.Get("ratelimit_blocked"); ok {
			if b, ok := val.(bool); ok && b {
				filterBlocked = true
				if filterRule == "" {
					filterRule = "RATE_LIMIT_EXCEEDED"
				}
			}
		}
		if val, ok := dctx.Get("unqualified_blocked"); ok {
			if b, ok := val.(bool); ok && b {
				filterBlocked = true
				if filterRule == "" {
					filterRule = "BLOCK_UNQUALIFIED"
				}
			}
		}
		if val, ok := dctx.Get("filter_rule"); ok {
			if s, ok := val.(string); ok {
				filterRule = s
			}
		}
		if val, ok := dctx.Get("filter_reason"); ok {
			if s, ok := val.(string); ok {
				filterReason = s
			}
		}

		// 查询历史与统计收集：复用本回调已算好的字段，无需新增中间件
		if ql != nil {
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			status := querylog.StatusAllowed
			switch {
			case err != nil:
				status = querylog.StatusError
			case filterBlocked:
				status = querylog.StatusBlocked
			case cacheHitStr == "stale":
				status = querylog.StatusStale
			case cacheHitStr != "":
				status = querylog.StatusCached
			}
			ql.Record(querylog.Record{
				Timestamp:    time.Now().UnixMilli(),
				Domain:       qName,
				QType:        qType,
				ClientIP:     formatIP(dctx.ClientIP),
				ClientName:   dctx.ClientName,
				DurationMs:   float64(duration.Nanoseconds()) / 1e6,
				RCode:        rcodeStr,
				Status:       status,
				Cached:       cacheHitStr != "",
				CacheHit:     cacheHitStr,
				Blocked:      filterBlocked,
				FilterRule:   filterRule,
				FilterReason: filterReason,
				Protocol:     dctx.Protocol,
				Error:        errStr,
			})
		}

		if p.ipcServer != nil {
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			p.ipcServer.Broadcast(ipc.Event{
				Type:      "query",
				Timestamp: time.Now().UnixMilli(),
				Data: ipc.QueryEventData{
					Domain:       qName,
					QType:        qType,
					ClientIP:     formatIP(dctx.ClientIP),
					ClientName:   dctx.ClientName,
					DurationMs:   float64(duration.Nanoseconds()) / 1e6,
					Success:      success,
					RCode:        rcodeStr,
					CacheHit:     cacheHitStr,
					Blocked:      filterBlocked,
					FilterRule:   filterRule,
					FilterReason: filterReason,
					ErrorMessage: errStr,
				},
			})
		}
	})

	rebindMgr := p.rebindMgr
	if rebindMgr == nil {
		rebindMgr = ditingdns.NewRebindProtectionManager(p.cfg.Rebind)
	}
	clientMw := ditingdns.NewClientMiddleware(p.clientResolver)
	accessMw := ditingdns.NewAccessMiddleware(p.accessControl)
	rateLimitMw := ditingdns.NewRateLimitMiddleware(p.rateLimiter)
	edeMw := ditingdns.NewEDEMiddleware(func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cfg.DNS.EnableEDE
	})
	echMw := ditingdns.NewECHMiddleware(func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cfg.DNS.BlockECH
	})
	sudnMw := ditingdns.NewSUDNMiddleware()
	ipv6Mw := ditingdns.NewIPv6Middleware(func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cfg.DNS.BlockIPv6
	})
	hostsMw := ditingdns.NewHostsMiddleware(p.hostsResolver)
	rewriteMw := ditingdns.NewRewriteMiddleware(p.rewriteEngine)
	safeSearchMw := ditingdns.NewSafeSearchMiddleware(p.safeSearchEng)
	privatePTRMw := ditingdns.NewPrivatePTRMiddleware(func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cfg.DNS.BlockPrivatePTR
	}, p.clientResolver)
	unqualifiedMw := ditingdns.NewDynamicUnqualifiedMiddleware(func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cfg.DNS.BlockUnqualified
	})
	filterMw := ditingdns.NewFilterMiddlewareWithGroups(p.filterEngine, p.groupResolver)
	rebindMw := ditingdns.NewRebindMiddleware(rebindMgr, p.cache)
	filterResponseMw := ditingdns.NewFilterResponseMiddleware(p.filterEngine, p.groupResolver, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cfg.Filter.FilterResponse
	}, p.cache)
	dns64Mw := ditingdns.NewDNS64Middleware(p.dns64Synth, p.resolver)
	ecsMw := ditingdns.NewECSMiddleware(p.ecsMgr)
	cacheMw := ditingdns.NewCacheMiddleware(p.cache, p.resolver)
	forwardMw := ditingdns.NewForwardMiddleware(p.resolver)
	// 响应侧复检位于 rebindMw 之后、dns64Mw 之前：
	// 既包裹缓存（缓存命中也复检），又在 DNS64 合成之后获得最终应答，且被 rebindMw 的已阻断判定短路。
	pipeline := ditingdns.NewPipeline(metricsMw, edeMw, echMw, clientMw, accessMw, rateLimitMw, sudnMw, hostsMw, rewriteMw, safeSearchMw, ipv6Mw, privatePTRMw, unqualifiedMw, filterMw, rebindMw, filterResponseMw, dns64Mw, ecsMw, cacheMw, forwardMw)
	udpAddrs, tcpAddrs := p.cfg.DNS.EffectiveListenAddresses()
	serverCfg := ditingdns.ServerConfig{
		UDPAddresses: udpAddrs,
		TCPAddresses: tcpAddrs,
		ReadTimeout:  p.cfg.DNS.ReadTimeout,
		WriteTimeout: p.cfg.DNS.WriteTimeout,
	}

	// 53 端口占用前置检测：若本地监听包含 53 且不可用，立即执行深度冲突诊断
	needs53 := false
	for _, a := range append(serverCfg.UDPAddresses, serverCfg.TCPAddresses...) {
		if strings.HasSuffix(a, ":53") {
			needs53 = true
			break
		}
	}
	if needs53 && p.portChecker != nil && !platform.AreAddressesAvailable(serverCfg.UDPAddresses, serverCfg.TCPAddresses) {
		res, checkErr := p.portChecker.CheckPort53ForAddresses(ctx, serverCfg.UDPAddresses, serverCfg.TCPAddresses)
		if checkErr == nil && res != nil && !res.Available {
			return &platform.PortConflictError{Result: res}
		}
		if checkErr != nil {
			return fallbackPortConflict(nil)
		}
	}

	srv := ditingdns.NewServer(serverCfg, pipeline)
	if err := srv.Start(); err != nil {
		if platform.IsPortBindConflict(err) {
			if p.portChecker != nil {
				if res, checkErr := p.portChecker.CheckPort53ForAddresses(ctx, serverCfg.UDPAddresses, serverCfg.TCPAddresses); checkErr == nil && res != nil {
					return &platform.PortConflictError{Result: res, Err: err}
				}
			}
			return fallbackPortConflict(err)
		}
		return fmt.Errorf("启动 DNS 监听器失败: %w", err)
	}

	p.server = srv
	p.dnsRunning = true
	log.Printf("[DNS] 谛听 DNS 监听器已就绪: %v\n", serverCfg.UDPAddresses)
	return nil
}

// StopDNS 实现 ServiceController 接口
func (p *program) StopDNS(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.dnsRunning {
		return nil
	}

	// 关键保护：若当前处于系统 DNS 接管状态，必须先安全还原网卡，防止系统网络彻底断连
	if p.takeoverMgr != nil && p.takeoverMgr.IsTakeoverActive() {
		log.Println("[DNS] 停止 DNS 监听前检测到接管生效中，自动先还原系统 DNS 接管...")
		if err := p.takeoverMgr.Restore(ctx); err != nil {
			log.Printf("[DNS警告] 自动还原系统 DNS 接管失败: %v\n", err)
		} else if p.ipcServer != nil {
			p.ipcServer.Broadcast(ipc.Event{Type: "takeover", Timestamp: time.Now().UnixMilli(), Data: "disabled"})
		}
	}

	if p.server != nil {
		_ = p.server.Shutdown()
		p.server = nil
	}
	p.dnsRunning = false
	log.Println("[DNS] 谛听 DNS 监听器已停止")
	return nil
}

// EnableTakeover 实现 ServiceController 接口
func (p *program) EnableTakeover(ctx context.Context) error {
	p.mu.Lock()
	dnsRun := p.dnsRunning
	p.mu.Unlock()

	if !dnsRun {
		log.Println("[接管] DNS 服务未处于运行状态，自动先启动本地 DNS 监听器...")
		if err := p.StartDNS(ctx); err != nil {
			var pErr *platform.PortConflictError
			if errors.As(err, &pErr) {
				return pErr
			}
			return fmt.Errorf("接管前启动 DNS 监听器失败: %w", err)
		}
	}

	adapters, err := p.adapterScan.GetActivePhysicalAdapters(ctx)
	if err != nil {
		return fmt.Errorf("枚举活动物理网卡失败: %w", err)
	}
	if len(adapters) == 0 {
		return fmt.Errorf("未检测到有效活动的物理网卡（以太网或 Wi-Fi）")
	}

	if err := p.takeoverMgr.Takeover(ctx, adapters); err != nil {
		return err
	}
	log.Printf("[接管] 已成功接管 %d 个活动物理网卡\n", len(adapters))
	return nil
}

// DisableTakeover 实现 ServiceController 接口
func (p *program) DisableTakeover(ctx context.Context) error {
	if err := p.takeoverMgr.Restore(ctx); err != nil {
		return err
	}
	log.Println("[接管] 已成功还原物理网卡 DNS 设置")
	return nil
}

// GetAdapters 实现 ServiceController 接口
func (p *program) GetAdapters(ctx context.Context) ([]platform.AdapterInfo, error) {
	if p.adapterScan == nil {
		p.adapterScan = newAdapterScanner(platform.NewDefaultExecutor())
	}
	return p.adapterScan.ScanAll(ctx)
}

// GetStatus 实现 ServiceController 接口
func (p *program) GetStatus(ctx context.Context) (*ipc.StatusResponse, error) {
	p.mu.Lock()
	dnsRun := p.dnsRunning
	p.mu.Unlock()

	upstreams := make([]ipc.UpstreamInfo, 0)
	if p.resolver != nil {
		stats := p.resolver.GetProviderStats()
		for _, stat := range stats {
			upstreams = append(upstreams, ipc.UpstreamInfo{
				ID:           stat.ID,
				Protocol:     string(stat.Protocol),
				Server:       stat.Server,
				URL:          stat.URL,
				Active:       stat.IsHealthy,
				SuccessCount: stat.SuccessCount,
				FailureCount: stat.FailureCount,
				AvgLatencyMs: stat.AvgLatencyMs,
			})
		}
	}

	takenAdapters := p.takeoverMgr.GetTakenOverAdapters()
	lanAddrs, _ := platform.GetLANAddresses()
	effUDP, _ := p.cfg.DNS.EffectiveListenAddresses()
	return &ipc.StatusResponse{
		Version:       Version,
		PID:           os.Getpid(),
		UptimeSeconds: int64(time.Since(p.startTime).Seconds()),
		DNS: ipc.DNSStatus{
			Running:           dnsRun,
			ListenAddresses:   effUDP,
			AllowLAN:          p.cfg.DNS.AllowLAN,
			LANAddresses:      lanAddrs,
			Mode:              p.cfg.Upstream.Mode,
			Upstreams:         upstreams,
			Bootstrap:         p.cfg.Upstream.Bootstrap,
			ConditionalRoutes: p.cfg.Upstream.ConditionalRoutes,
			EnableEDE:         p.cfg.DNS.EnableEDE,
			BlockUnqualified:  p.cfg.DNS.BlockUnqualified,
			BlockPrivatePTR:   p.cfg.DNS.BlockPrivatePTR,
			BlockIPv6:         p.cfg.DNS.BlockIPv6,
			BlockECH:          p.cfg.DNS.BlockECH,
			EDNSPadding:       p.cfg.DNS.EDNSPadding,
		},
		Takeover: ipc.TakeoverStatus{
			Active:   p.takeoverMgr.IsTakeoverActive(),
			Adapters: takenAdapters,
		},
		Metrics:    p.metrics.Snapshot(),
		Cache:      p.getCacheStatsSnapshot(),
		Filter:     p.getFilterStatsSnapshot(),
		Hosts:      p.cfg.Hosts,
		Rebind:     p.cfg.Rebind,
		Access:     p.cfg.Access,
		RateLimit:  p.cfg.RateLimit,
		DNS64:      p.cfg.DNS64,
		Rewrite:    p.cfg.Rewrite,
		SafeSearch: p.cfg.SafeSearch,
		ECS:        p.cfg.ECS,
		Clients:    p.getClientsSnapshot(),
	}, nil
}

func (p *program) getCacheStatsSnapshot() core.CacheStats {
	if p.cache != nil {
		return p.cache.Stats()
	}
	return core.CacheStats{Enabled: false}
}

func (p *program) getFilterStatsSnapshot() core.FilterStats {
	if p.filterEngine != nil {
		return p.filterEngine.GetStats()
	}
	return core.FilterStats{Enabled: false}
}

func (p *program) getClientsSnapshot() []core.ClientInfo {
	if p.clientResolver != nil {
		return p.clientResolver.GetClients()
	}
	return []core.ClientInfo{}
}
