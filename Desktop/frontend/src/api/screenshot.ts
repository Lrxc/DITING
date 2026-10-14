/**
 * 软件界面截图接口
 * 依赖 Wails 原生窗口截屏能力（当前仅 Windows 桌面端实现），非桌面环境不支持。
 */

import { CaptureWindow } from '../../wailsjs/go/main/App';

export async function captureAppScreenshot(outputPath = ''): Promise<string> {
  if (typeof window !== 'undefined' && (window as any).go?.main?.App?.CaptureWindow) {
    return await CaptureWindow(outputPath);
  }
  throw new Error('当前环境不支持原生窗口截图，请在桌面客户端中使用');
}

// 挂载至全局 window 对象，便于测试脚本、控制台调用或自动化测试
if (typeof window !== 'undefined') {
  (window as any).captureScreenshot = captureAppScreenshot;
}
