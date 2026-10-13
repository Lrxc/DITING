package main

import (
	"context"
	"fmt"
	"log"

	"github.com/haoze-evolluling/diting/desktop/internal/config"
	"github.com/haoze-evolluling/diting/desktop/internal/core"
	"github.com/haoze-evolluling/diting/desktop/internal/ipc"
)

// GetClientGroups 返回当前客户端分组配置与可选订阅列表
func (p *program) GetClientGroups(ctx context.Context) (*ipc.ClientGroupConfigResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg := p.cfg.Client.GroupConfig()
	res := &ipc.ClientGroupConfigResponse{
		Enabled:   cfg.Enabled,
		Groups:    cfg.Groups,
		RuleLists: p.clientGroupRuleListOptions(),
	}
	if res.Groups == nil {
		res.Groups = []core.ClientGroup{}
	}
	return res, nil
}

// clientGroupRuleListOptions 汇总可供分组绑定的订阅列表（调用方需持有 p.mu）
func (p *program) clientGroupRuleListOptions() []ipc.ClientGroupRuleListOption {
	opts := []ipc.ClientGroupRuleListOption{}
	if p.filterEngine == nil {
		return opts
	}
	for _, l := range p.filterEngine.GetFilterLists() {
		opts = append(opts, ipc.ClientGroupRuleListOption{ID: l.ID, Name: l.Name, Enabled: l.Enabled})
	}
	return opts
}

// ConfigureClientGroups 更新客户端分组配置并热生效
func (p *program) ConfigureClientGroups(ctx context.Context, req ipc.ConfigureClientGroupsRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.filterEngine == nil {
		return fmt.Errorf("规则过滤引擎未初始化")
	}

	cfg := p.cfg.Client.GroupConfig()
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.Groups != nil {
		cfg.Groups = *req.Groups
	}
	if err := core.NormalizeClientGroupConfig(&cfg); err != nil {
		log.Printf("[Clients] 客户端分组配置归一化提示: %v\n", err)
	}

	// 1. 热更新规则引擎的分组匹配器
	if err := p.filterEngine.SetClientGroups(cfg); err != nil {
		return err
	}

	// 2. 重建 IP→分组解析器（DNS 热路径使用）
	if gr, err := core.NewGroupResolver(cfg); err == nil {
		p.groupResolver = gr
	} else {
		log.Printf("[Clients] 重建客户端分组匹配器警告: %v\n", err)
	}

	// 3. 同步持久化配置
	p.cfg.Client.GroupsEnabled = cfg.Enabled
	p.cfg.Client.Groups = cfg.Groups
	if p.configPath != "" {
		if err := config.SaveConfig(p.configPath, p.cfg); err != nil {
			log.Printf("[配置] 持久化客户端分组配置失败: %v\n", err)
		}
	}
	return nil
}
