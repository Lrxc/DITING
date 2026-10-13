/**
 * 网络与表单通用校验工具函数
 */

/**
 * 校验 IP 地址是否有效（支持 IPv4 / IPv6，可包含端口如 :53 或 [::1]:53）
 */
export function validateIPAddress(input: string): boolean {
  const trimmed = input.trim();
  if (!trimmed) return false;

  let host = trimmed;
  let portStr = '';
  if (trimmed.startsWith('[')) {
    const endBracket = trimmed.indexOf(']');
    if (endBracket === -1) return false;
    host = trimmed.substring(1, endBracket);
    const rest = trimmed.substring(endBracket + 1);
    if (rest.startsWith(':')) {
      portStr = rest.substring(1);
    } else if (rest.length > 0) {
      return false;
    }
  } else if (trimmed.includes(':') && trimmed.indexOf(':') === trimmed.lastIndexOf(':')) {
    const parts = trimmed.split(':');
    host = parts[0] || '';
    portStr = parts[1] || '';
  }

  if (portStr) {
    if (!/^\d+$/.test(portStr)) return false;
    const p = parseInt(portStr, 10);
    if (p <= 0 || p > 65535) return false;
  }

  const ipv4Parts = host.split('.');
  if (ipv4Parts.length === 4) {
    return ipv4Parts.every((part) => {
      if (!/^\d+$/.test(part)) return false;
      const num = parseInt(part, 10);
      return num >= 0 && num <= 255 && (part === '0' || !part.startsWith('0'));
    });
  }

  if (host.includes(':')) {
    return /^[0-9a-fA-F:]+$/.test(host);
  }

  return false;
}

/**
 * 校验端口号 (1-65535)
 */
export function validatePort(port: number | string): boolean {
  const p = typeof port === 'string' ? parseInt(port.trim(), 10) : port;
  return !Number.isNaN(p) && p >= 1 && p <= 65535;
}

/**
 * 校验 DoH 协议 URL 地址
 */
export function validateDoHUrl(url: string): { valid: boolean; error?: string } {
  const trimmed = url.trim();
  if (!trimmed) {
    return { valid: false, error: 'DoH 协议必须填写加密解析地址 (URL，如 https://...)' };
  }
  if (!trimmed.startsWith('https://') && !trimmed.startsWith('http://')) {
    return { valid: false, error: 'DoH 加密解析地址必须以 https:// 开头' };
  }
  return { valid: true };
}
