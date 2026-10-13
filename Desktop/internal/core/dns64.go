package core

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// WellKnownDNS64Prefix RFC 6052 规定的标准 DNS64 / NAT64 知名 IPv6 前缀
const WellKnownDNS64Prefix = "64:ff9b::/96"

// DNS64Config DNS64 IPv6-only 网络过渡合成配置。
// 借鉴自 blocky (dns64_resolver.go)、AdGuard Home (dns64.go) 与 Technitium DNS (RFC 6052 / RFC 6147)。
type DNS64Config struct {
	Enabled  bool     `json:"enabled"`  // 是否启用 DNS64 合成
	Prefixes []string `json:"prefixes"` // 用于合成 AAAA 记录的 IPv6 前缀列表（为空时默认使用 64:ff9b::/96）
}

// DefaultDNS64Config 返回默认 DNS64 配置
func DefaultDNS64Config() DNS64Config {
	return DNS64Config{
		Enabled:  false,
		Prefixes: []string{WellKnownDNS64Prefix},
	}
}

// DNS64Synthesizer 实现 RFC 6052 与 RFC 6147 的 AAAA 记录合成引擎
type DNS64Synthesizer struct {
	mu           sync.RWMutex
	enabled      bool
	prefixes     []*net.IPNet
	exclusionSet []*net.IPNet
}

// NewDNS64Synthesizer 创建 DNS64 合成引擎
func NewDNS64Synthesizer(cfg DNS64Config) (*DNS64Synthesizer, error) {
	s := &DNS64Synthesizer{
		enabled: cfg.Enabled,
	}

	// 初始化排除网段 (RFC 6147 Section 5.1.4: 避免回环、链路本地以及重复合成循环)
	defaultExclusions := []string{
		"::ffff:0:0/96", // IPv4-mapped
		"::1/128",       // Loopback
		"fe80::/10",     // Link-local
	}
	for _, excl := range defaultExclusions {
		_, ipNet, err := net.ParseCIDR(excl)
		if err == nil {
			s.exclusionSet = append(s.exclusionSet, ipNet)
		}
	}

	prefixes := cfg.Prefixes
	if len(prefixes) == 0 {
		prefixes = []string{WellKnownDNS64Prefix}
	}

	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(p)
		if err != nil {
			return nil, fmt.Errorf("解析 DNS64 前缀 %q 失败: %w", p, err)
		}
		// 校验前缀是否为 IPv6 且掩码支持 /96
		ones, bits := ipNet.Mask.Size()
		if bits != 128 || ones != 96 {
			return nil, fmt.Errorf("当前实现遵循 RFC 6052 标准 /96 前缀规范，不支持 %s", p)
		}
		s.prefixes = append(s.prefixes, ipNet)
		// 按照 RFC 规范将配置的自身前缀加入排除集，防止多次级联递归合成
		s.exclusionSet = append(s.exclusionSet, ipNet)
	}

	return s, nil
}

// IsEnabled 判断 DNS64 功能是否开启
func (s *DNS64Synthesizer) IsEnabled() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// MapIPv4ToIPv6 按照 RFC 6052 第 2.2 节将 IPv4 地址映射为 /96 前缀的 IPv6 地址
func (s *DNS64Synthesizer) MapIPv4ToIPv6(v4 net.IP) net.IP {
	v4 = v4.To4()
	if v4 == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var prefix *net.IPNet
	if len(s.prefixes) > 0 {
		prefix = s.prefixes[0]
	} else {
		_, prefix, _ = net.ParseCIDR(WellKnownDNS64Prefix)
	}

	v6 := make(net.IP, 16)
	// 前 12 字节来自 prefix
	copy(v6[:12], prefix.IP[:12])
	// 尾部 4 字节填充 IPv4
	copy(v6[12:], v4)
	return v6
}

// IsExcluded 判断 IPv6 地址是否在 RFC 6147 排除名单中
func (s *DNS64Synthesizer) IsExcluded(ip net.IP) bool {
	if ip == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, subnet := range s.exclusionSet {
		if subnet.Contains(ip) {
			return true
		}
	}
	return false
}

// SynthesizeFromA 根据 A 记录切片合成 AAAA 记录列表
func (s *DNS64Synthesizer) SynthesizeFromA(aRecords []*dns.A) []*dns.AAAA {
	if !s.IsEnabled() || len(aRecords) == 0 {
		return nil
	}

	var res []*dns.AAAA
	for _, a := range aRecords {
		if a == nil || a.A == nil {
			continue
		}
		v6 := s.MapIPv4ToIPv6(a.A)
		if v6 == nil {
			continue
		}
		aaaa := &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   a.Hdr.Name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    a.Hdr.Ttl,
			},
			AAAA: v6,
		}
		res = append(res, aaaa)
	}
	return res
}
