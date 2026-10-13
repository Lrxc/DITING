/** 强制安全搜索与快捷管控类型定义（独立成文件以控制 types.ts 长度） */

/** 强制安全搜索的服务商标识（与后端 SafeSearchServices 保持一致） */
export type SafeSearchService =
  | 'bing'
  | 'duckduckgo'
  | 'ecosia'
  | 'google'
  | 'pixabay'
  | 'yandex'
  | 'youtube';

export interface SafeSearchServiceOption {
  id: SafeSearchService;
  label: string;
  /** 强制生效的目标入口，仅用于界面说明 */
  hint: string;
}

/** 内置服务商清单与说明（前端展示用，避免逐项硬编码） */
export const SAFE_SEARCH_SERVICES: SafeSearchServiceOption[] = [
  { id: 'bing', label: 'Bing', hint: 'strict.bing.com' },
  { id: 'duckduckgo', label: 'DuckDuckGo', hint: 'safe.duckduckgo.com' },
  { id: 'ecosia', label: 'Ecosia', hint: 'strict-safe-search.ecosia.org' },
  { id: 'google', label: 'Google', hint: 'forcesafesearch.google.com（全 TLD）' },
  { id: 'pixabay', label: 'Pixabay', hint: 'safesearch.pixabay.com' },
  { id: 'yandex', label: 'Yandex', hint: '213.180.193.56' },
  { id: 'youtube', label: 'YouTube', hint: 'restrictmoderate.youtube.com' },
];

/** 强制安全搜索配置（服务商采用黑名单式表达，空即全部启用） */
export interface SafeSearchConfig {
  enabled: boolean;
  disabled: SafeSearchService[];
  ttl: number;
}

export interface ConfigureSafeSearchRequest {
  enabled?: boolean;
  disabled?: SafeSearchService[];
  ttl?: number;
}

/** 定时暂停拦截状态 */
export interface FilterPauseResponse {
  paused: boolean;
  remainingSeconds: number;
  pausedUntilMs: number;
  blockingEnabled: boolean;
}

export interface FilterPauseRequest {
  paused: boolean;
  durationSec?: number;
}
