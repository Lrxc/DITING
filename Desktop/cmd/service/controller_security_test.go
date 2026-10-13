package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/haoze-evolluling/diting/desktop/internal/config"
	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/haoze-evolluling/diting/desktop/internal/ipc"
)

func TestController_SecurityAndTransitionConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	cfg := config.DefaultConfig()
	if err := config.SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("save config failed: %v", err)
	}

	ac, _ := core.NewAccessControl(cfg.Access)
	limiter := core.NewRateLimiter(context.Background(), cfg.RateLimit)
	synth, _ := core.NewDNS64Synthesizer(cfg.DNS64)

	prg := &program{
		cfg:           cfg,
		configPath:    cfgPath,
		accessControl: ac,
		rateLimiter:   limiter,
		dns64Synth:    synth,
	}

	ctx := context.Background()

	// 1. 测试 ConfigureAccess
	accessReq := ipc.ConfigureAccessRequest{
		AllowedClients: []string{"192.168.1.0/24", "10.0.0.1"},
		BlockedClients: []string{"192.168.1.250"},
	}
	if err := prg.ConfigureAccess(ctx, accessReq); err != nil {
		t.Fatalf("ConfigureAccess failed: %v", err)
	}
	if len(prg.cfg.Access.AllowedClients) != 2 || len(prg.cfg.Access.BlockedClients) != 1 {
		t.Errorf("unexpected access config: %+v", prg.cfg.Access)
	}

	// 2. 测试 ConfigureRateLimit
	rateReq := ipc.ConfigureRateLimitRequest{
		Enabled:    true,
		Rate:       20.0,
		Burst:      40,
		IPv4Prefix: 24,
		IPv6Prefix: 56,
		ExemptIPs:  []string{"127.0.0.1", "192.168.1.1"},
	}
	if err := prg.ConfigureRateLimit(ctx, rateReq); err != nil {
		t.Fatalf("ConfigureRateLimit failed: %v", err)
	}
	if !prg.cfg.RateLimit.Enabled || prg.cfg.RateLimit.Rate != 20.0 || prg.cfg.RateLimit.Burst != 40 {
		t.Errorf("unexpected rate limit config: %+v", prg.cfg.RateLimit)
	}

	// 3. 测试 ConfigureDNS64
	dns64Req := ipc.ConfigureDNS64Request{
		Enabled:  true,
		Prefixes: []string{"64:ff9b::/96", "2001:db8:1234::/64"},
	}
	if err := prg.ConfigureDNS64(ctx, dns64Req); err != nil {
		t.Fatalf("ConfigureDNS64 failed: %v", err)
	}
	if !prg.cfg.DNS64.Enabled || len(prg.cfg.DNS64.Prefixes) != 2 {
		t.Errorf("unexpected dns64 config: %+v", prg.cfg.DNS64)
	}

	// 4. 测试 ConfigureBlockUnqualified
	unqualReq := ipc.ConfigureBlockUnqualifiedRequest{
		BlockUnqualified: true,
	}
	if err := prg.ConfigureBlockUnqualified(ctx, unqualReq); err != nil {
		t.Fatalf("ConfigureBlockUnqualified failed: %v", err)
	}
	if !prg.cfg.DNS.BlockUnqualified {
		t.Errorf("expected BlockUnqualified to be true")
	}

	// 验证磁盘配置文件持久化是否正确写入
	reloaded, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("reload config failed: %v", err)
	}
	if !reloaded.DNS.BlockUnqualified || !reloaded.DNS64.Enabled || !reloaded.RateLimit.Enabled {
		t.Errorf("persisted config mismatch: %+v", reloaded)
	}
	_ = os.RemoveAll(tmpDir)
}
