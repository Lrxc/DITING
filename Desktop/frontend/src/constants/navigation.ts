/**
 * 侧边导航与类别配置（App 与路由解析共用同一份定义）。
 *
 * 分类逻辑按「用户使用场景」而非实现模块划分：
 *   观测（总览） → 怎么解析（解析） → 拦什么（防护） → 管谁（设备） → 配本体（系统）
 */

export type NavTab = 'overview' | 'resolve' | 'protect' | 'devices' | 'system';

export interface NavItemDef {
  id: NavTab;
  label: string;
  icon: string;
  /** 顶栏语境说明 */
  hint: string;
}

export const NAV_ITEMS: readonly NavItemDef[] = [
  { id: 'overview', label: '总览', icon: 'dashboard', hint: '运行状态 · 实时查询 · 历史与统计' },
  { id: 'resolve', label: '解析', icon: 'upstream', hint: '上游调度 · 缓存加速 · 重写与 IPv6 过渡' },
  { id: 'protect', label: '防护', icon: 'shield', hint: '规则订阅 · 拦截策略与安全防护' },
  { id: 'devices', label: '设备', icon: 'devices', hint: '网卡接管 · 局域网服务 · 终端管控' },
  { id: 'system', label: '系统', icon: 'settings', hint: '外观 · 服务 · 远程控制台 · 诊断恢复' },
] as const;

export const VALID_TABS: readonly NavTab[] = NAV_ITEMS.map((item) => item.id);

export interface SubTabDef {
  id: string;
  label: string;
}

export const SUB_TABS: Record<NavTab, readonly SubTabDef[]> = {
  overview: [
    { id: 'status', label: '状态总览' },
    { id: 'queries', label: '实时查询' },
    { id: 'history', label: '查询历史' },
    { id: 'stats', label: '统计中心' },
  ],
  resolve: [
    { id: 'upstream', label: '上游解析' },
    { id: 'cache', label: '缓存加速' },
    { id: 'rewrite', label: 'DNS 重写' },
    { id: 'dns64', label: 'IPv6 / DNS64' },
  ],
  protect: [
    { id: 'lists', label: '规则订阅' },
    { id: 'custom', label: '自定义规则' },
    { id: 'test', label: '规则检测' },
    { id: 'config', label: '拦截方式' },
    { id: 'advanced', label: '高级拦截' },
    { id: 'schedule', label: '时间计划' },
  ],
  devices: [
    { id: 'adapters', label: '网卡接管' },
    { id: 'lan', label: '局域网 DNS' },
    { id: 'clients', label: '设备管理' },
    { id: 'privacy', label: '安全隐私' },
  ],
  system: [
    { id: 'appearance', label: '外观主题' },
    { id: 'service', label: '服务与连接' },
    { id: 'lanweb', label: '局域网控制台' },
    { id: 'diagnostics', label: '诊断与修复' },
  ],
};

/** 旧大类 id → 新大类 id（历史书签与旧链接重定向） */
export const LEGACY_TAB_MAP: Record<string, NavTab> = {
  dashboard: 'overview',
  logs: 'overview',
  network: 'devices',
  accel: 'resolve',
  rules: 'protect',
};

/** 旧大类在无小类时的默认小类（logs 原属实时查询语境） */
const LEGACY_TAB_DEFAULT_SUB: Record<string, string> = {
  logs: 'queries',
};

/** 旧小类 id → 所属新大类（用于 #/upstream、#/cache 这类仅含小类的旧链接） */
const LEGACY_SUB_OWNER: Record<string, NavTab> = {
  status: 'overview',
  queries: 'overview',
  history: 'overview',
  stats: 'overview',
  upstream: 'resolve',
  cache: 'resolve',
  rewrite: 'resolve',
  dns64: 'resolve',
  lists: 'protect',
  custom: 'protect',
  test: 'protect',
  config: 'protect',
  advanced: 'protect',
  schedule: 'protect',
  adapters: 'devices',
  lan: 'devices',
  clients: 'devices',
  privacy: 'devices',
  settings: 'system',
};

/** 旧「大类/小类」组合 → 新「大类/小类」（仅记录发生跨类迁移的组合） */
const LEGACY_ROUTE_MAP: Record<string, string> = {
  'logs/queries': 'overview/queries',
  'network/adapters': 'devices/adapters',
  'network/upstream': 'resolve/upstream',
  'network/lan': 'devices/lan',
  'network/clients': 'devices/clients',
  'network/dns64': 'resolve/dns64',
  'accel/cache': 'resolve/cache',
  'rules/rewrite': 'resolve/rewrite',
  'rules/privacy': 'devices/privacy',
  'system/settings': 'system/appearance',
};

export function isValidTab(value: string): value is NavTab {
  return (VALID_TABS as readonly string[]).includes(value);
}

/** 取某大类的默认（首个）小类 */
export function defaultSub(tab: NavTab): string {
  return SUB_TABS[tab][0]?.id ?? '';
}

function pickSub(tab: NavTab, sub: string): string {
  const subs = SUB_TABS[tab];
  return sub && subs.some((s) => s.id === sub) ? sub : defaultSub(tab);
}

/**
 * 归一化任意「大类 + 小类」到当前有效路由。
 * 处理顺序：旧组合路由 → 当前大类 id → 旧大类 id → 仅含小类的旧链接。
 * 无法识别时返回 null（由调用方保留默认分类）。
 */
export function resolveRoute(rawTab: string, rawSub = ''): { tab: NavTab; sub: string } | null {
  const t = (rawTab || '').toLowerCase().trim();
  const s = (rawSub || '').toLowerCase().trim();
  if (!t) return null;

  // 1) 旧「大类/小类」组合（可跨大类迁移归属）
  if (s) {
    const mapped = LEGACY_ROUTE_MAP[`${t}/${s}`];
    if (mapped) {
      const [mappedTab, mappedSub] = mapped.split('/');
      return { tab: mappedTab as NavTab, sub: mappedSub };
    }
  }

  // 2) 当前有效大类 id
  if (isValidTab(t)) {
    return { tab: t, sub: pickSub(t, s) };
  }

  // 3) 旧大类 id
  const legacyTab = LEGACY_TAB_MAP[t];
  if (legacyTab) {
    const wanted = s || LEGACY_TAB_DEFAULT_SUB[t] || '';
    return { tab: legacyTab, sub: pickSub(legacyTab, wanted) };
  }

  // 4) 仅含小类的旧单段链接（如 #/upstream、#/settings）
  if (!s) {
    const owner = LEGACY_SUB_OWNER[t];
    if (owner) return { tab: owner, sub: pickSub(owner, t) };
  }

  return null;
}

/** 归一化任意标签值到当前有效的大类 id（含旧 id 重定向） */
export function normalizeTab(value: string): NavTab | null {
  return resolveRoute(value)?.tab ?? null;
}
