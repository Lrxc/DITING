<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { ipc } from '../../api/ipc';
import type { StatusResponse, ECSConfig, RebindConfig, HostsConfig } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import StatusBadge from '../StatusBadge.vue';
import ToastBanner from '../ToastBanner.vue';
import { extractSwitchValue } from '../../utils/switch';
import { useToast } from '../../composables/useToast';

const status = ref<StatusResponse | null>(null);
const loading = ref(false);
const saving = ref(false);

// 协议增强与隐私状态
const blockPrivatePTR = ref(true);
const blockUnqualified = ref(false);
const blockECH = ref(false);
const blockIPv6 = ref(false);
const ednsPadding = ref(128);

// ECS 策略状态
const ecsConfig = ref<ECSConfig>({
  enabled: true,
  stripECS: true,
  customSubnet: '',
});

// Rebind 与 Hosts 状态展示
const rebindConfig = ref<RebindConfig>({
  enabled: true,
  allowedDomains: [],
});
const hostsConfig = ref<HostsConfig>({
  enabled: true,
  useSystemHost: true,
  customFiles: [],
  ttl: 60,
});

const { toast, show: showToast, dismiss } = useToast();

async function loadData() {
  loading.value = true;
  try {
    const st = await ipc.getStatus();
    if (st) {
      status.value = st;
      blockPrivatePTR.value = st.dns?.blockPrivatePTR ?? true;
      blockUnqualified.value = st.dns?.blockUnqualified ?? false;
      blockECH.value = st.dns?.blockECH ?? false;
      blockIPv6.value = st.dns?.blockIPv6 ?? false;
      ednsPadding.value = st.dns?.ednsPadding ?? 128;

      if (st.ecs) {
        ecsConfig.value = st.ecs;
      }
      if (st.rebind) {
        rebindConfig.value = st.rebind;
      }
      if (st.hosts) {
        hostsConfig.value = st.hosts;
      }
    }
  } catch (err: any) {
    showToast(err.message || '加载安全隐私配置失败', true);
  } finally {
    loading.value = false;
  }
}

async function handleTogglePrivatePTR(e: Event) {
  const nextVal = extractSwitchValue(e);
  blockPrivatePTR.value = nextVal;
  await savePrivacyConfig();
}

async function handleToggleIPv6(e: Event) {
  const nextVal = extractSwitchValue(e);
  blockIPv6.value = nextVal;
  await savePrivacyConfig();
}

async function handleToggleECH(e: Event) {
  const nextVal = extractSwitchValue(e);
  blockECH.value = nextVal;
  await savePrivacyConfig();
}

async function handlePaddingChange() {
  await savePrivacyConfig();
}

async function savePrivacyConfig() {
  saving.value = true;
  try {
    await ipc.configurePrivacy({
      blockPrivatePTR: blockPrivatePTR.value,
      blockIPv6: blockIPv6.value,
      blockECH: blockECH.value,
      ednsPadding: Number(ednsPadding.value) || 0,
    });
    showToast('隐私保护与协议增强策略已更新并即时生效');
  } catch (err: any) {
    showToast(err.message || '更新隐私配置失败', true);
  } finally {
    saving.value = false;
  }
}

async function handleToggleUnqualified(e: Event) {
  const nextVal = extractSwitchValue(e);
  saving.value = true;
  try {
    await ipc.configureBlockUnqualified({
      blockUnqualified: nextVal,
    });
    blockUnqualified.value = nextVal;
    showToast(`防单标签域名外泄已${nextVal ? '开启' : '关闭'}`);
  } catch (err: any) {
    showToast(err.message || '更新单标签防外泄失败', true);
  } finally {
    saving.value = false;
  }
}

async function handleToggleECSEnabled(e: Event) {
  const nextVal = extractSwitchValue(e);
  ecsConfig.value.enabled = nextVal;
  await saveECSConfig();
}

async function handleToggleECSStrip(e: Event) {
  const nextVal = extractSwitchValue(e);
  ecsConfig.value.stripECS = nextVal;
  await saveECSConfig();
}

async function saveECSConfig() {
  saving.value = true;
  try {
    await ipc.configureECS({
      enabled: ecsConfig.value.enabled,
      stripECS: ecsConfig.value.stripECS,
      customSubnet: (ecsConfig.value.customSubnet || '').trim(),
    });
    showToast('EDNS Client Subnet 客户端子网策略已更新');
  } catch (err: any) {
    showToast(err.message || '更新 ECS 配置失败', true);
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  loadData();
});
</script>

<template>
  <div class="space-y-6 select-none max-w-7xl mx-auto pb-12">
    <ToastBanner v-if="toast" :message="toast.message" :type="toast.isError ? 'error' : 'success'" dismissible @dismiss="dismiss" />

    <!-- 宽屏两列网格：左侧隐私与协议增强，右侧 ECS 客户端子网 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 items-stretch">
      <!-- 1. 协议隐私与外泄防御卡片 -->
      <AppCard
        title="协议隐私与外泄防御"
        subtitle="阻断局域网私有 PTR 与单标签域名泄露，防御 ECH 加密绕过与报文侧信道嗅探"
        icon="shield"
      >
        <div class="space-y-4 mt-3 text-xs">
          <!-- 私有 PTR 防泄露 -->
          <div class="app-card-sub p-3 space-y-2">
            <div class="flex items-center justify-between">
              <div>
                <span class="font-bold text-text-main text-[13px]">私有反向 PTR 解析防泄露</span>
                <span class="text-[10px] text-text-muted ml-2 font-mono">RFC 6303 / RFC 7793</span>
              </div>
              <md-switch
                :selected="blockPrivatePTR"
                :disabled="saving"
                @change="handleTogglePrivatePTR"
              />
            </div>
            <p class="text-[11px] text-text-sub leading-relaxed">
              阻断局域网私有 IP（如 <code class="font-mono">192.168.x.x</code>、<code class="font-mono">10.x.x.x</code>、<code class="font-mono">fd00::/8</code>）的反向域名 PTR 查询向公共上游泄露。命中且未在本地绑定的反查将直接返回权威 NXDOMAIN。
            </p>
          </div>

          <!-- 防单标签域名外泄 -->
          <div class="app-card-sub p-3 space-y-2">
            <div class="flex items-center justify-between">
              <div>
                <span class="font-bold text-text-main text-[13px]">阻止无点单标签域名外泄</span>
                <span class="text-[10px] text-text-muted ml-2">Intranet Single-Label</span>
              </div>
              <md-switch
                :selected="blockUnqualified"
                :disabled="saving"
                @change="handleToggleUnqualified"
              />
            </div>
            <p class="text-[11px] text-text-sub leading-relaxed">
              拦截没有顶级域名点分隔的内部单标签请求（如 <code class="font-mono">router</code>、<code class="font-mono">printer</code>），防止因拼写错误或内部搜索域配置缺失导致内网主机名泄漏至公共递归解析器。
            </p>
          </div>

          <!-- ECH 防护 -->
          <div class="app-card-sub p-3 space-y-2">
            <div class="flex items-center justify-between">
              <div>
                <span class="font-bold text-text-main text-[13px]">剥离 ECH (加密 ClientHello)</span>
                <span class="text-[10px] text-text-muted ml-2">Prevent Filter Bypass</span>
              </div>
              <md-switch
                :selected="blockECH"
                :disabled="saving"
                @change="handleToggleECH"
              />
            </div>
            <p class="text-[11px] text-text-sub leading-relaxed">
              从上游 HTTPS / SVCB 资源记录中过滤剥离 <code class="font-mono">ech</code> 参数。防止客户端使用 TLS 1.3 ECH 加密真实访问域名，保障本地防火墙与规则引擎审计不被绕过。
            </p>
          </div>

          <!-- IPv6 过滤 / 禁用 -->
          <div class="app-card-sub p-3 space-y-2">
            <div class="flex items-center justify-between">
              <div>
                <span class="font-bold text-text-main text-[13px]">过滤 IPv6 (AAAA) 查询</span>
                <span class="text-[10px] text-text-muted ml-2">IPv4-only 提速</span>
              </div>
              <md-switch
                :selected="blockIPv6"
                :disabled="saving"
                @change="handleToggleIPv6"
              />
            </div>
            <p class="text-[11px] text-text-sub leading-relaxed">
              针对所有 AAAA 查询秒级应答权威 NODATA，并在 HTTPS 查询中自动过滤 ipv6hint。在未开通或 IPv6 较差的网络环境下，彻底消除因双栈握手失败或重试导致的网页加载卡顿。
            </p>
          </div>

          <!-- EDNS Padding -->
          <div class="app-card-sub p-3 flex items-center justify-between gap-3">
            <div>
              <span class="font-bold text-text-main text-[13px] block">EDNS0 报文填充大小</span>
              <span class="text-[11px] text-text-muted mt-0.5 block">
                RFC 7830 / 8467。对加密 DNS (DoT / DoH) 填充至固定块，防范报文长度指纹侧信道分析。
              </span>
            </div>
            <select
              v-model.number="ednsPadding"
              @change="handlePaddingChange"
              :disabled="saving"
              class="h-[32px] px-2.5 rounded-md border border-surface-border bg-surface-card text-xs text-text-main font-mono shrink-0 focus:outline-none focus:border-accent-seal"
            >
              <option :value="0">0 (禁用填充)</option>
              <option :value="128">128 字节 (推荐)</option>
              <option :value="256">256 字节</option>
              <option :value="468">468 字节 (RFC 推荐)</option>
            </select>
          </div>
        </div>
      </AppCard>

      <!-- 2. EDNS Client Subnet (ECS) 隐私策略卡片 -->
      <AppCard
        title="EDNS Client Subnet (ECS 策略)"
        subtitle="RFC 7871 规范：平衡 CDN 智能就近调度与客户端出口 IP 隐私保护"
        icon="tune"
      >
        <template #badge>
          <StatusBadge
            :status="ecsConfig.enabled ? 'active' : 'inactive'"
            :text="ecsConfig.enabled ? 'ECS 策略已生效' : '策略已停用'"
            size="sm"
          />
        </template>

        <template #actions>
          <md-switch
            :selected="ecsConfig.enabled"
            :disabled="saving"
            @change="handleToggleECSEnabled"
          />
        </template>

        <div class="space-y-4 mt-3 text-xs">
          <p class="text-[12px] text-text-sub leading-relaxed">
            EDNS Client Subnet (ECS) 允许向递归解析器携带客户端所在的 IP 子网网段以优化 CDN 调度。但未经处理的原生 ECS 可能会向上游公共 DNS 泄露家庭或企业内网真实网段。
          </p>

          <!-- 剥离客户端真实子网 -->
          <div class="app-card-sub p-3 space-y-2">
            <div class="flex items-center justify-between">
              <div>
                <span class="font-bold text-text-main text-[13px]">剥离请求中的真实客户端子网</span>
                <span class="text-[10px] text-text-muted ml-2 font-mono">Strip ECS (推荐)</span>
              </div>
              <md-switch
                :selected="ecsConfig.stripECS"
                :disabled="saving || !ecsConfig.enabled"
                @change="handleToggleECSStrip"
              />
            </div>
            <p class="text-[11px] text-text-sub leading-relaxed">
              开启后，谛听在转发 DNS 查询时将主动抹除客户端发送的真实出口子网信息，保障家庭或内网各设备的网络隐私。
            </p>
          </div>

          <!-- 自定义伪装子网 -->
          <div>
            <label class="font-semibold text-text-main block mb-1">
              自定义转发子网 (Custom Subnet CIDR)
            </label>
            <input
              v-model="ecsConfig.customSubnet"
              type="text"
              placeholder="例如: 203.0.113.0/24 (留空表示不追加)"
              :disabled="saving || !ecsConfig.enabled"
              class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono focus:outline-none focus:border-accent-seal"
            />
            <span class="text-[10px] text-text-muted mt-1 block">
              可选。填写公共出口网段（如特定云服务或省份网段）向权威 DNS 伪装位置以保持最优 CDN 调度，留空则完全保密。
            </span>
          </div>

          <!-- 保存按钮 -->
          <div class="flex justify-end pt-2">
            <button
              type="button"
              @click="saveECSConfig"
              :disabled="saving || !ecsConfig.enabled"
              class="app-btn-primary"
            >
              <M3Icon name="check" :size="14" />
              <span>保存 ECS 策略</span>
            </button>
          </div>

          <!-- 核心防护状态磁贴 (Rebinding & Hosts) -->
          <div class="pt-2 border-t border-surface-border-sub space-y-3">
            <span class="text-[11px] font-bold text-text-muted block">内置安全防御模块运行状态</span>

            <div class="grid grid-cols-2 gap-2.5">
              <div class="app-card-sub p-2.5 space-y-1">
                <div class="flex items-center justify-between">
                  <span class="font-semibold text-text-main text-[12px]">DNS 重绑定防御</span>
                  <span
                    class="px-1.5 py-0.2 rounded text-[10px] font-semibold"
                    :class="rebindConfig.enabled ? 'bg-status-success-bg text-status-success' : 'bg-status-warning-bg text-status-warning'"
                  >
                    {{ rebindConfig.enabled ? '生效中' : '未开启' }}
                  </span>
                </div>
                <p class="text-[10px] text-text-muted">阻断外网域名解析为局域网私有 IP，防御内网穿透攻击</p>
              </div>

              <div class="app-card-sub p-2.5 space-y-1">
                <div class="flex items-center justify-between">
                  <span class="font-semibold text-text-main text-[12px]">本地 Hosts 解析器</span>
                  <span
                    class="px-1.5 py-0.2 rounded text-[10px] font-semibold"
                    :class="hostsConfig.enabled ? 'bg-status-success-bg text-status-success' : 'bg-status-warning-bg text-status-warning'"
                  >
                    {{ hostsConfig.enabled ? '权威应答中' : '未开启' }}
                  </span>
                </div>
                <p class="text-[10px] text-text-muted">
                  {{ hostsConfig.useSystemHost ? '已自动联动操作系统系统级 hosts 映射' : '仅使用自定义映射文件' }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </AppCard>
    </div>
  </div>
</template>
