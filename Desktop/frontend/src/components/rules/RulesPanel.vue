<script setup lang="ts">
import { ref, computed } from 'vue';
import { ipc } from '../../api/ipc';
import type { FilterStats, FilterConfig } from '../../api/types';
import StatusBadge from '../../components/StatusBadge.vue';
import M3Icon from '../../components/M3Icon.vue';
import MetricStrip, { type StatEntry } from '../../components/ui/MetricStrip.vue';
import ToastBanner from '../../components/ToastBanner.vue';
import RuleListsPanel from '../../components/RuleListsPanel.vue';
import RuleTestPanel from '../../components/RuleTestPanel.vue';
import { revertSwitch, extractSwitchValue } from '../../utils/switch';
import { usePoller } from '../../composables/usePoller';
import { useToast } from '../../composables/useToast';

const stats = ref<FilterStats>({
  enabled: true,
  totalRules: 0,
  activeLists: 0,
  totalQueries: 0,
  blockedQueries: 0,
  allowedQueries: 0,
  blockRate: 0,
});

const config = ref<FilterConfig>({
  enabled: true,
  blockMode: 'null_ip',
  blockingIPv4: '0.0.0.0',
  blockingIPv6: '::',
  customRules: [],
  lists: [],
  updateIntervalHours: 24,
});

const listCount = ref(0);
const customRulesText = ref('');
/** 子页签由上层 RulesView 的分段控件驱动（hash 可直达） */
const activeTab = defineModel<string>('sub', { default: 'lists' });

const ruleStats = computed<StatEntry[]>(() => [
  { label: '已拦截请求', value: stats.value.blockedQueries.toLocaleString(), unit: '次', tone: 'seal' },
  { label: '拦截比例', value: stats.value.blockRate.toFixed(1), unit: '%' },
  { label: '有效防护规则', value: stats.value.totalRules.toLocaleString(), unit: '条' },
  { label: '已启用规则库', value: stats.value.activeLists, unit: '个' },
]);

const saving = ref(false);
const { toast, show: showToast, dismiss } = useToast();

async function fetchData() {
  try {
    const [st, cfg, l, rules] = await Promise.all([
      ipc.getFilterStats(),
      ipc.getFilterConfig(),
      ipc.getFilterLists(),
      ipc.getCustomRules(),
    ]);
    if (st) stats.value = st;
    if (cfg) config.value = cfg;
    listCount.value = (l || []).length;
    customRulesText.value = (rules || []).join('\n');
  } catch (err: any) {
    showToast(err?.message || '获取规则数据失败', true);
  }
}

async function toggleMasterSwitch(e: Event) {
  const nextVal = extractSwitchValue(e);
  try {
    saving.value = true;
    await ipc.updateFilterConfig({ enabled: nextVal });
    stats.value.enabled = nextVal;
    config.value.enabled = nextVal;
    showToast(`规则拦截防护已${nextVal ? '开启' : '关闭'}`);
  } catch (err: any) {
    revertSwitch(e, !nextVal);
    showToast(err?.message || '更新开关失败', true);
  } finally {
    saving.value = false;
  }
}

async function saveCustomRules() {
  try {
    saving.value = true;
    const lines = customRulesText.value
      .split('\n')
      .map((l) => l.trim())
      .filter((l) => l.length > 0);
    await ipc.setCustomRules(lines);
    showToast(`自定义规则已保存生效 (共 ${lines.length} 条)`);
    await fetchData();
  } catch (err: any) {
    showToast(err?.message || '保存自定义规则失败', true);
  } finally {
    saving.value = false;
  }
}

async function saveConfig() {
  try {
    saving.value = true;
    await ipc.updateFilterConfig({
      blockMode: config.value.blockMode,
      blockingIPv4: config.value.blockingIPv4,
      blockingIPv6: config.value.blockingIPv6,
      updateIntervalHours: Number(config.value.updateIntervalHours) || 24,
    });
    showToast('拦截策略配置已保存');
    await fetchData();
  } catch (err: any) {
    showToast(err?.message || '保存策略失败', true);
  } finally {
    saving.value = false;
  }
}

usePoller(fetchData, 4000);
</script>

<template>
  <div class="space-y-6 max-w-7xl mx-auto pb-12 select-none">
    <!-- 顶部消息提示 -->
    <ToastBanner v-if="toast" :message="toast.message" :type="toast.isError ? 'error' : 'success'" dismissible @dismiss="dismiss" />

    <!-- 顶栏核心主控与指标卡片 -->
    <div class="app-card p-5 space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-3 border-b border-surface-border-sub">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
            <M3Icon name="shield" :size="18" class="text-text-main" />
          </div>
          <div>
            <div class="flex items-center gap-2 flex-wrap">
              <h2 class="section-title text-[14px]">广告拦截与安全防护引擎</h2>
              <StatusBadge
                :status="stats.enabled ? 'active' : 'inactive'"
                :text="stats.enabled ? '防护生效中' : '防护已暂停'"
                size="sm"
                :pulse="stats.enabled"
              />
            </div>
            <p class="text-[11px] text-text-muted mt-0.5">
              毫秒级识别并阻断弹窗广告、隐私追踪与钓鱼恶意网址
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2 shrink-0">
          <span class="text-[11px] font-semibold text-text-sub hidden sm:inline">
            {{ stats.enabled ? '规则拦截开启' : '拦截已关闭' }}
          </span>
          <md-switch
            :selected="stats.enabled"
            @change="toggleMasterSwitch"
            :disabled="saving"
          />
        </div>
      </div>

      <!-- 拦截指标带 -->
      <MetricStrip :stats="ruleStats" />
    </div>

    <!-- TAB 1: 订阅规则源列表 -->
    <RuleListsPanel
      v-if="activeTab === 'lists'"
      @error="(m) => showToast(m, true)"
      @success="(m) => showToast(m)"
      @changed="fetchData"
    />

    <!-- TAB 2: 自定义规则编辑器 (宽屏两列对等布局，窄屏单列) -->
    <div v-if="activeTab === 'custom'" class="grid grid-cols-1 lg:grid-cols-2 gap-5 items-stretch">
      <!-- 左侧：编写指南卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center gap-2.5 pb-3 border-b border-surface-border-sub">
            <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
              <M3Icon name="help_outline" :size="16" class="text-text-main" />
            </div>
            <div>
              <h3 class="section-title text-[14px]">AdGuard 语法速查指南</h3>
              <span class="text-[11px] text-text-muted">主流广告过滤与信任规则编写格式</span>
            </div>
          </div>

          <div class="space-y-2.5 mt-3 text-xs">
            <div class="app-card-sub p-2.5 space-y-1">
              <div class="flex items-center justify-between">
                <span class="font-semibold text-text-main">域名与子域名拦截</span>
                <span class="text-[10px] text-text-muted">常用格式</span>
              </div>
              <code class="font-mono text-accent-seal text-[12px] block">||example.com^</code>
              <p class="text-[11px] text-text-muted">匹配并拦截 example.com 及其所有子域名（如 ad.example.com）</p>
            </div>

            <div class="app-card-sub p-2.5 space-y-1">
              <div class="flex items-center justify-between">
                <span class="font-semibold text-text-main">白名单信任放行</span>
                <span class="text-[10px] text-text-muted">豁免放行</span>
              </div>
              <code class="font-mono text-status-success text-[12px] block">@@||safe.com^</code>
              <p class="text-[11px] text-text-muted">即便其他规则命中，也始终允许解析 safe.com</p>
            </div>

            <div class="app-card-sub p-2.5 space-y-1">
              <div class="flex items-center justify-between">
                <span class="font-semibold text-text-main">Hosts 文件格式</span>
                <span class="text-[10px] text-text-muted">传统兼容</span>
              </div>
              <code class="font-mono text-text-main text-[12px] block">0.0.0.0 bad.com</code>
              <p class="text-[11px] text-text-muted">兼容标准 Hosts 文件屏蔽特定恶意域名的语法</p>
            </div>

            <div class="app-card-sub p-2.5 space-y-1">
              <div class="flex items-center justify-between">
                <span class="font-semibold text-text-main">最高优先级强力拦截</span>
                <span class="text-[10px] text-text-muted">强制优先</span>
              </div>
              <code class="font-mono text-accent-seal text-[12px] block">||urgent.com^$important</code>
              <p class="text-[11px] text-text-muted">忽略所有白名单规则，强制拦截该域名</p>
            </div>
          </div>
        </div>

        <div class="text-[11px] text-text-muted pt-2 border-t border-surface-border-sub">
          提示：每行一条规则，以 <code class="font-mono">!</code> 开头的行为注释。
        </div>
      </div>

      <!-- 右侧：编辑器卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div class="flex flex-col flex-1">
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="tune" :size="16" class="text-text-main" />
              </div>
              <div>
                <h3 class="section-title text-[14px]">自定义规则文本编辑器</h3>
                <span class="text-[11px] text-text-muted">编辑用户专属的过滤拦截或放行清单</span>
              </div>
            </div>

            <button type="button" @click="saveCustomRules" :disabled="saving" class="app-btn-primary app-btn-compact">
              <M3Icon name="check" :size="14" />
              <span>{{ saving ? '保存中...' : '保存规则' }}</span>
            </button>
          </div>

          <div class="mt-3 flex-1 flex flex-col">
            <div class="app-card-sub p-2 focus-within:border-accent-seal flex-1 flex flex-col">
              <textarea
                v-model="customRulesText"
                rows="15"
                placeholder="在此输入自定义拦截或信任规则，每行一条...&#10;||ad.example.com^&#10;@@||mysite.com^"
                class="w-full h-full min-h-[280px] bg-transparent border-0 resize-y p-2 text-xs font-mono text-text-main focus:outline-none placeholder:text-text-muted"
              ></textarea>
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between text-[11px] text-text-muted pt-2 border-t border-surface-border-sub">
          <span>行数统计: {{ customRulesText ? customRulesText.split('\n').filter(l => l.trim()).length : 0 }} 条规则</span>
          <span>修改后点击保存立即热重载生效</span>
        </div>
      </div>
    </div>

    <!-- TAB 3: 域名规则检测工具 -->
    <RuleTestPanel v-if="activeTab === 'test'" @error="(m) => showToast(m, true)" />

    <!-- TAB 4: 策略配置 (宽屏两列对等布局，窄屏单列) -->
    <div v-if="activeTab === 'config'" class="grid grid-cols-1 lg:grid-cols-2 gap-5 items-stretch">
      <!-- 左侧：响应动作卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center gap-2.5 pb-3 border-b border-surface-border-sub">
            <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
              <M3Icon name="shield" :size="16" class="text-text-main" />
            </div>
            <div>
              <h3 class="section-title text-[14px]">拦截响应处理方式</h3>
              <span class="text-[11px] text-text-muted">当遇到被拦截网址时向客户端返回的行为</span>
            </div>
          </div>

          <div class="space-y-3 mt-3.5 text-xs">
            <label
              class="flex items-start gap-3 cursor-pointer p-3 rounded-md border transition-all"
              :class="config.blockMode === 'null_ip' ? 'bg-brand-container/40 border-brand-primary/40' : 'border-surface-border-sub bg-surface-card-sub hover:bg-surface-hover'"
            >
              <input type="radio" v-model="config.blockMode" value="null_ip" class="accent-brand-primary mt-0.5" />
              <div>
                <div class="font-bold text-text-main text-[13px]">立即返回空地址 (0.0.0.0 / ::)</div>
                <div class="text-[11px] text-text-sub mt-0.5 leading-relaxed">
                  强烈推荐。以极速终止广告和弹窗资源加载，不会引起浏览器反复重发请求。
                </div>
              </div>
            </label>

            <label
              class="flex items-start gap-3 cursor-pointer p-3 rounded-md border transition-all"
              :class="config.blockMode === 'nxdomain' ? 'bg-brand-container/40 border-brand-primary/40' : 'border-surface-border-sub bg-surface-card-sub hover:bg-surface-hover'"
            >
              <input type="radio" v-model="config.blockMode" value="nxdomain" class="accent-brand-primary mt-0.5" />
              <div>
                <div class="font-bold text-text-main text-[13px]">提示域名不存在 (NXDOMAIN)</div>
                <div class="text-[11px] text-text-sub mt-0.5 leading-relaxed">
                  告知客户端该网址不存在，部分标准网络软件将放弃进一步连接探测。
                </div>
              </div>
            </label>

            <label
              class="flex items-start gap-3 cursor-pointer p-3 rounded-md border transition-all"
              :class="config.blockMode === 'refused' ? 'bg-brand-container/40 border-brand-primary/40' : 'border-surface-border-sub bg-surface-card-sub hover:bg-surface-hover'"
            >
              <input type="radio" v-model="config.blockMode" value="refused" class="accent-brand-primary mt-0.5" />
              <div>
                <div class="font-bold text-text-main text-[13px]">明确拒绝请求 (REFUSED)</div>
                <div class="text-[11px] text-text-sub mt-0.5 leading-relaxed">
                  向客户端返回拒绝码，表示请求被 DNS 防火墙安全策略主动阻止。
                </div>
              </div>
            </label>
          </div>
        </div>

        <div class="text-[11px] text-text-muted pt-2 border-t border-surface-border-sub">
          当前模式: <span class="font-mono text-text-main font-semibold">{{ config.blockMode }}</span>
        </div>
      </div>

      <!-- 右侧：更新周期与参数卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center gap-2.5 pb-3 border-b border-surface-border-sub">
            <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
              <M3Icon name="sync" :size="16" class="text-text-main" />
            </div>
            <div>
              <h3 class="section-title text-[14px]">规则自动更新与网络参数</h3>
              <span class="text-[11px] text-text-muted">订阅规则库同步周期与重定向 IP 设置</span>
            </div>
          </div>

          <div class="space-y-3.5 mt-3 text-xs">
            <div>
              <label class="text-[11px] font-semibold text-text-muted block mb-1">规则库自动同步周期 (小时)</label>
              <input
                v-model="config.updateIntervalHours"
                type="number"
                min="1"
                class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs focus:outline-none focus:border-accent-seal text-text-main font-mono"
              />
              <span class="text-[10px] text-text-muted mt-1 block">定时从远程规则源获取更新并增量载入，默认 24 小时</span>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
              <div>
                <label class="text-[11px] font-semibold text-text-muted block mb-1">阻断 IPv4 响应地址</label>
                <input
                  v-model="config.blockingIPv4"
                  type="text"
                  class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs focus:outline-none focus:border-accent-seal text-text-main font-mono"
                />
              </div>

              <div>
                <label class="text-[11px] font-semibold text-text-muted block mb-1">阻断 IPv6 响应地址</label>
                <input
                  v-model="config.blockingIPv6"
                  type="text"
                  class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs focus:outline-none focus:border-accent-seal text-text-main font-mono"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="pt-2 border-t border-surface-border-sub flex justify-end">
          <button type="button" @click="saveConfig" :disabled="saving" class="app-btn-primary">
            <M3Icon name="check" :size="15" />
            <span>{{ saving ? '保存中...' : '保存策略与参数' }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
