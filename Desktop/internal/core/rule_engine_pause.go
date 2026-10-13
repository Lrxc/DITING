package core

import "time"

// 定时暂停拦截（借鉴 Technitium TemporaryDisableBlocking / pi-hole piholeEnable 定时）。
//
// 设计取舍：暂停为**纯运行时状态**，不写入 config.json。
// 若持久化，服务重启后无人值守的暂停会长期关闭过滤，属安全上不可接受的状态；
// 因此进程重启即自动恢复拦截，由前端按需重新发起暂停。

// PauseFor 暂停拦截指定时长（d <= 0 表示暂停至手动恢复）。
func (e *RuleEngine) PauseFor(d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	nowFn := e.now
	if nowFn == nil {
		nowFn = time.Now
	}
	if d <= 0 {
		// 以远未来时刻表示「暂停至手动恢复」
		e.pausedUntil = nowFn().Add(100 * 365 * 24 * time.Hour)
		return
	}
	e.pausedUntil = nowFn().Add(d)
}

// ResumePause 立即恢复拦截。
func (e *RuleEngine) ResumePause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pausedUntil = time.Time{}
}

// PauseState 返回当前暂停状态：是否暂停、剩余时长、预计恢复时刻。
//
// 剩余时长为 0 表示未暂停或暂停已自然到期。
func (e *RuleEngine) PauseState() (paused bool, remaining time.Duration, until time.Time) {
	e.mu.RLock()
	until = e.pausedUntil
	nowFn := e.now
	e.mu.RUnlock()

	if until.IsZero() {
		return false, 0, time.Time{}
	}
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()
	if !now.Before(until) {
		return false, 0, until
	}
	return true, until.Sub(now), until
}

// pauseStateUnlocked 读取暂停状态（调用方须持有 e.mu，读锁或写锁均可）。
func (e *RuleEngine) pauseStateUnlocked() (paused bool, until time.Time) {
	if e.pausedUntil.IsZero() {
		return false, time.Time{}
	}
	nowFn := e.now
	if nowFn == nil {
		nowFn = time.Now
	}
	if !nowFn().Before(e.pausedUntil) {
		return false, e.pausedUntil
	}
	return true, e.pausedUntil
}
