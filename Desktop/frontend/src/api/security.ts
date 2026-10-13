import type {
  ConfigureAccessRequest,
  ConfigureRateLimitRequest,
  ConfigureDNS64Request,
  ConfigureBlockUnqualifiedRequest,
  ConfigurePrivacyRequest,
  ConfigureECSRequest,
  ConfigureRebindRequest,
  ConfigureHostsRequest,
} from './types';
import type { HttpTransport } from './http';

/** 安全防护、访问控制、速率限制与协议隐私 API */
export class SecurityApi {
  constructor(private http: HttpTransport) {}

  /** 配置客户端访问控制黑白名单 */
  public async configureAccess(req: ConfigureAccessRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/access/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 配置客户端请求速率限制 */
  public async configureRateLimit(req: ConfigureRateLimitRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/ratelimit/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 配置 DNS64 IPv6-only 网络过渡合成 */
  public async configureDNS64(req: ConfigureDNS64Request): Promise<void> {
    await this.http.request<void>('/api/v1/dns/dns64/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 配置单标签域名防外泄 */
  public async configureBlockUnqualified(req: ConfigureBlockUnqualifiedRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/unqualified/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 配置私有反向 PTR、IPv6过滤、ECH防护与 EDNS 填充隐私策略 */
  public async configurePrivacy(req: ConfigurePrivacyRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/privacy/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 配置 EDNS0 Client Subnet (ECS) 客户端子网隐私策略 */
  public async configureECS(req: ConfigureECSRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/ecs/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 配置 DNS Rebinding 攻击防御与白名单 */
  public async configureRebind(req: ConfigureRebindRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/rebind/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  /** 配置系统与自定义 Hosts 本地解析 */
  public async configureHosts(req: ConfigureHostsRequest): Promise<void> {
    await this.http.request<void>('/api/v1/dns/hosts/configure', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }
}
