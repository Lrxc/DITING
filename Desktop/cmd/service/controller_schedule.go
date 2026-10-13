package main

import (
	"context"
	"fmt"
	"time"

	"github.com/haoze-evolluling/diting/desktop/internal/config"
	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/haoze-evolluling/diting/desktop/internal/ipc"
)

// GetBlockingSchedule 实现 ServiceController 接口
func (p *program) GetBlockingSchedule(ctx context.Context) (*ipc.ScheduleConfigResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.filterEngine == nil {
		return &ipc.ScheduleConfigResponse{
			Enabled:   false,
			Schedules: []core.BlockingSchedule{},
			Active:    true,
		}, nil
	}

	cfg := p.filterEngine.GetConfig()
	res := &ipc.ScheduleConfigResponse{
		Enabled:   cfg.ScheduleEnabled,
		Schedules: cfg.Schedules,
		Active:    p.filterEngine.IsBlockingActive(),
	}
	if s := p.filterEngine.CurrentSchedule(); s != nil {
		res.ActiveID = s.ID
	}
	return res, nil
}

// ConfigureBlockingSchedule 实现 ServiceController 接口
func (p *program) ConfigureBlockingSchedule(ctx context.Context, req ipc.ConfigureScheduleRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.filterEngine == nil {
		return fmt.Errorf("规则过滤引擎未初始化")
	}

	cfg := p.filterEngine.GetConfig()
	if req.Enabled != nil {
		cfg.ScheduleEnabled = *req.Enabled
	}
	if req.Schedules != nil {
		cfg.Schedules = *req.Schedules
	}

	if err := p.filterEngine.UpdateConfig(cfg); err != nil {
		return err
	}
	p.cfg.Filter = p.filterEngine.GetConfig()
	_ = config.SaveConfig(p.configPath, p.cfg)
	return nil
}

// GetFilterPause 实现 ServiceController 接口
func (p *program) GetFilterPause(ctx context.Context) (*ipc.FilterPauseResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.filterEngine == nil {
		return &ipc.FilterPauseResponse{Paused: false, BlockingEnabled: false}, nil
	}

	paused, remaining, until := p.filterEngine.PauseState()
	res := &ipc.FilterPauseResponse{
		Paused:          paused,
		BlockingEnabled: p.filterEngine.GetConfig().Enabled,
	}
	if paused {
		res.RemainingSeconds = int64(remaining.Seconds())
		res.PausedUntilMs = until.UnixMilli()
	}
	return res, nil
}

// SetFilterPause 实现 ServiceController 接口。
//
// 暂停为纯运行时状态，不写入 config.json；仅广播事件以便各前端同步倒计时。
func (p *program) SetFilterPause(ctx context.Context, req ipc.FilterPauseRequest) error {
	p.mu.Lock()
	engine := p.filterEngine
	ipcServer := p.ipcServer
	p.mu.Unlock()

	if engine == nil {
		return fmt.Errorf("规则过滤引擎未初始化")
	}

	if req.Paused {
		engine.PauseFor(time.Duration(req.DurationSec) * time.Second)
	} else {
		engine.ResumePause()
	}

	if ipcServer != nil {
		state := "resumed"
		if req.Paused {
			state = "paused"
		}
		ipcServer.Broadcast(ipc.Event{Type: "filter", Timestamp: time.Now().UnixMilli(), Data: state})
	}
	return nil
}
