//go:build linux

package linux

import (
	"github.com/haoze-evolluling/diting/desktop/internal/platform"
)

// LinuxAspectRatioManager Linux 平台窗口宽高比管理存根。
type LinuxAspectRatioManager struct{}

// NewAspectRatioManager 构造 Linux 窗口宽高比管理器。
func NewAspectRatioManager() platform.AspectRatioManager {
	return &LinuxAspectRatioManager{}
}

// LockAspectRatio 在 Linux 环境下为空操作。
func (m *LinuxAspectRatioManager) LockAspectRatio(title string, ratio float64) {
	// Linux 无头/Web 服务模式下无需锁定原生窗口宽高比
}
