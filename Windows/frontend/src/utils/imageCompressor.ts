/**
 * 登录页自定义背景图片处理与持久化工具
 * 支持本地图片客户端 Canvas 智能压缩、网络图片预检及 LocalStorage 容灾
 */

export const BG_STORAGE_KEY_TYPE = 'diting_login_bg_type';
export const BG_STORAGE_KEY_URL = 'diting_login_custom_bg';
export const BG_STORAGE_KEY_OPACITY = 'diting_login_bg_opacity';
export const BG_STORAGE_KEY_BLUR = 'diting_login_bg_blur';

export interface LoginBgConfig {
  type: 'none' | 'url' | 'custom';
  url: string;
  opacity: number; // 0 - 100
  blur: number; // 0 - 20 px
  updatedAt?: number;
}


/**
 * 客户端 Canvas 图像缩放与智能压缩：
 * 避免用户上传数兆原始照片导致 LocalStorage 超过 5MB 限额配额崩溃。
 */
export async function compressImageFile(
  file: File,
  maxWidth = 1920,
  maxHeight = 1080,
  quality = 0.82
): Promise<string> {
  // SVG 矢量图片体积通常很小，直接作为 DataURL 返回
  if (file.type === 'image/svg+xml') {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as string);
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  }

  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = reject;
    reader.onload = (e) => {
      const img = new Image();
      img.onerror = () => reject(new Error('无法解析图片文件，格式可能不被支持'));
      img.onload = () => {
        let width = img.width;
        let height = img.height;

        // 等比缩放计算
        if (width > maxWidth || height > maxHeight) {
          const ratio = Math.min(maxWidth / width, maxHeight / height);
          width = Math.round(width * ratio);
          height = Math.round(height * ratio);
        }

        const canvas = document.createElement('canvas');
        canvas.width = width;
        canvas.height = height;

        const ctx = canvas.getContext('2d');
        if (!ctx) {
          reject(new Error('Canvas 2D 上下文初始化失败'));
          return;
        }

        ctx.drawImage(img, 0, 0, width, height);

        // 优先使用 WebP 编码以获得更小体积与高画质，降级使用 JPEG/PNG
        let dataUrl = '';
        try {
          const webp = canvas.toDataURL('image/webp', quality);
          if (webp.startsWith('data:image/webp')) {
            dataUrl = webp;
          }
        } catch {}

        if (!dataUrl) {
          if (file.type === 'image/png') {
            dataUrl = canvas.toDataURL('image/png');
          } else {
            dataUrl = canvas.toDataURL('image/jpeg', quality);
          }
        }

        // 智能体积容灾：若 Base64 编码依然过大（超过 1.5MB），二次降采样以严格避免 LocalStorage 5MB 配额溢出
        if (dataUrl.length > 1.5 * 1024 * 1024) {
          try {
            const scaledCanvas = document.createElement('canvas');
            const targetRatio = Math.min(1280 / width, 720 / height, 0.75);
            scaledCanvas.width = Math.max(320, Math.round(width * targetRatio));
            scaledCanvas.height = Math.max(180, Math.round(height * targetRatio));
            const sCtx = scaledCanvas.getContext('2d');
            if (sCtx) {
              sCtx.drawImage(canvas, 0, 0, scaledCanvas.width, scaledCanvas.height);
              const secondWebp = scaledCanvas.toDataURL('image/webp', 0.72);
              if (secondWebp.startsWith('data:image/webp')) {
                dataUrl = secondWebp;
              } else {
                dataUrl = scaledCanvas.toDataURL('image/jpeg', 0.72);
              }
            }
          } catch {}
        }

        resolve(dataUrl);
      };
      img.src = e.target?.result as string;
    };
    reader.readAsDataURL(file);
  });
}

/**
 * 校验网络图片地址是否可用（含严格协议校验与超时保护）
 */
export function validateImageUrl(url: string, timeoutMs = 8000): Promise<boolean> {
  return new Promise((resolve) => {
    if (!url || !url.trim()) {
      resolve(false);
      return;
    }
    const cleanUrl = url.trim();
    // 仅允许 http/https 绝对网络地址或安全 data/blob 链接，拦截相对路径或危险协议
    if (!/^https?:\/\//i.test(cleanUrl) && !cleanUrl.startsWith('data:') && !cleanUrl.startsWith('blob:')) {
      resolve(false);
      return;
    }

    let isSettled = false;
    const img = new Image();
    const timer = setTimeout(() => {
      if (!isSettled) {
        isSettled = true;
        img.src = '';
        resolve(false);
      }
    }, timeoutMs);

    img.onload = () => {
      if (!isSettled) {
        isSettled = true;
        clearTimeout(timer);
        resolve(true);
      }
    };
    img.onerror = () => {
      if (!isSettled) {
        isSettled = true;
        clearTimeout(timer);
        resolve(false);
      }
    };
    img.src = cleanUrl;
  });
}

/**
 * 判断 URL 是否指向服务端内置的自定义背景图片接口
 */
export function isInternalBgUrl(url?: string): boolean {
  if (!url) return false;
  return url.includes('/api/v1/auth/background/image');
}

/**
 * 读取本地保存的登录页背景配置
 * 默认情况下不设置背景图片 (url 为空)
 */
export function loadLoginBgConfig(): LoginBgConfig {
  try {
    const rawType = localStorage.getItem(BG_STORAGE_KEY_TYPE) as 'none' | 'url' | 'custom' | null;
    const url = localStorage.getItem(BG_STORAGE_KEY_URL) || '';
    const rawOpacity = localStorage.getItem(BG_STORAGE_KEY_OPACITY);
    const rawBlur = localStorage.getItem(BG_STORAGE_KEY_BLUR);

    const opacity = rawOpacity !== null ? Number(rawOpacity) : 40;
    const blur = rawBlur !== null ? Number(rawBlur) : 0;

    let type: 'none' | 'url' | 'custom' = rawType || (url ? (isInternalBgUrl(url) ? 'custom' : 'url') : 'none');
    if (!url) {
      type = 'none';
    }

    return {
      type,
      url,
      opacity: Number.isNaN(opacity) ? 40 : Math.min(100, Math.max(0, opacity)),
      blur: Number.isNaN(blur) ? 0 : Math.min(20, Math.max(0, blur)),
    };
  } catch {
    return { type: 'none', url: '', opacity: 40, blur: 0 };
  }
}

/**
 * 持久化登录页背景配置至 LocalStorage（仅作轻量化秒开缓存，不存放 Base64 避免 QuotaExceededError）
 */
export function saveLoginBgConfig(
  type: 'none' | 'url' | 'custom',
  url: string,
  opacity = 40,
  blur = 0
): { success: boolean; error?: string } {
  try {
    const clampedOpacity = Math.min(100, Math.max(0, opacity));
    const clampedBlur = Math.min(20, Math.max(0, blur));

    // 关键优化：若为服务端存储的自定义背景，避免在 LocalStorage 中存放数兆 Base64，
    // 仅保存服务端路由标记或网络链接，彻底避免超过 LocalStorage 5MB 配额。
    let storageUrl = url;
    if (url.startsWith('data:image/')) {
      storageUrl = '/api/v1/auth/background/image';
    }

    if (type === 'none' || !storageUrl) {
      localStorage.removeItem(BG_STORAGE_KEY_URL);
      localStorage.removeItem(BG_STORAGE_KEY_TYPE);
    } else {
      localStorage.setItem(BG_STORAGE_KEY_TYPE, type);
      localStorage.setItem(BG_STORAGE_KEY_URL, storageUrl);
    }
    localStorage.setItem(BG_STORAGE_KEY_OPACITY, String(clampedOpacity));
    localStorage.setItem(BG_STORAGE_KEY_BLUR, String(clampedBlur));
    return { success: true };
  } catch (err: any) {
    return {
      success: false,
      error: '保存本地背景配置缓存失败',
    };
  }
}

/**
 * 清除用户自定义背景，恢复系统默认无背景状态
 */
export function clearLoginBgConfig(): void {
  try {
    localStorage.removeItem(BG_STORAGE_KEY_TYPE);
    localStorage.removeItem(BG_STORAGE_KEY_URL);
    localStorage.removeItem(BG_STORAGE_KEY_OPACITY);
    localStorage.removeItem(BG_STORAGE_KEY_BLUR);
  } catch {}
}

/**
 * 解析并规范化背景图片 URL：
 * 若为服务端相对路径（如 /api/v1/auth/background/image），结合 baseUrl 拼装为多端可访问的完整 URL。
 */
export function resolveBackgroundUrl(rawUrl: string, baseUrl?: string): string {
  if (!rawUrl || !rawUrl.trim()) return '';
  const trimmed = rawUrl.trim();
  if (
    trimmed.startsWith('http://') ||
    trimmed.startsWith('https://') ||
    trimmed.startsWith('data:') ||
    trimmed.startsWith('blob:')
  ) {
    return trimmed;
  }

  if (trimmed.startsWith('/')) {
    if (baseUrl) {
      return `${baseUrl.replace(/\/+$/, '')}${trimmed}`;
    }
    if (typeof window !== 'undefined' && window.location.origin) {
      return `${window.location.origin}${trimmed}`;
    }
  }

  return trimmed;
}

