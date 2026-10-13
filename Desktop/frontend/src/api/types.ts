/** 谛听 (DITING) 核心与全模块类型定义 */

export interface ApiResponse<T = any> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
  conflict?: PortCheckResult;
}

export type UpstreamProtocol = 'PLAIN' | 'DOH' | 'DOT';

export interface UpstreamInfo {
  id: string;
  protocol: UpstreamProtocol;
  server: string;
  url?: string;
  weight?: number;
  active: boolean;
  successCount: number;
  failureCount: number;
  avgLatencyMs: number;
}

export interface BootstrapServer {
  id: string;
  name: string;
  address: string;
  weight?: number;
}

export interface BootstrapConfig {
  enabled: boolean;
  servers: BootstrapServer[];
}

export interface DNSStatus {
  running: boolean;
  listenAddresses: string[];
  allowLAN?: boolean;
  lanAddresses?: string[];
  mode: string;
  upstreams: UpstreamInfo[];
  bootstrap?: BootstrapConfig;
  conditionalRoutes?: Record<string, string>;
  enableEDE?: boolean;
  blockUnqualified?: boolean;
  blockPrivatePTR?: boolean;
  blockIPv6?: boolean;
  blockECH?: boolean;
  ednsPadding?: number;
}

export interface AdapterState {
  id: string;
  name: string;
  index: number;
  description: string;
  ipv4DHCP: boolean;
  ipv6DHCP: boolean;
  ipv4DNS: string[];
  ipv6DNS: string[];
}

export interface TakeoverStatus {
  active: boolean;
  adapters: AdapterState[];
}

export interface MetricsStatus {
  totalQueries: number;
  successQueries: number;
  failedQueries: number;
  avgLatencyMs: number;
  qps: number;
}

export interface CacheStats {
  enabled: boolean;
  totalHits: number;
  totalMisses: number;
  staleHits: number;
  negativeHits: number;
  hitRatio: number;
  entryCount: number;
  maxEntries: number;
  evictionCount: number;
}

export interface FilterStats {
  enabled: boolean;
  totalRules: number;
  activeLists: number;
  totalQueries: number;
  blockedQueries: number;
  allowedQueries: number;
  blockRate: number;
}

export interface StatusResponse {
  version: string;
  pid: number;
  uptimeSeconds: number;
  dns: DNSStatus;
  takeover: TakeoverStatus;
  metrics: MetricsStatus;
  cache?: CacheStats;
  filter?: FilterStats;
  hosts?: HostsConfig;
  rebind?: RebindConfig;
  access?: AccessControlConfig;
  rateLimit?: RateLimitConfig;
  dns64?: DNS64Config;
  rewrite?: RewriteConfig;
  ecs?: ECSConfig;
  clients?: ClientInfo[];
}

export type CacheMode = 'follow_dns_ttl' | 'limit_max_ttl' | 'fixed_ttl' | string;

export interface CacheConfig {
  enabled: boolean;
  maxEntries: number;
  mode: CacheMode;
  maxTtlSeconds: number;
  fixedTtlSeconds: number;
  minTtlEnabled: boolean;
  minTtlSeconds: number;
  staleFallbackEnabled: boolean;
  staleFallbackSeconds: number;
  negativeTtlEnabled: boolean;
  negativeTtlSeconds: number;
  optimistic: boolean;
}

export interface CacheEntryItem {
  domain: string;
  qtype: string;
  ttl: number;
  originalTtl: number;
  remainingTtl: number;
  expiresAt: number;
  staleUntil: number;
  hitCount: number;
  lastHitAt: number;
  isNegative: boolean;
  status: 'fresh' | 'stale';
  ipList?: string[];
}

export interface CacheEntriesResponse {
  total: number;
  entries: CacheEntryItem[];
}

export interface CacheDomainStat {
  domain: string;
  qtype: string;
  hitCount: number;
  lastHitAt: number;
}

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

export interface AdapterInfo {
  id: string;
  name: string;
  description: string;
  index: number;
  status: string;
  gateway: string;
  ipv4DHCP: boolean;
  ipv6DHCP: boolean;
  ipv4DNS: string[];
  ipv6DNS: string[];
  isPhysical: boolean;
}

export interface PortConflictInfo {
  port?: number;
  protocol: string;
  localAddress: string;
  pid: number;
  processName: string;
  serviceName?: string;
  isICS: boolean;
  isSelf: boolean;
  diagnosis?: string;
}

export interface PortCheckResult {
  available: boolean;
  hasICS: boolean;
  canAutofix?: boolean;
  diagnostic: string;
  conflicts: PortConflictInfo[];
}

export interface QueryEventData {
  domain: string;
  qtype: string;
  clientIP: string;
  clientName?: string;
  durationMs: number;
  success: boolean;
  rcode?: string;
  cacheHit?: string;
  blocked?: boolean;
  filterRule?: string;
  filterReason?: string;
  errorMessage?: string;
}

export interface FilterList {
  id: string;
  name: string;
  url: string;
  enabled: boolean;
  rulesCount: number;
  lastUpdated: number;
  checksum?: string;
}

export type BlockMode = 'null_ip' | 'nxdomain' | 'refused' | string;

export interface FilterConfig {
  enabled: boolean;
  blockMode: BlockMode;
  blockingIPv4: string;
  blockingIPv6: string;
  customRules: string[];
  lists: FilterList[];
  updateIntervalHours: number;
  dataDir?: string;
}

export interface CheckHostResult {
  blocked: boolean;
  action: 'block' | 'allow' | 'pass' | string;
  matchedRule?: string;
  listName?: string;
  reason?: string;
}

export interface WebSocketEvent {
  type: 'query' | 'metrics' | 'takeover' | 'dns' | 'alert' | 'upstream' | 'background';
  timestamp: number;
  data: any;
}

export interface ProviderConfig {
  id: string;
  protocol: UpstreamProtocol;
  server: string;
  url?: string;
  weight?: number;
}

export interface UpstreamConfigureRequest {
  mode: string;
  providers: ProviderConfig[];
  bootstrap?: BootstrapConfig;
  conditionalRoutes?: Record<string, string>;
}

export interface TestUpstreamRequest {
  protocol: string;
  server: string;
  url?: string;
}

export interface TestUpstreamResponse {
  success: boolean;
  latencyMs: number;
  error?: string;
}

export interface CoreServiceStatus {
  installed: boolean;
  running: boolean;
  state: 'running' | 'stopped' | 'not_installed' | 'start_pending' | 'stop_pending' | 'unknown' | string;
  stateText: string;
  executablePath: string;
  isElevated: boolean;
  canInstall: boolean;
  message?: string;
}

export interface LANStatusResponse {
  allowLAN: boolean;
  listenAddresses: string[];
  lanAddresses: string[];
  firewallAllowed: boolean;
}

export interface ConfigureLANRequest {
  allowLAN: boolean;
  configureFirewall?: boolean;
}

export interface AuthStatusResponse {
  initialized: boolean;
  webEnabled: boolean;
  authenticated: boolean;
  username: string;
  locked: boolean;
  lockoutRemainingSec: number;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  username: string;
  expiresAt: number;
}

export interface SetupAuthRequest {
  username: string;
  password: string;
}

export interface ChangePasswordRequest {
  username: string;
  oldPassword?: string;
  newPassword: string;
}

export interface WebStatusResponse {
  enabled: boolean;
  port: number;
  listenAddress: string;
  lanAddresses: string[];
  webUrls: string[];
  firewallAllowed: boolean;
  initialized: boolean;
  username: string;
}

export interface ConfigureWebRequest {
  enabled: boolean;
  port?: number;
  configureFirewall?: boolean;
}

export interface LoginBackgroundConfig {
  type: 'none' | 'url' | 'custom';
  url: string;
  opacity: number;
  blur: number;
  updatedAt: number;
}

export interface SetLoginBackgroundRequest {
  type?: 'none' | 'url' | 'custom';
  url: string;
  opacity: number;
  blur: number;
  adminPassword?: string;
}

export interface RewriteRule {
  domain: string;
  ips?: string[];
  cname?: string;
  txt?: string[];
  ttl?: number;
}

export interface RewriteConfig {
  enabled: boolean;
  ttl: number;
  rules: RewriteRule[];
}

export interface ConfigureRewriteRequest {
  enabled?: boolean;
  ttl?: number;
  rules?: RewriteRule[];
}

export interface ClientAlias {
  ip: string;
  name: string;
}

export interface ClientInfo {
  ip: string;
  name: string;
  queryCount: number;
  lastSeen: number;
}

export interface ConfigureClientAliasRequest {
  ip: string;
  name: string;
}

export interface AccessControlConfig {
  allowedClients: string[];
  blockedClients: string[];
}

export interface ConfigureAccessRequest {
  allowedClients: string[];
  blockedClients: string[];
}

export interface RateLimitConfig {
  enabled: boolean;
  rate: number;
  burst: number;
  ipv4Prefix: number;
  ipv6Prefix: number;
  exemptIPs: string[];
}

export interface ConfigureRateLimitRequest {
  enabled: boolean;
  rate: number;
  burst: number;
  ipv4Prefix: number;
  ipv6Prefix: number;
  exemptIPs: string[];
}

export interface DNS64Config {
  enabled: boolean;
  prefixes: string[];
}

export interface ConfigureDNS64Request {
  enabled: boolean;
  prefixes: string[];
}

export interface ConfigureBlockUnqualifiedRequest {
  blockUnqualified: boolean;
}

export interface ConfigurePrivacyRequest {
  enableEDE?: boolean;
  blockPrivatePTR?: boolean;
  blockIPv6?: boolean;
  blockECH?: boolean;
  ednsPadding?: number;
}

export interface ECSConfig {
  enabled: boolean;
  stripECS: boolean;
  customSubnet: string;
}

export interface ConfigureECSRequest {
  enabled?: boolean;
  stripECS?: boolean;
  customSubnet?: string;
}

export interface HostsConfig {
  enabled: boolean;
  useSystemHost: boolean;
  customFiles: string[];
  ttl: number;
}

export interface RebindConfig {
  enabled: boolean;
  allowedDomains: string[];
}

export interface ConfigureRebindRequest {
  enabled?: boolean;
  allowedDomains?: string[];
}

export interface ConfigureHostsRequest {
  enabled?: boolean;
  useSystemHost?: boolean;
  customFiles?: string[];
  ttl?: number;
}


