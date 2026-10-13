package core

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// DefaultHostsTTL 本地 hosts 映射记录的默认 TTL 秒数
const DefaultHostsTTL uint32 = 60

// HostsConfig Hosts 解析器配置
type HostsConfig struct {
	Enabled       bool     `json:"enabled"`
	UseSystemHost bool     `json:"useSystemHost"` // 是否加载操作系统默认 hosts 文件
	CustomFiles   []string `json:"customFiles"`   // 自定义 hosts 文件路径列表
	TTL           uint32   `json:"ttl"`           // 返回记录的 TTL，默认 60
}

// DefaultHostsConfig 默认 hosts 配置
func DefaultHostsConfig() HostsConfig {
	return HostsConfig{
		Enabled:       true,
		UseSystemHost: true,
		CustomFiles:   []string{},
		TTL:           DefaultHostsTTL,
	}
}

// SystemHostsPath 获取当前操作系统的主机 hosts 文件绝对路径
func SystemHostsPath() string {
	if runtime.GOOS == "windows" {
		sysRoot := os.Getenv("SystemRoot")
		if sysRoot == "" {
			sysRoot = `C:\Windows`
		}
		return filepath.Join(sysRoot, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// HostsResolver 本地系统与自定义 Hosts 文件解析器
// 借鉴自 blocky (hosts_file_resolver.go) 与 AdGuard Home
type HostsResolver struct {
	mu          sync.RWMutex
	v4Map       map[string][]net.IP   // domain -> []IPv4
	v6Map       map[string][]net.IP   // domain -> []IPv6
	ptrMap      map[string][]string // ip.String() -> []hostname (支持多别名)
	ttl         uint32
	loadedFiles []string
}

// NewHostsResolver 创建并初始化 Hosts 解析器
func NewHostsResolver(cfg HostsConfig) (*HostsResolver, error) {
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = DefaultHostsTTL
	}
	r := &HostsResolver{
		v4Map:  make(map[string][]net.IP),
		v6Map:  make(map[string][]net.IP),
		ptrMap: make(map[string][]string),
		ttl:    ttl,
	}

	if cfg.Enabled {
		if cfg.UseSystemHost {
			sysPath := SystemHostsPath()
			_ = r.LoadFile(sysPath)
		}
		for _, f := range cfg.CustomFiles {
			_ = r.LoadFile(f)
		}
	}
	return r, nil
}

// Reload 重新加载所有已注册的 hosts 文件
func (r *HostsResolver) Reload() error {
	r.mu.Lock()
	files := make([]string, len(r.loadedFiles))
	copy(files, r.loadedFiles)
	r.v4Map = make(map[string][]net.IP)
	r.v6Map = make(map[string][]net.IP)
	r.ptrMap = make(map[string][]string)
	r.loadedFiles = nil
	r.mu.Unlock()

	var lastErr error
	for _, f := range files {
		if err := r.LoadFile(f); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// LoadFile 加载并解析指定的 hosts 文件内容
func (r *HostsResolver) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r.mu.Lock()
	defer r.mu.Unlock()
	r.parseReader(f)
	r.loadedFiles = append(r.loadedFiles, path)
	return nil
}

// LoadReader 从指定 Reader 加载 hosts 格式文本
func (r *HostsResolver) LoadReader(rd io.Reader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parseReader(rd)
}

func (r *HostsResolver) parseReader(rd io.Reader) {
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 移除行内注释
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// 剥离 Windows/Linux 接口 Zone Scope 标识（如 fe80::1%eth0）
		rawIP := fields[0]
		if idx := strings.Index(rawIP, "%"); idx != -1 {
			rawIP = rawIP[:idx]
		}

		ip := net.ParseIP(rawIP)
		if ip == nil {
			continue
		}

		isV4 := (ip.To4() != nil)
		ipStr := ip.String()

		for _, rawHost := range fields[1:] {
			host := strings.ToLower(strings.Trim(rawHost, "."))
			if host == "" {
				continue
			}

			if isV4 {
				v4 := ip.To4()
				if !containsIP(r.v4Map[host], v4) {
					r.v4Map[host] = append(r.v4Map[host], v4)
				}
			} else {
				if !containsIP(r.v6Map[host], ip) {
					r.v6Map[host] = append(r.v6Map[host], ip)
				}
			}

			// 记录反向 PTR 查找（支持多别名映射）
			if !containsString(r.ptrMap[ipStr], host) {
				r.ptrMap[ipStr] = append(r.ptrMap[ipStr], host)
			}
		}
	}
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func containsIP(list []net.IP, target net.IP) bool {
	for _, ip := range list {
		if ip.Equal(target) {
			return true
		}
	}
	return false
}

// HasDomain 检查本地 hosts 是否收录了指定域名（用于判定是否可提供权威 NODATA）
func (r *HostsResolver) HasDomain(domain string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	host := strings.ToLower(strings.Trim(domain, "."))
	_, inV4 := r.v4Map[host]
	_, inV6 := r.v6Map[host]
	return inV4 || inV6
}

// Lookup 正向解析 domain
func (r *HostsResolver) Lookup(domain string, qtype uint16) []dns.RR {
	r.mu.RLock()
	defer r.mu.RUnlock()

	host := strings.ToLower(strings.Trim(domain, "."))
	fqdn := dns.Fqdn(domain)
	var records []dns.RR

	switch qtype {
	case dns.TypeA:
		if ips, ok := r.v4Map[host]; ok {
			for _, ip := range ips {
				records = append(records, &dns.A{
					Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: r.ttl},
					A:   ip,
				})
			}
		}
	case dns.TypeAAAA:
		if ips, ok := r.v6Map[host]; ok {
			for _, ip := range ips {
				records = append(records, &dns.AAAA{
					Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: r.ttl},
					AAAA: ip,
				})
			}
		}
	}
	return records
}

// LookupPTR 反向解析 IP 对应的 PTR 记录（返回所有关联主机名别名）
func (r *HostsResolver) LookupPTR(arpaName string) []dns.RR {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ip := ParseIPFromArpa(arpaName)
	if ip == nil {
		return nil
	}

	hosts, ok := r.ptrMap[ip.String()]
	if !ok || len(hosts) == 0 {
		return nil
	}

	fqdn := dns.Fqdn(arpaName)
	records := make([]dns.RR, 0, len(hosts))
	for _, host := range hosts {
		records = append(records, &dns.PTR{
			Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: r.ttl},
			Ptr: dns.Fqdn(host),
		})
	}
	return records
}

func parseHexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

// ParseIPFromArpa 从 .in-addr.arpa 或 .ip6.arpa 解析出原生 IP
func ParseIPFromArpa(arpa string) net.IP {
	arpa = strings.ToLower(strings.Trim(arpa, "."))
	if strings.HasSuffix(arpa, ".in-addr.arpa") {
		prefix := strings.TrimSuffix(arpa, ".in-addr.arpa")
		parts := strings.Split(prefix, ".")
		if len(parts) != 4 {
			return nil
		}
		// 逆序还原 IPv4
		return net.ParseIP(fmt.Sprintf("%s.%s.%s.%s", parts[3], parts[2], parts[1], parts[0]))
	}

	if strings.HasSuffix(arpa, ".ip6.arpa") {
		prefix := strings.TrimSuffix(arpa, ".ip6.arpa")
		hexs := strings.Split(prefix, ".")
		if len(hexs) != 32 {
			return nil
		}
		// 逆序组装 16 字节
		var ipBytes [16]byte
		for i := 0; i < 16; i++ {
			if len(hexs[i*2]) != 1 || len(hexs[i*2+1]) != 1 {
				return nil
			}
			low, ok1 := parseHexNibble(hexs[i*2][0])
			high, ok2 := parseHexNibble(hexs[i*2+1][0])
			if !ok1 || !ok2 {
				return nil
			}
			ipBytes[15-i] = (high << 4) | low
		}
		return net.IP(ipBytes[:])
	}
	return nil
}
