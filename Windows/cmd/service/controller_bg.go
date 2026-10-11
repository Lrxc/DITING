package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/haoze-evolluling/diting/windows/internal/config"
	"github.com/haoze-evolluling/diting/windows/internal/ipc"
)

func (p *program) getBackgroundStorageDir() string {
	cfgPath := p.configPath
	if cfgPath == "" {
		cfgPath = config.GetDefaultConfigPath()
	}
	return filepath.Dir(cfgPath)
}

func (p *program) findExistingBgFile() (string, error) {
	dir := p.getBackgroundStorageDir()
	matches, err := filepath.Glob(filepath.Join(dir, "login_bg.*"))
	if err != nil || len(matches) == 0 {
		return "", os.ErrNotExist
	}
	return matches[0], nil
}

func (p *program) cleanExistingBgFiles() {
	dir := p.getBackgroundStorageDir()
	matches, _ := filepath.Glob(filepath.Join(dir, "login_bg.*"))
	for _, f := range matches {
		_ = os.Remove(f)
	}
}

// GetLoginBackground 实现 ServiceController 接口
func (p *program) GetLoginBackground(ctx context.Context) (*ipc.LoginBackgroundResponse, error) {
	p.mu.Lock()
	bg := p.cfg.Web.Background
	p.mu.Unlock()

	// 容灾检查：若配置为 custom 本地图片，检查磁盘文件是否存在
	if bg.Type == "custom" {
		filePath, err := p.findExistingBgFile()
		if err != nil || filePath == "" {
			return &ipc.LoginBackgroundResponse{
				Type:      "none",
				URL:       "",
				Opacity:   40,
				Blur:      0,
				UpdatedAt: bg.UpdatedAt,
			}, nil
		}
		bgURL := fmt.Sprintf("/api/v1/auth/background/image?t=%d", bg.UpdatedAt)
		return &ipc.LoginBackgroundResponse{
			Type:      "custom",
			URL:       bgURL,
			Opacity:   bg.Opacity,
			Blur:      bg.Blur,
			UpdatedAt: bg.UpdatedAt,
		}, nil
	}

	if bg.Type == "url" && bg.URL != "" {
		return &ipc.LoginBackgroundResponse{
			Type:      "url",
			URL:       bg.URL,
			Opacity:   bg.Opacity,
			Blur:      bg.Blur,
			UpdatedAt: bg.UpdatedAt,
		}, nil
	}

	return &ipc.LoginBackgroundResponse{
		Type:      "none",
		URL:       "",
		Opacity:   40,
		Blur:      0,
		UpdatedAt: bg.UpdatedAt,
	}, nil
}

// SetLoginBackground 实现 ServiceController 接口
func (p *program) SetLoginBackground(ctx context.Context, req ipc.SetLoginBackgroundRequest) error {
	opacity := req.Opacity
	if opacity < 0 {
		opacity = 0
	} else if opacity > 100 {
		opacity = 100
	}

	blur := req.Blur
	if blur < 0 {
		blur = 0
	} else if blur > 20 {
		blur = 20
	}

	trimmedURL := strings.TrimSpace(req.URL)

	// 1. 本地图片 Base64 上传处理
	if strings.HasPrefix(trimmedURL, "data:image/") {
		commaIdx := strings.Index(trimmedURL, ",")
		if commaIdx == -1 {
			return fmt.Errorf("无效的图片 Data URL 数据格式")
		}
		meta := trimmedURL[:commaIdx]
		rawB64 := trimmedURL[commaIdx+1:]

		ext := ".webp"
		switch {
		case strings.Contains(meta, "image/webp"):
			ext = ".webp"
		case strings.Contains(meta, "image/png"):
			ext = ".png"
		case strings.Contains(meta, "image/jpeg"), strings.Contains(meta, "image/jpg"):
			ext = ".jpg"
		case strings.Contains(meta, "image/svg+xml"), strings.Contains(meta, "image/svg"):
			ext = ".svg"
		}

		imgBytes, err := base64.StdEncoding.DecodeString(rawB64)
		if err != nil {
			return fmt.Errorf("图片 Base64 解码失败: %w", err)
		}
		if len(imgBytes) == 0 {
			return fmt.Errorf("图片数据内容为空")
		}
		if len(imgBytes) > 10*1024*1024 {
			return fmt.Errorf("图片数据体积过大 (超过 10MB 限额)")
		}

		// 格式有效性验证
		if !isValidImageBytes(imgBytes, ext) {
			return fmt.Errorf("图片文件头校验失败，格式可能已损坏或不支持")
		}

		dir := p.getBackgroundStorageDir()
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建背景存储目录失败: %w", err)
		}

		targetPath := filepath.Join(dir, "login_bg"+ext)
		tmpPath := filepath.Join(dir, "bg_upload.tmp")
		if err := os.WriteFile(tmpPath, imgBytes, 0644); err != nil {
			return fmt.Errorf("写入背景临时文件失败: %w", err)
		}

		p.cleanExistingBgFiles()
		_ = os.Remove(targetPath)
		if err := os.Rename(tmpPath, targetPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("持久化背景图片文件失败: %w", err)
		}

		p.mu.Lock()
		p.cfg.Web.Background = config.LoginBgConfig{
			Type:      "custom",
			URL:       "/api/v1/auth/background/image",
			Opacity:   opacity,
			Blur:      blur,
			UpdatedAt: time.Now().Unix(),
		}
		_ = config.SaveConfig(p.configPath, p.cfg)
		p.mu.Unlock()
		return nil
	}

	// 2. 外部网络 URL 处理
	if strings.HasPrefix(trimmedURL, "http://") || strings.HasPrefix(trimmedURL, "https://") {
		if len(trimmedURL) > 2048 {
			return fmt.Errorf("图片网络链接长度超过 2048 字符上限")
		}

		p.cleanExistingBgFiles()
		p.mu.Lock()
		p.cfg.Web.Background = config.LoginBgConfig{
			Type:      "url",
			URL:       trimmedURL,
			Opacity:   opacity,
			Blur:      blur,
			UpdatedAt: time.Now().Unix(),
		}
		_ = config.SaveConfig(p.configPath, p.cfg)
		p.mu.Unlock()
		return nil
	}

	// 3. 空 URL，等同于重置为默认背景
	return p.ResetLoginBackground(ctx)
}

// ResetLoginBackground 实现 ServiceController 接口
func (p *program) ResetLoginBackground(ctx context.Context) error {
	p.cleanExistingBgFiles()
	p.mu.Lock()
	p.cfg.Web.Background = config.LoginBgConfig{
		Type:      "none",
		URL:       "",
		Opacity:   40,
		Blur:      0,
		UpdatedAt: time.Now().Unix(),
	}
	_ = config.SaveConfig(p.configPath, p.cfg)
	p.mu.Unlock()
	return nil
}

// GetLoginBackgroundImage 实现 ServiceController 接口
func (p *program) GetLoginBackgroundImage(ctx context.Context) ([]byte, string, time.Time, error) {
	filePath, err := p.findExistingBgFile()
	if err != nil {
		return nil, "", time.Time{}, os.ErrNotExist
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "image/webp"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".svg":
		contentType = "image/svg+xml"
	case ".webp":
		contentType = "image/webp"
	}

	return data, contentType, info.ModTime(), nil
}

// VerifyAdminPassword 实现 ServiceController 接口
func (p *program) VerifyAdminPassword(ctx context.Context, plainPassword string) bool {
	if p.authMgr == nil {
		return false
	}
	return p.authMgr.VerifyPassword(plainPassword)
}

// IsAuthInitialized 实现 ServiceController 接口
func (p *program) IsAuthInitialized() bool {
	if p.authMgr == nil {
		return false
	}
	return p.authMgr.IsInitialized()
}

// isValidImageBytes 校验图片前导魔数与文件头合法性
func isValidImageBytes(data []byte, ext string) bool {
	if len(data) < 8 {
		return false
	}
	switch ext {
	case ".webp":
		return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	case ".png":
		return bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n"))
	case ".jpg", ".jpeg":
		return len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
	case ".svg":
		header := strings.ToLower(string(data[:min(len(data), 512)]))
		return strings.Contains(header, "<svg") || strings.Contains(header, "<?xml")
	}
	return true
}
