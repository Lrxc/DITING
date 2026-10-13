import type {
  QueryLogConfig,
  QueryLogSearchQuery,
  QueryLogSearchResult,
  QueryStatsResponse,
} from './types';
import type { HttpTransport } from './http';

/** 查询历史与统计中心 */
export class QueryLogApi {
  constructor(private http: HttpTransport) {}

  public search(query: QueryLogSearchQuery = {}): Promise<QueryLogSearchResult> {
    const params = new URLSearchParams();
    if (query.domain) params.set('domain', query.domain);
    if (query.client) params.set('client', query.client);
    if (query.qtype) params.set('qtype', query.qtype);
    if (query.rcode) params.set('rcode', query.rcode);
    if (query.status) params.set('status', query.status);
    if (query.startMs) params.set('startMs', query.startMs.toString());
    if (query.endMs) params.set('endMs', query.endMs.toString());
    if (query.limit) params.set('limit', query.limit.toString());
    if (query.offset) params.set('offset', query.offset.toString());
    if (query.includeFile === false) params.set('includeFile', '0');
    const qs = params.toString();
    return this.http.request<QueryLogSearchResult>(`/api/v1/querylog/search${qs ? `?${qs}` : ''}`);
  }

  public getStats(): Promise<QueryStatsResponse> {
    return this.http.request<QueryStatsResponse>('/api/v1/querylog/stats');
  }

  public async clear(): Promise<void> {
    await this.http.request<void>('/api/v1/querylog/clear', { method: 'POST' });
  }

  public getConfig(): Promise<QueryLogConfig> {
    return this.http.request<QueryLogConfig>('/api/v1/querylog/config');
  }

  public async updateConfig(config: QueryLogConfig): Promise<void> {
    await this.http.request<void>('/api/v1/querylog/config', {
      method: 'POST',
      body: JSON.stringify(config),
    });
  }
}
