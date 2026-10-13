import type {
  RewriteRule,
  ConfigureRewriteRequest,
  ClientInfo,
  ConfigureClientAliasRequest,
  ClientGroupConfigResponse,
  ConfigureClientGroupsRequest,
} from './types';
import type { HttpTransport } from './http';

/** 自定义 DNS 重写与客户端主机名管理 API */
export class RewriteApi {
  constructor(private http: HttpTransport) {}

  /** 获取所有自定义重写规则 */
  public getRewriteRules(): Promise<RewriteRule[]> {
    return this.http.request<RewriteRule[]>('/api/v1/dns/rewrite/rules');
  }

  /** 添加或更新单条自定义重写规则 */
  public async addRewriteRule(rule: RewriteRule): Promise<void> {
    await this.http.request<void>('/api/v1/dns/rewrite/rules', {
      method: 'POST',
      body: JSON.stringify(rule),
    });
  }

  /** 删除指定域名的重写规则 */
  public async deleteRewriteRule(domain: string): Promise<void> {
    await this.http.request<void>(`/api/v1/dns/rewrite/rules?domain=${encodeURIComponent(domain)}`, {
      method: 'DELETE',
    });
  }

  /** 整体配置重写引擎（开关、默认 TTL、规则集） */
  public async configureRewrite(req: ConfigureRewriteRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/rewrite/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 获取所有已识别的局域网活跃客户端 */
  public getClients(): Promise<ClientInfo[]> {
    return this.http.request<ClientInfo[]>('/api/v1/clients');
  }

  /** 设置或清除客户端友好静态别名 */
  public async configureClientAlias(req: ConfigureClientAliasRequest): Promise<void> {
    await this.http.request<void>('/api/v1/clients/alias', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 获取客户端分组配置与可选订阅列表 */
  public getClientGroups(): Promise<ClientGroupConfigResponse> {
    return this.http.request<ClientGroupConfigResponse>('/api/v1/clients/groups');
  }

  /** 更新客户端分组配置（分组级规则集与拦截方式） */
  public async configureClientGroups(req: ConfigureClientGroupsRequest): Promise<void> {
    await this.http.request<void>('/api/v1/clients/groups', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }
}
