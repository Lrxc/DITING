package core

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

// DefaultRewriteTTL 默认重写记录生存期（秒）
const DefaultRewriteTTL uint32 = 60

// RewriteRule 定义单条 DNS 重写/伪装规则
type RewriteRule struct {
	Domain string   `json:"domain"`          // 匹配域名，支持泛域名如 "*.lan" 或精准匹配 "myhost.local"
	IPs    []string `json:"ips,omitempty"`   // A (IPv4) 或 AAAA (IPv6) 解析目标
	CNAME  string   `json:"cname,omitempty"` // CNAME 目标别名
	TXT    []string `json:"txt,omitempty"`   // TXT 记录内容列表
	TTL    uint32   `json:"ttl,omitempty"`   // 生存期（秒），为 0 时使用系统默认 TTL
}

// RewriteConfig DNS 重写配置
type RewriteConfig struct {
	Enabled bool          `json:"enabled"` // 是否启用自定义 DNS 重写与伪装
	TTL     uint32        `json:"ttl"`     // 默认 TTL，默认 60s
	Rules   []RewriteRule `json:"rules"`   // 规则列表
}

// DefaultRewriteConfig 返回默认配置
func DefaultRewriteConfig() RewriteConfig {
	return RewriteConfig{
		Enabled: true,
		TTL:     DefaultRewriteTTL,
		Rules:   []RewriteRule{},
	}
}

// NormalizeRewriteConfig 校验与修补重写配置默认值
func NormalizeRewriteConfig(cfg *RewriteConfig) {
	if cfg == nil {
		return
	}
	if cfg.TTL == 0 {
		cfg.TTL = DefaultRewriteTTL
	}
	if cfg.Rules == nil {
		cfg.Rules = []RewriteRule{}
	}
}

// compiledRule 内部编译后的快速检索结构体
type compiledRule struct {
	domain     string
	isWildcard bool
	suffix     string // 去除 "*." 后的后缀，全小写，不带前后点
	v4List     []net.IP
	v6List     []net.IP
	cname      string // 标准 FQDN (以点结尾)
	txt        []string
	ttl        uint32
}

// RewriteEngine 自定义 DNS 重写与伪装引擎
// 深度借鉴自 AdGuard Home (dnsrewrite.go)、Blocky (custom_dns_resolver.go) 与 dnscrypt-proxy (plugin_cloak.go)
type RewriteEngine struct {
	mu            sync.RWMutex
	enabled       bool
	defaultTTL    uint32
	exactRules    map[string]*compiledRule // domain -> rule
	wildcardRules []*compiledRule          // 泛域名规则列表（按后缀长度从长到短优先匹配）
	ptrMap        map[string][]string      // ip.String() -> []FQDN domain (支持多别名反向 PTR 解析)
	originalRules []RewriteRule
}

// NewRewriteEngine 创建并初始化 DNS 重写引擎
func NewRewriteEngine(cfg RewriteConfig) *RewriteEngine {
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = DefaultRewriteTTL
	}
	engine := &RewriteEngine{
		enabled:       cfg.Enabled,
		defaultTTL:    ttl,
		exactRules:    make(map[string]*compiledRule),
		wildcardRules: make([]*compiledRule, 0),
		ptrMap:        make(map[string][]string),
		originalRules: make([]RewriteRule, 0),
	}
	for _, rule := range cfg.Rules {
		engine.addRuleInternal(rule)
	}
	return engine
}

// normalizeRewriteDomain 规范化域名字符串为无前后点的小写形式
func normalizeRewriteDomain(domain string) string {
	d := strings.TrimSpace(strings.ToLower(domain))
	return strings.Trim(d, ".")
}

// addRuleInternal 将一条规则编译并加入索引
func (e *RewriteEngine) addRuleInternal(rule RewriteRule) {
	norm := normalizeRewriteDomain(rule.Domain)
	if norm == "" {
		return
	}

	ttl := rule.TTL
	if ttl == 0 {
		ttl = e.defaultTTL
	}

	c := &compiledRule{
		domain: norm,
		ttl:    ttl,
		txt:    rule.TXT,
	}

	if strings.HasPrefix(norm, "*.") {
		c.isWildcard = true
		c.suffix = strings.TrimPrefix(norm, "*.")
	}

	if rule.CNAME != "" {
		cname := strings.TrimSpace(rule.CNAME)
		if !strings.HasSuffix(cname, ".") {
			cname += "."
		}
		c.cname = cname
	}

	for _, ipStr := range rule.IPs {
		ip := net.ParseIP(strings.TrimSpace(ipStr))
		if ip == nil {
			continue
		}
		if v4 := ip.To4(); v4 != nil {
			c.v4List = append(c.v4List, v4)
			// 自动建立反向 PTR 映射（精准域名优先，支持多别名）
			if !c.isWildcard {
				e.appendPTR(v4.String(), norm+".")
			}
		} else {
			c.v6List = append(c.v6List, ip)
			if !c.isWildcard {
				e.appendPTR(ip.String(), norm+".")
			}
		}
	}

	e.originalRules = append(e.originalRules, rule)

	if c.isWildcard {
		e.wildcardRules = append(e.wildcardRules, c)
		// 按后缀长度降序排序，长后缀更精确优先匹配
		sort.SliceStable(e.wildcardRules, func(i, j int) bool {
			return len(e.wildcardRules[i].suffix) > len(e.wildcardRules[j].suffix)
		})
	} else {
		e.exactRules[norm] = c
	}
}

func (e *RewriteEngine) appendPTR(ipStr, fqdn string) {
	for _, exist := range e.ptrMap[ipStr] {
		if exist == fqdn {
			return
		}
	}
	e.ptrMap[ipStr] = append(e.ptrMap[ipStr], fqdn)
}

func (e *RewriteEngine) findRule(norm string) (*compiledRule, bool) {
	if cr, ok := e.exactRules[norm]; ok {
		return cr, true
	}
	for _, w := range e.wildcardRules {
		if norm == w.suffix || strings.HasSuffix(norm, "."+w.suffix) {
			return w, true
		}
	}
	return nil, false
}

func (e *RewriteEngine) chaseCNAME(targetFQDN string, qtype uint16, visited map[string]bool) []dns.RR {
	if len(visited) > 8 {
		return nil
	}
	targetNorm := normalizeRewriteDomain(targetFQDN)
	if targetNorm == "" || visited[targetNorm] {
		return nil
	}
	visited[targetNorm] = true

	rule, ok := e.findRule(targetNorm)
	if !ok {
		return nil
	}

	var chained []dns.RR
	switch qtype {
	case dns.TypeA:
		if len(rule.v4List) > 0 {
			for _, ip := range rule.v4List {
				chained = append(chained, &dns.A{
					Hdr: dns.RR_Header{Name: dns.Fqdn(targetFQDN), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: rule.ttl},
					A:   ip,
				})
			}
		} else if rule.cname != "" {
			chained = append(chained, &dns.CNAME{
				Hdr:    dns.RR_Header{Name: dns.Fqdn(targetFQDN), Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: rule.ttl},
				Target: rule.cname,
			})
			chained = append(chained, e.chaseCNAME(rule.cname, qtype, visited)...)
		}
	case dns.TypeAAAA:
		if len(rule.v6List) > 0 {
			for _, ip := range rule.v6List {
				chained = append(chained, &dns.AAAA{
					Hdr:  dns.RR_Header{Name: dns.Fqdn(targetFQDN), Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: rule.ttl},
					AAAA: ip,
				})
			}
		} else if rule.cname != "" {
			chained = append(chained, &dns.CNAME{
				Hdr:    dns.RR_Header{Name: dns.Fqdn(targetFQDN), Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: rule.ttl},
				Target: rule.cname,
			})
			chained = append(chained, e.chaseCNAME(rule.cname, qtype, visited)...)
		}
	}
	return chained
}

// Lookup 查找匹配的重写规则
func (e *RewriteEngine) Lookup(domain string, qtype uint16) (answers []dns.RR, matched bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled {
		return nil, false
	}

	norm := normalizeRewriteDomain(domain)
	if norm == "" {
		return nil, false
	}

	cr, ok := e.findRule(norm)
	if !ok {
		return nil, false
	}

	// 匹配成功，标记为已被重写规则捕获
	matched = true
	fqdn := domain
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}

	switch qtype {
	case dns.TypeA:
		if len(cr.v4List) > 0 {
			for _, ip := range cr.v4List {
				answers = append(answers, &dns.A{
					Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: cr.ttl},
					A:   ip,
				})
			}
		} else if cr.cname != "" {
			answers = append(answers, &dns.CNAME{
				Hdr:    dns.RR_Header{Name: fqdn, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: cr.ttl},
				Target: cr.cname,
			})
			answers = append(answers, e.chaseCNAME(cr.cname, dns.TypeA, map[string]bool{norm: true})...)
		}
	case dns.TypeAAAA:
		if len(cr.v6List) > 0 {
			for _, ip := range cr.v6List {
				answers = append(answers, &dns.AAAA{
					Hdr:  dns.RR_Header{Name: fqdn, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: cr.ttl},
					AAAA: ip,
				})
			}
		} else if cr.cname != "" {
			answers = append(answers, &dns.CNAME{
				Hdr:    dns.RR_Header{Name: fqdn, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: cr.ttl},
				Target: cr.cname,
			})
			answers = append(answers, e.chaseCNAME(cr.cname, dns.TypeAAAA, map[string]bool{norm: true})...)
		}
	case dns.TypeCNAME:
		if cr.cname != "" {
			answers = append(answers, &dns.CNAME{
				Hdr:    dns.RR_Header{Name: fqdn, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: cr.ttl},
				Target: cr.cname,
			})
		}
	case dns.TypeTXT:
		if len(cr.txt) > 0 {
			answers = append(answers, &dns.TXT{
				Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: cr.ttl},
				Txt: cr.txt,
			})
		}
	}

	return answers, true
}

// LookupPTR 根据 IP 反向解析域名（O(1) 精确反查，支持多域名别名）
func (e *RewriteEngine) LookupPTR(ptrQuery string) (answers []dns.RR, matched bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled {
		return nil, false
	}

	ip := ParseIPFromArpa(ptrQuery)
	if ip == nil {
		return nil, false
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	domains, ok := e.ptrMap[ip.String()]
	if !ok || len(domains) == 0 {
		return nil, false
	}

	fqdn := dns.Fqdn(ptrQuery)
	for _, domain := range domains {
		answers = append(answers, &dns.PTR{
			Hdr: dns.RR_Header{Name: fqdn, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: e.defaultTTL},
			Ptr: domain,
		})
	}

	return answers, true
}

// LookupReverse 实现 ReverseLookuper 接口，根据 IP 反向查询重写匹配的域名
func (e *RewriteEngine) LookupReverse(ip net.IP) string {
	if ip == nil {
		return ""
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.enabled {
		return ""
	}
	if domains, ok := e.ptrMap[ip.String()]; ok && len(domains) > 0 {
		return strings.TrimSuffix(domains[0], ".")
	}
	return ""
}

// AddRule 动态添加单条规则
func (e *RewriteEngine) AddRule(rule RewriteRule) error {
	norm := normalizeRewriteDomain(rule.Domain)
	if norm == "" {
		return fmt.Errorf("无效的重写域名")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否已有同名规则，有则替换
	newRules := make([]RewriteRule, 0, len(e.originalRules)+1)
	for _, r := range e.originalRules {
		if normalizeRewriteDomain(r.Domain) != norm {
			newRules = append(newRules, r)
		}
	}
	newRules = append(newRules, rule)

	e.rebuildRules(newRules)
	return nil
}

// DeleteRule 根据域名删除规则
func (e *RewriteEngine) DeleteRule(domain string) bool {
	norm := normalizeRewriteDomain(domain)
	if norm == "" {
		return false
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	found := false
	newRules := make([]RewriteRule, 0, len(e.originalRules))
	for _, r := range e.originalRules {
		if normalizeRewriteDomain(r.Domain) == norm {
			found = true
			continue
		}
		newRules = append(newRules, r)
	}

	if found {
		e.rebuildRules(newRules)
	}
	return found
}

// rebuildRules 在写锁保护下根据原规则切片全量重建索引
func (e *RewriteEngine) rebuildRules(rules []RewriteRule) {
	e.exactRules = make(map[string]*compiledRule)
	e.wildcardRules = make([]*compiledRule, 0)
	e.ptrMap = make(map[string][]string)
	e.originalRules = make([]RewriteRule, 0, len(rules))

	for _, r := range rules {
		e.addRuleInternal(r)
	}
}

// Configure 整体更新重写配置
func (e *RewriteEngine) Configure(cfg RewriteConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.enabled = cfg.Enabled
	if cfg.TTL > 0 {
		e.defaultTTL = cfg.TTL
	}
	e.rebuildRules(cfg.Rules)
}

// GetRules 获取当前所有重写规则
func (e *RewriteEngine) GetRules() []RewriteRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	res := make([]RewriteRule, len(e.originalRules))
	copy(res, e.originalRules)
	return res
}

// IsEnabled 检查重写引擎是否启用
func (e *RewriteEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}
