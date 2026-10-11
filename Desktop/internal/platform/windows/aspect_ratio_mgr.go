//go:build windows

package windows

import (
	"github.com/haoze-evolluling/diting/desktop/internal/platform"
)

// WindowsAspectRatioManager 管理 Windows 窗口宽高比锁定。
type WindowsAspectRatioManager struct{}

// NewAspectRatioManager 构造 Windows 窗口宽高比管理器。
func NewAspectRatioManager() platform.AspectRatioManager {
	return &WindowsAspectRatioManager{}
}

// LockAspectRatio 锁定指定标题窗口的缩放宽高比。
func (m *WindowsAspectRatioManager) LockAspectRatio(title string, ratio float64) {
	LockWindowAspectRatio(title, ratio)
}
