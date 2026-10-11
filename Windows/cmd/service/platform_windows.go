//go:build windows

package main

import (
	"context"

	"github.com/haoze-evolluling/diting/windows/internal/platform"
	"github.com/haoze-evolluling/diting/windows/internal/platform/windows"
)

// newDNSManager 构造当前平台（Windows）的 DNS 接管管理器。
func newDNSManager(exec platform.CommandExecutor, store platform.StateStore, version string) platform.DNSManager {
	return windows.NewDNSManager(exec, store, version)
}

// newAdapterScanner 构造当前平台（Windows）的网卡枚举扫描器。
func newAdapterScanner(exec platform.CommandExecutor) platform.AdapterScanner {
	return windows.NewAdapterScanner(exec)
}

// newPortChecker 构造当前平台（Windows）的端口冲突探测器。
func newPortChecker(exec platform.CommandExecutor) platform.PortChecker {
	return windows.NewPortChecker(exec)
}

// firewallCheckPort53 检测 53 端口防火墙放行规则是否生效。
func firewallCheckPort53(ctx context.Context) (bool, error) {
	return windows.CheckFirewallPort53(ctx, platform.NewDefaultExecutor())
}

// firewallConfigurePort53 配置 53 端口防火墙放行规则。
func firewallConfigurePort53(ctx context.Context, enable bool) error {
	return windows.ConfigureFirewallPort53(ctx, platform.NewDefaultExecutor(), enable)
}

// firewallCheckWeb 检测 Web 管理端口防火墙放行规则是否生效。
func firewallCheckWeb(ctx context.Context) (bool, error) {
	return windows.CheckFirewallPortWeb(ctx, platform.NewDefaultExecutor())
}

// firewallConfigureWeb 配置 Web 管理端口防火墙放行规则。
func firewallConfigureWeb(ctx context.Context, port int, enable bool) error {
	return windows.ConfigureFirewallPortWeb(ctx, platform.NewDefaultExecutor(), port, enable)
}
