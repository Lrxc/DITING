/** 查询历史与统计中心类型定义（从 types.ts 拆出以控制单文件长度） */

/** 单条查询的处置状态 */
export type QueryStatus = 'allowed' | 'blocked' | 'cached' | 'stale' | 'error';

/** 单条查询历史记录 */
export interface QueryLogRecord {
  ts: number;
  domain: string;
  qtype?: string;
  clientIP?: string;
  clientName?: string;
  durationMs: number;
  rcode?: string;
  status: QueryStatus;
  cached?: boolean;
  cacheHit?: string;
  blocked?: boolean;
  filterRule?: string;
  filterReason?: string;
  protocol?: string;
  error?: string;
}

export interface QueryLogSearchQuery {
  domain?: string;
  client?: string;
  qtype?: string;
  rcode?: string;
  status?: string;
  startMs?: number;
  endMs?: number;
  limit?: number;
  offset?: number;
  /** 内存环不足时是否回退扫描落盘历史，默认 true */
  includeFile?: boolean;
}

export interface QueryLogSearchResult {
  total: number;
  entries: QueryLogRecord[];
}

export interface QueryNameCount {
  name: string;
  count: number;
}

export interface QueryHourPoint {
  hourMs: number;
  queries: number;
  blocked: number;
  cached: number;
}

export interface QuerySummary {
  queries: number;
  blocked: number;
  cached: number;
  forwarded: number;
  errors: number;
  avgResponseMs: number;
  cacheHitRate: number;
}

export interface QueryStatsResponse {
  summary: QuerySummary;
  perHour: QueryHourPoint[] | null;
  topDomains: QueryNameCount[] | null;
  topBlocked: QueryNameCount[] | null;
  topClients: QueryNameCount[] | null;
}

export interface QueryLogConfig {
  enabled: boolean;
  maxEntries: number;
  fileEnabled: boolean;
  retentionDays: number;
  logClients: boolean;
  dataDir: string;
}
