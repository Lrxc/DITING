//go:build linux

package linux

import (
	"errors"

	"github.com/haoze-evolluling/diting/desktop/internal/platform"
)

// LinuxWindowCapturer Linux 平台窗口截屏存根。
type LinuxWindowCapturer struct{}

// NewWindowCapturer 构造 Linux 窗口截屏器。
func NewWindowCapturer() platform.WindowCapturer {
	return &LinuxWindowCapturer{}
}

// CaptureWindow 在无原生窗口环境下返回明确提示。
func (c *LinuxWindowCapturer) CaptureWindow(outputPath string) (string, error) {
	return "", errors.New("Linux 环境目前以服务与 Web 控制台形式交付，暂不支持窗口截图")
}
