/**
 * 登录页自定义背景图片处理与持久化工具
 * 支持本地图片客户端 Canvas 智能压缩、网络图片预检及 LocalStorage 容灾
 */

export const BG_STORAGE_KEY_URL = 'diting_login_custom_bg';
export const BG_STORAGE_KEY_OPACITY = 'diting_login_bg_opacity';
export const BG_STORAGE_KEY_BLUR = 'diting_login_bg_blur';

export interface LoginBgConfig {
  url: string;
  opacity: number; // 0 - 100
  blur: number; // 0 - 20 px
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

        // 优先使用 WebP 编码以获得更小体积与高画质，降级使用 JPEG
        try {
          const dataUrl = canvas.toDataURL('image/webp', quality);
          if (dataUrl.startsWith('data:image/webp')) {
            resolve(dataUrl);
            return;
          }
        } catch {}

        resolve(canvas.toDataURL('image/jpeg', quality));
      };
      img.src = e.target?.result as string;
    };
    reader.readAsDataURL(file);
  });
}

/**
 * 校验网络图片地址是否可用
 */
export function validateImageUrl(url: string): Promise<boolean> {
  return new Promise((resolve) => {
    if (!url || !url.trim()) {
      resolve(false);
      return;
    }
    const img = new Image();
    img.onload = () => resolve(true);
    img.onerror = () => resolve(false);
    img.src = url.trim();
  });
}

/**
 * 读取本地保存的登录页背景配置
 * 默认情况下不设置背景图片 (url 为空)
 */
export function loadLoginBgConfig(): LoginBgConfig {
  try {
    const url = localStorage.getItem(BG_STORAGE_KEY_URL) || '';
    const rawOpacity = localStorage.getItem(BG_STORAGE_KEY_OPACITY);
    const rawBlur = localStorage.getItem(BG_STORAGE_KEY_BLUR);

    const opacity = rawOpacity !== null ? Number(rawOpacity) : 40;
    const blur = rawBlur !== null ? Number(rawBlur) : 0;

    return {
      url,
      opacity: Number.isNaN(opacity) ? 40 : Math.min(100, Math.max(0, opacity)),
      blur: Number.isNaN(blur) ? 0 : Math.min(20, Math.max(0, blur)),
    };
  } catch {
    return { url: '', opacity: 40, blur: 0 };
  }
}

/**
 * 持久化登录页背景配置至 LocalStorage
 */
export function saveLoginBgConfig(url: string, opacity = 40, blur = 0): { success: boolean; error?: string } {
  try {
    if (url) {
      localStorage.setItem(BG_STORAGE_KEY_URL, url);
    } else {
      localStorage.removeItem(BG_STORAGE_KEY_URL);
    }
    localStorage.setItem(BG_STORAGE_KEY_OPACITY, String(opacity));
    localStorage.setItem(BG_STORAGE_KEY_BLUR, String(blur));
    return { success: true };
  } catch (err: any) {
    return {
      success: false,
      error: '保存背景配置失败：可能超过了浏览器本地存储容量配额，请尝试使用网络图片地址。',
    };
  }
}

/**
 * 清除用户自定义背景，恢复系统默认无背景状态
 */
export function clearLoginBgConfig(): void {
  try {
    localStorage.removeItem(BG_STORAGE_KEY_URL);
    localStorage.removeItem(BG_STORAGE_KEY_OPACITY);
    localStorage.removeItem(BG_STORAGE_KEY_BLUR);
  } catch {}
}
