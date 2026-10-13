package main

import (
	"context"
	"fmt"
	"log"

	"github.com/haoze-evolluling/diting/desktop/internal/config"
	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/haoze-evolluling/diting/desktop/internal/ipc"
)

// ConfigureAccess 实现 ServiceController 接口
func (p *program) ConfigureAccess(ctx context.Context, req ipc.ConfigureAccessRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := core.AccessControlConfig{
		AllowedClients: req.AllowedClients,
		BlockedClients: req.BlockedClients,
	}

	if p.accessControl != nil {
		if err := p.accessControl.Update(cfg); err != nil {
			return fmt.Errorf("更新客户端访问控制失败: %w", err)
		}
	}

	p.cfg.Access = cfg
	if p.configPath != "" {
		if err := config.SaveConfig(p.configPath, p.cfg); err != nil {
			log.Printf("[配置] 持久化 Access 配置失败: %v\n", err)
		}
	}
	return nil
}

// ConfigureRateLimit 实现 ServiceController 接口
func (p *program) ConfigureRateLimit(ctx context.Context, req ipc.ConfigureRateLimitRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := core.RateLimitConfig{
		Enabled:    req.Enabled,
		Rate:       req.Rate,
		Burst:      req.Burst,
		IPv4Prefix: req.IPv4Prefix,
		IPv6Prefix: req.IPv6Prefix,
		ExemptIPs:  req.ExemptIPs,
	}

	if p.rateLimiter != nil {
		p.rateLimiter.UpdateConfig(cfg)
	}

	p.cfg.RateLimit = cfg
	if p.configPath != "" {
		if err := config.SaveConfig(p.configPath, p.cfg); err != nil {
			log.Printf("[配置] 持久化 RateLimit 配置失败: %v\n", err)
		}
	}
	return nil
}

// ConfigureDNS64 实现 ServiceController 接口
func (p *program) ConfigureDNS64(ctx context.Context, req ipc.ConfigureDNS64Request) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := core.DNS64Config{
		Enabled:  req.Enabled,
		Prefixes: req.Prefixes,
	}

	if p.dns64Synth != nil {
		if err := p.dns64Synth.UpdateConfig(cfg); err != nil {
			return fmt.Errorf("更新 DNS64 配置失败: %w", err)
		}
	}

	p.cfg.DNS64 = cfg
	if p.configPath != "" {
		if err := config.SaveConfig(p.configPath, p.cfg); err != nil {
			log.Printf("[配置] 持久化 DNS64 配置失败: %v\n", err)
		}
	}
	return nil
}

// ConfigureBlockUnqualified 实现 ServiceController 接口
func (p *program) ConfigureBlockUnqualified(ctx context.Context, req ipc.ConfigureBlockUnqualifiedRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cfg.DNS.BlockUnqualified = req.BlockUnqualified
	if p.configPath != "" {
		if err := config.SaveConfig(p.configPath, p.cfg); err != nil {
			log.Printf("[配置] 持久化 BlockUnqualified 配置失败: %v\n", err)
		}
	}
	return nil
}
