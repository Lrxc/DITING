//go:build windows

package windows

import (
	"fmt"
	"os"

	"github.com/haoze-evolluling/diting/desktop/internal/platform"
	"golang.org/x/sys/windows/registry"
)

const (
	runRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName   = "DitingDNS"
)

// WindowsAutoStartManager 负责通过 Windows 注册表管理当前用户的开机自启配置。
type WindowsAutoStartManager struct{}

// NewAutoStartManager 构造 Windows 自启管理器。
func NewAutoStartManager() platform.AutoStartManager {
	return &WindowsAutoStartManager{}
}

// IsEnabled 检查当前用户是否已启用开机自启。
func (m *WindowsAutoStartManager) IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValueName)
	return err == nil
}

// SetEnabled 开启或关闭当前用户的开机自启。
func (m *WindowsAutoStartManager) SetEnabled(enable bool) (bool, error) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runRegistryKey, registry.SET_VALUE)
	if err != nil {
		return false, fmt.Errorf("访问注册表失败: %w", err)
	}
	defer k.Close()

	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return false, fmt.Errorf("获取执行路径失败: %w", err)
		}
		if err := k.SetStringValue(runValueName, `"`+exePath+`"`); err != nil {
			return false, fmt.Errorf("写入自启注册表失败: %w", err)
		}
		return true, nil
	}

	_ = k.DeleteValue(runValueName)
	return false, nil
}
