package core

import (
	"log"

	"github.com/haoze-evolluling/diting/desktop/internal/platform"
)

const (
	// BlockModeNullIP 返回空 IP (0.0.0.0 / ::)
	BlockModeNullIP = "null_ip"
	// BlockModeNXDOMAIN 返回 NXDOMAIN (Name Error)
	BlockModeNXDOMAIN = "nxdomain"
	// BlockModeRefused 返回 REFUSED
	BlockModeRefused = "refused"

	// DefaultBlockingIPv4 默认 IPv4 拦截响应地址
	DefaultBlockingIPv4 = "0.0.0.0"
	// DefaultBlockingIPv6 默认 IPv6 拦截响应地址
	DefaultBlockingIPv6 = "::"

	// DefaultRuleTTL 拦截响应的默认缓存 TTL 秒数
	DefaultRuleTTL uint32 = 10
)

// FilterList 订阅规则源描述
type FilterList struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"` // 支持 http://, https:// 或本地文件绝对/相对路径
	Enabled     bool   `json:"enabled"`
	RulesCount  int    `json:"rulesCount"`
	LastUpdated int64  `json:"lastUpdated"` // Unix 毫秒时间戳
	Checksum    string `json:"checksum"`
}

// FilterConfig 规则过滤引擎运行与策略配置
type FilterConfig struct {
	Enabled             bool         `json:"enabled"`
	BlockMode           string       `json:"blockMode"`    // "null_ip" | "nxdomain" | "refused"
	BlockingIPv4        string       `json:"blockingIPv4"` // 默认 "0.0.0.0"
	BlockingIPv6        string       `json:"blockingIPv6"` // 默认 "::"
	CustomRules         []string     `json:"customRules"`  // 用户自定义规则文本行
	Lists               []FilterList `json:"lists"`        // 订阅规则列表
	UpdateIntervalHours int          `json:"updateIntervalHours"`
	DataDir             string       `json:"dataDir"` // 规则源 .txt 缓存目录

	// 分时拦截（时间计划）：开启后仅在命中计划的窗口内执行拦截
	ScheduleEnabled bool               `json:"scheduleEnabled"`
	Schedules       []BlockingSchedule `json:"schedules"`

	// FilterResponse 响应侧别名链复检开关（默认关闭）。
	// 开启后对上游应答中的 CNAME 目标与 A/AAAA 记录所属名二次匹配黑名单规则，
	// 用于拦截「QNAME 合法但别名指向黑名单域」的深链绕过。
	FilterResponse bool `json:"filterResponse"`
}

// DefaultFilterConfig 生成默认过滤配置
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		Enabled:             true,
		BlockMode:           BlockModeNullIP,
		BlockingIPv4:        DefaultBlockingIPv4,
		BlockingIPv6:        DefaultBlockingIPv6,
		CustomRules:         []string{},
		Lists:               defaultSubscriptionLists(),
		UpdateIntervalHours: 24,
		DataDir:             platform.DefaultFilterDir(),
		ScheduleEnabled:     false,
		Schedules:           []BlockingSchedule{},
		FilterResponse:      false,
	}
}

// defaultSubscriptionLists 提供默认内置订阅列表
func defaultSubscriptionLists() []FilterList {
	return []FilterList{
		{
			ID:          "adguard-dns-filter",
			Name:        "AdGuard DNS Filter",
			URL:         "https://adguardteam.github.io/HostlistsRegistry/assets/filter_1.txt",
			Enabled:     true,
			RulesCount:  0,
			LastUpdated: 0,
		},
	}
}

// NormalizeFilterConfig 校验与修补配置项默认值
func NormalizeFilterConfig(cfg *FilterConfig) {
	if cfg == nil {
		return
	}
	if cfg.BlockMode == "" {
		cfg.BlockMode = BlockModeNullIP
	}
	if cfg.BlockingIPv4 == "" {
		cfg.BlockingIPv4 = DefaultBlockingIPv4
	}
	if cfg.BlockingIPv6 == "" {
		cfg.BlockingIPv6 = DefaultBlockingIPv6
	}
	if cfg.UpdateIntervalHours <= 0 {
		cfg.UpdateIntervalHours = 24
	}
	if cfg.DataDir == "" {
		cfg.DataDir = DefaultFilterConfig().DataDir
	}
	if cfg.CustomRules == nil {
		cfg.CustomRules = []string{}
	}
	if cfg.Lists == nil {
		cfg.Lists = defaultSubscriptionLists()
	}
	// 编译时间计划，丢弃非法条目并记录日志（坏配置不阻断拦截常开语义）
	if cfg.Schedules == nil {
		cfg.Schedules = []BlockingSchedule{}
	} else {
		valid := cfg.Schedules[:0]
		for i := range cfg.Schedules {
			if err := NormalizeSchedule(&cfg.Schedules[i]); err != nil {
				log.Printf("[FilterConfig] 丢弃非法时间计划 (%s): %v", cfg.Schedules[i].Name, err)
				continue
			}
			valid = append(valid, cfg.Schedules[i])
		}
		cfg.Schedules = valid
	}
}

// FilterStats 规则引擎运行时统计指标
type FilterStats struct {
	Enabled        bool    `json:"enabled"`
	TotalRules     int     `json:"totalRules"`
	ActiveLists    int     `json:"activeLists"`
	TotalQueries   uint64  `json:"totalQueries"`
	BlockedQueries uint64  `json:"blockedQueries"`
	AllowedQueries uint64  `json:"allowedQueries"`
	BlockRate      float64 `json:"blockRate"`
	// ScheduleActive 当前时刻拦截是否处于生效窗口（计划关闭时恒为 true）
	ScheduleActive bool `json:"scheduleActive"`
}

// CheckHostResult 域名过滤检测结果
type CheckHostResult struct {
	Blocked     bool   `json:"blocked"`
	Action      string `json:"action"` // "block" | "allow" | "pass"
	MatchedRule string `json:"matchedRule"`
	ListName    string `json:"listName"`
	Reason      string `json:"reason"`
}

// CheckDomainRequest 域名检测请求 DTO
type CheckDomainRequest struct {
	Domain string `json:"domain"`
	QType  string `json:"qtype,omitempty"`
}

// UpdateRulesRequest 更新用户自定义规则请求 DTO
type UpdateRulesRequest struct {
	Rules []string `json:"rules"`
}
