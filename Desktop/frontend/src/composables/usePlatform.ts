import { computed, readonly, ref, type Ref } from 'vue';
import { ipc } from '../api/ipc';
import { isDesktopApp } from '../utils/env';

/** 运行平台标识（unknown 表示尚未确认） */
export type PlatformOS = 'windows' | 'linux' | 'macos' | 'unknown';

const os = ref<PlatformOS>('unknown');
let initialized = false;

/** 依据浏览器 UA 给出平台初值（Web 模式的即时回退） */
function guessFromUserAgent(): PlatformOS {
  if (typeof navigator === 'undefined') return 'unknown';
  const ua = navigator.userAgent || '';
  if (/Windows/i.test(ua)) return 'windows';
  if (/Linux|X11/i.test(ua)) return 'linux';
  if (/Macintosh|Mac OS X/i.test(ua)) return 'macos';
  return 'unknown';
}

/**
 * 初始化运行平台：先以 UA 给出初值，再以权威来源覆盖。
 *
 * - 桌面端：调用 Wails 绑定 GetPlatform()。
 * - Web 控制台：请求免鉴权 /api/v1/meta 获取宿主机 GOOS。
 *
 * 任一步失败都保留 UA 初值，不会阻塞界面渲染。
 */
export async function initPlatform(): Promise<PlatformOS> {
  if (initialized) return os.value;
  initialized = true;

  os.value = guessFromUserAgent();

  try {
    if (isDesktopApp()) {
      const p = await window.go?.main?.App?.GetPlatform?.();
      if (p) os.value = normalize(p);
    } else {
      const meta = await ipc.getMeta();
      if (meta?.os) os.value = normalize(meta.os);
    }
  } catch {
    // 保留 UA 初值
  }

  return os.value;
}

/** 归一化平台字符串（兼容 Wails 返回的 darwin 等别名） */
function normalize(raw: string): PlatformOS {
  const v = (raw || '').toLowerCase();
  if (v === 'windows' || v === 'win32') return 'windows';
  if (v === 'linux') return 'linux';
  if (v === 'darwin' || v === 'macos' || v === 'mac') return 'macos';
  return 'unknown';
}

/**
 * 平台自适应组合式：供组件按 OS 切换文案与功能入口。
 *
 * 注意：os 为模块级响应式单例，initPlatform() 完成后所有已挂载组件自动刷新。
 */
export function usePlatform(): {
  os: Readonly<Ref<PlatformOS>>;
  isWindows: Readonly<Ref<boolean>>;
  isLinux: Readonly<Ref<boolean>>;
  platformLabel: Readonly<Ref<string>>;
  osReady: Readonly<Ref<boolean>>;
} {
  return {
    os: readonly(os) as Readonly<Ref<PlatformOS>>,
    isWindows: computed(() => os.value === 'windows'),
    isLinux: computed(() => os.value === 'linux'),
    platformLabel: computed(() => {
      switch (os.value) {
        case 'windows':
          return 'Windows';
        case 'linux':
          return 'Linux';
        case 'macos':
          return 'macOS';
        default:
          return '';
      }
    }),
    osReady: computed(() => os.value !== 'unknown'),
  };
}
