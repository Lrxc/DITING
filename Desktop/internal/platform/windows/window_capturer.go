//go:build windows

package windows

import (
	"github.com/haoze-evolluling/diting/desktop/internal/platform"
)

// WindowsWindowCapturer 实现 Windows 窗口截屏能力。
type WindowsWindowCapturer struct{}

// NewWindowCapturer 构造 Windows 窗口截屏器。
func NewWindowCapturer() platform.WindowCapturer {
	return &WindowsWindowCapturer{}
}

// CaptureWindow 截取当前客户端自身窗口界面并保存为 PNG 图像文件。
func (c *WindowsWindowCapturer) CaptureWindow(outputPath string) (string, error) {
	return CaptureDitingWindow(outputPath)
}
