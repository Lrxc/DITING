package core

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// RuleEngine 谛听 Windows 规则过滤与拦截引擎核心
type RuleEngine struct {
	mu             sync.RWMutex
	cfg            FilterConfig
	matcher        *RuleMatcher
	totalQueries   atomic.Uint64
	blockedQueries atomic.Uint64
	allowedQueries atomic.Uint64

	httpClient *http.Client
	stopCh     chan struct{}

	// now 时钟注入点，默认 time.Now；测试可借此固定"当前时刻"
	now func() time.Time
}

// NewRuleEngine 初始化规则引擎并自动加载构建规则索引
func NewRuleEngine(cfg FilterConfig) *RuleEngine {
	NormalizeFilterConfig(&cfg)

	engine := &RuleEngine{
		cfg:     cfg,
		matcher: NewRuleMatcher(),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		stopCh: make(chan struct{}),
		now:    time.Now,
	}

	// 确保规则缓存目录存在
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Printf("[RuleEngine] 创建缓存目录失败 (%s): %v", cfg.DataDir, err)
	}

	// 首次全量加载规则
	if err := engine.rebuildUnlocked(); err != nil {
		log.Printf("[RuleEngine] 初始加载规则出现警告: %v", err)
	}

	// 启动定期更新定时器
	go engine.runUpdateTicker()

	return engine
}

// IsEnabled 查询规则过滤总开关
func (e *RuleEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cfg.Enabled
}

// SetEnabled 设置规则过滤总开关
func (e *RuleEngine) SetEnabled(enabled bool) {
	e.mu.Lock()
	e.cfg.Enabled = enabled
	e.mu.Unlock()
}

// GetConfig 获取当前过滤配置副本
func (e *RuleEngine) GetConfig() FilterConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cfg
}

// UpdateConfig 更新过滤配置并热重载
func (e *RuleEngine) UpdateConfig(cfg FilterConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	old := e.cfg

	// 保持未显式更新字段
	if cfg.CustomRules == nil {
		cfg.CustomRules = old.CustomRules
	}
	if cfg.Lists == nil {
		cfg.Lists = old.Lists
	}
	if cfg.DataDir == "" {
		cfg.DataDir = old.DataDir
	}
	// 深拷贝时间计划，避免编译产物写回引擎在用切片的后备数组
	if cfg.Schedules == nil {
		cfg.Schedules = append([]BlockingSchedule(nil), old.Schedules...)
	} else {
		cfg.Schedules = append([]BlockingSchedule(nil), cfg.Schedules...)
	}
	NormalizeFilterConfig(&cfg)

	e.cfg = cfg

	// 规则索引仅由自定义规则与订阅列表决定；其余字段（开关、时间计划等）热更新即可
	matcherInputsUnchanged := e.matcher != nil && e.matcher.TotalRules() > 0 &&
		len(old.CustomRules) == len(cfg.CustomRules) &&
		len(old.Lists) == len(cfg.Lists)
	if matcherInputsUnchanged {
		return nil
	}

	return e.rebuildUnlocked()
}

func (e *RuleEngine) getMatcher() *RuleMatcher {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.matcher
}

// GetStats 获取当前运行时拦截与规则统计指标
func (e *RuleEngine) GetStats() FilterStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	total := e.totalQueries.Load()
	blocked := e.blockedQueries.Load()
	allowed := e.allowedQueries.Load()
	var rate float64
	if total > 0 {
		rate = float64(blocked) / float64(total) * 100
	}

	activeLists := 0
	for _, l := range e.cfg.Lists {
		if l.Enabled {
			activeLists++
		}
	}

	totalRules := 0
	if e.matcher != nil {
		totalRules = e.matcher.TotalRules()
	}

	// 计划关闭或未配置启用计划时恒为生效（避免重入加锁，直接内联判定）
	scheduleActive := true
	if e.cfg.ScheduleEnabled && HasEnabledSchedule(e.cfg.Schedules) {
		scheduleActive = ActiveSchedule(e.cfg.Schedules, e.currentTime())
	}

	return FilterStats{
		Enabled:        e.cfg.Enabled,
		TotalRules:     totalRules,
		ActiveLists:    activeLists,
		TotalQueries:   total,
		BlockedQueries: blocked,
		AllowedQueries: allowed,
		BlockRate:      rate,
		ScheduleActive: scheduleActive,
	}
}

// Match 对域名进行过滤评估并记录遥测统计
func (e *RuleEngine) Match(domain string, qtype uint16) CheckHostResult {
	if !e.IsEnabled() {
		return CheckHostResult{Action: "pass"}
	}

	// 分时拦截：计划开启且未处于生效窗口时，整体放行且不记入拦截统计
	if !e.IsBlockingActive() {
		return CheckHostResult{Action: "pass", Reason: ReasonScheduleInactive}
	}

	matcher := e.getMatcher()
	if matcher == nil {
		return CheckHostResult{Action: "pass"}
	}

	e.totalQueries.Add(1)
	res := matcher.Match(domain, qtype)

	if res.Blocked {
		e.blockedQueries.Add(1)
	} else if res.Action == "allow" {
		e.allowedQueries.Add(1)
	}

	return res
}

// IsBlockingActive 判断当前时刻拦截功能是否处于生效状态。
//
// 语义：时间计划开关关闭，或未配置任何启用计划时，拦截恒为常开（等价历史行为）；
// 开关开启且存在启用计划时，仅命中窗口才拦截。
func (e *RuleEngine) IsBlockingActive() bool {
	return e.IsBlockingActiveAt(e.currentTime())
}

// IsBlockingActiveAt 判断给定时刻拦截功能是否处于生效状态（便于测试注入时钟）。
func (e *RuleEngine) IsBlockingActiveAt(now time.Time) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.cfg.ScheduleEnabled || !HasEnabledSchedule(e.cfg.Schedules) {
		return true
	}
	return ActiveSchedule(e.cfg.Schedules, now)
}

// CurrentSchedule 返回当前生效的时间计划（无则 nil）。
func (e *RuleEngine) CurrentSchedule() *BlockingSchedule {
	return e.CurrentScheduleAt(e.currentTime())
}

// CurrentScheduleAt 返回给定时刻生效的时间计划（无则 nil）。
func (e *RuleEngine) CurrentScheduleAt(now time.Time) *BlockingSchedule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.cfg.ScheduleEnabled {
		return nil
	}
	return FirstActiveSchedule(e.cfg.Schedules, now)
}

// SetClock 注入自定义时钟（测试专用），传 nil 恢复 time.Now。
func (e *RuleEngine) SetClock(now func() time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if now == nil {
		e.now = time.Now
		return
	}
	e.now = now
}

// currentTime 读取注入时钟的当前时刻（锁外调用，仅读函数指针）。
func (e *RuleEngine) currentTime() time.Time {
	e.mu.RLock()
	now := e.now
	e.mu.RUnlock()
	if now == nil {
		now = time.Now
	}
	return now()
}

// CheckHost 提供外部纯只读检测（不增加正式查询指标计数）
func (e *RuleEngine) CheckHost(domain string, qtype uint16) CheckHostResult {
	matcher := e.getMatcher()
	if matcher == nil {
		return CheckHostResult{Action: "pass"}
	}
	return matcher.Match(domain, qtype)
}

// rebuildUnlocked 从自定义规则与订阅缓存重建匹配器并原子切换
func (e *RuleEngine) rebuildUnlocked() error {
	var allRules []*ParsedRule

	// 1. 解析用户自定义规则 (最高优先级)
	for _, line := range e.cfg.CustomRules {
		if r, ok := ParseRuleLine(line, "custom"); ok && r != nil {
			allRules = append(allRules, r)
		}
	}

	// 2. 加载各启用订阅列表的缓存文件
	for i := range e.cfg.Lists {
		list := &e.cfg.Lists[i]
		if !list.Enabled {
			continue
		}
		cacheFile := filepath.Join(e.cfg.DataDir, list.ID+".txt")
		if data, err := os.ReadFile(cacheFile); err == nil {
			listRules, err := ParseRules(bytes.NewReader(data), list.ID)
			if err == nil {
				list.RulesCount = len(listRules)
				allRules = append(allRules, listRules...)
			}
		}
	}

	// 3. 构建新的匹配器实例并全量原子切换
	newMatcher := NewRuleMatcher()
	newMatcher.BuildFromRules(allRules)
	e.matcher = newMatcher

	log.Printf("[RuleEngine] 规则索引构建完毕，共激活 %d 条规则", len(allRules))
	return nil
}

func (e *RuleEngine) runUpdateTicker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.mu.Lock()
			interval := time.Duration(e.cfg.UpdateIntervalHours) * time.Hour
			needUpdate := false
			for _, l := range e.cfg.Lists {
				if l.Enabled && time.Since(time.UnixMilli(l.LastUpdated)) > interval {
					needUpdate = true
					break
				}
			}
			if needUpdate {
				for _, l := range e.cfg.Lists {
					if l.Enabled && time.Since(time.UnixMilli(l.LastUpdated)) > interval {
						_ = e.refreshListUnlocked(l.ID)
					}
				}
				_ = e.rebuildUnlocked()
			}
			e.mu.Unlock()
		}
	}
}

// Close 关闭规则引擎后台协程
func (e *RuleEngine) Close() {
	close(e.stopCh)
}
