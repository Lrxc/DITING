package main

import (
	"context"
	"fmt"

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
