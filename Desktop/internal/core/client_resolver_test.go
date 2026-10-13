package core

import (
	"context"
	"net"
	"testing"
	"time"
)

type dummyLookuper struct {
	names map[string]string
}

func (d *dummyLookuper) LookupReverse(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return d.names[ip.String()]
}

func TestClientResolver_Identification(t *testing.T) {
	dummy := &dummyLookuper{
		names: map[string]string{
			"192.168.1.50": "router.local",
		},
	}

	cfg := ClientConfig{
		Enabled:    true,
		EnableRDNS: true,
		CacheTTL:   time.Hour,
		Aliases: []ClientAlias{
			{IP: "192.168.1.100", Name: "Boss-iPhone"},
		},
	}

	resolver := NewClientResolver(cfg, dummy)

	// 自定义 mock rDNS
	resolver.lookupAddr = func(ctx context.Context, addr string) ([]string, error) {
		if addr == "192.168.1.200" {
			return []string{"printer.corp.lan."}, nil
		}
		return nil, net.UnknownNetworkError("not found")
	}

	// 1. 回环地址
	name := resolver.IdentifyClient(net.ParseIP("127.0.0.1"))
	if name != "localhost" {
		t.Fatalf("expected localhost, got %s", name)
	}

	nameV6 := resolver.IdentifyClient(net.ParseIP("::1"))
	if nameV6 != "localhost" {
		t.Fatalf("expected localhost, got %s", nameV6)
	}

	// 2. 静态别名
	name = resolver.IdentifyClient(net.ParseIP("192.168.1.100"))
	if name != "Boss-iPhone" {
		t.Fatalf("expected Boss-iPhone, got %s", name)
	}

	// 3. 本地 ReverseLookuper (Hosts/Rewrite)
	name = resolver.IdentifyClient(net.ParseIP("192.168.1.50"))
	if name != "router.local" {
		t.Fatalf("expected router.local, got %s", name)
	}

	// 4. 异步 rDNS 解析 (第一次调用触发异步解析并返回空或未缓存值，短暂停留后缓存生效)
	_ = resolver.IdentifyClient(net.ParseIP("192.168.1.200"))
	// 等待后台 goroutine 完成
	time.Sleep(50 * time.Millisecond)
	name = resolver.IdentifyClient(net.ParseIP("192.168.1.200"))
	if name != "printer.corp.lan" {
		t.Fatalf("expected printer.corp.lan from rDNS, got %s", name)
	}

	// 5. 客户端统计检查
	clients := resolver.GetClients()
	if len(clients) < 4 {
		t.Fatalf("expected at least 4 recorded clients, got %d", len(clients))
	}

	// 6. 动态更新别名
	resolver.SetAlias("192.168.1.200", "Office-Printer")
	name = resolver.IdentifyClient(net.ParseIP("192.168.1.200"))
	if name != "Office-Printer" {
		t.Fatalf("expected alias override Office-Printer, got %s", name)
	}
}
