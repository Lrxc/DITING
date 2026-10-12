<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { ipc } from '../../api/ipc';
import type { CacheStats, CacheConfig, CacheEntryItem, CacheDomainStat } from '../../api/types';
import StatusBadge from '../StatusBadge.vue';
import Panel from '../ui/Panel.vue';
import M3Icon from '../M3Icon.vue';
import AppModal from '../AppModal.vue';
import ToastBanner from '../ToastBanner.vue';
import CacheTopDomains from './CacheTopDomains.vue';
import CacheConfigCard from './CacheConfigCard.vue';
import CacheEntriesTable from './CacheEntriesTable.vue';
import { revertSwitch, extractSwitchValue } from '../../utils/switch';
import { usePoller } from '../../composables/usePoller';
import { useToast } from '../../composables/useToast';

const stats = ref<CacheStats>({
  enabled: true,
  totalHits: 0,
  totalMisses: 0,
  staleHits: 0,
  negativeHits: 0,
  hitRatio: 0,
  entryCount: 0,
  maxEntries: 4096,
  evictionCount: 0,
});

const config = ref<CacheConfig>({
  enabled: true,
  maxEntries: 4096,
  mode: 'limit_max_ttl',
  maxTtlSeconds: 3600,
  fixedTtlSeconds: 3600,
  minTtlEnabled: true,
  minTtlSeconds: 60,
  staleFallbackEnabled: true,
  staleFallbackSeconds: 300,
  negativeTtlEnabled: true,
  negativeTtlSeconds: 30,
  optimistic: true,
});

const entries = ref<CacheEntryItem[]>([]);
const topDomains = ref<CacheDomainStat[]>([]);
const totalEntriesCount = ref(0);
const currentSearch = ref('');

const loading = ref(false);
const saving = ref(false);
const clearing = ref(false);
const isClearDialogOpen = ref(false);
const { toast, show: showToast, dismiss } = useToast();

const hitRatioPercent = computed(() => {
  const ratio = (stats.value.hitRatio || 0) * 100;
  return `${ratio.toFixed(1)}%`;
});

const cacheStats = computed(() => [
  { label: '总命中数', value: stats.value.totalHits.toLocaleString() },
  { label: '实时联网解析', value: stats.value.totalMisses.toLocaleString() },
  { label: '弱网应急保障', value: stats.value.staleHits.toLocaleString() },
  { label: '无效网址拦截', value: stats.value.negativeHits.toLocaleString() },
  { label: '已存记录 / 上限', value: `${stats.value.entryCount} / ${stats.value.maxEntries}` },
]);

async function loadData() {
  loading.value = true;
  try {
    const [st, cfg, entryRes, top] = await Promise.all([
      ipc.getCacheStats(),
      ipc.getCacheConfig(),
      ipc.getCacheEntries(currentSearch.value, 100),
      ipc.getCacheTopDomains(10),
    ]);
    stats.value = st;
    config.value = cfg;
    entries.value = entryRes?.entries || [];
    totalEntriesCount.value = entryRes?.total || 0;
    topDomains.value = top || [];
  } catch (err: any) {
    showToast(err.message || '加载缓存数据失败', true);
  } finally {
    loading.value = false;
  }
}

async function pollMetrics() {
  try {
    const [st, entryRes, top] = await Promise.all([
      ipc.getCacheStats(),
      ipc.getCacheEntries(currentSearch.value, 100),
      ipc.getCacheTopDomains(10),
    ]);
    stats.value = st;
    entries.value = entryRes?.entries || [];
    totalEntriesCount.value = entryRes?.total || 0;
    topDomains.value = top || [];
  } catch {
    // 忽略静默遥测异常
  }
}

async function handleSearch(query: string) {
  currentSearch.value = query;
  try {
    const entryRes = await ipc.getCacheEntries(query, 100);
    entries.value = entryRes?.entries || [];
    totalEntriesCount.value = entryRes?.total || 0;
  } catch (err: any) {
    showToast(err.message, true);
  }
}

async function handleToggleCache(e: Event) {
  const enable = extractSwitchValue(e);
  try {
    const updated = { ...config.value, enabled: enable };
    await ipc.updateCacheConfig(updated);
    config.value = updated;
    stats.value.enabled = enable;
    showToast(enable ? '解析加速已开启' : '解析加速已关闭');
  } catch (err: any) {
    showToast(`切换加速状态失败: ${err.message}`, true);
    revertSwitch(e, !enable);
  }
}

async function handleSaveConfig() {
  saving.value = true;
  try {
    await ipc.updateCacheConfig(config.value);
    showToast('加速策略配置已保存并即时生效！');
    await loadData();
  } catch (err: any) {
    showToast(`保存配置失败: ${err.message}`, true);
  } finally {
    saving.value = false;
  }
}

async function handleConfirmClear() {
  clearing.value = true;
  try {
    await ipc.clearCache();
    isClearDialogOpen.value = false;
    showToast('加速缓存已全部清空！');
    await loadData();
  } catch (err: any) {
    showToast(`清空缓存失败: ${err.message}`, true);
  } finally {
    clearing.value = false;
  }
}

onMounted(() => {
  loadData();
});

usePoller(pollMetrics, 4000);
</script>

<template>
  <div class="space-y-5 pb-12 select-none">
    <!-- 状态反馈提示 -->
    <ToastBanner
      v-if="toast"
      :message="toast.message"
      :type="toast.isError ? 'error' : 'success'"
      compact
      dismissible
      @dismiss="dismiss"
    />

    <!-- 顶层两列响应式卡片网格 (宽屏两列，窄屏单列) -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 items-stretch">
      <!-- 1. 智能加速引擎与命中率总控卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="cache" :size="18" class="text-text-main" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <h3 class="section-title text-[14px]">智能解析加速引擎</h3>
                  <StatusBadge
                    :status="stats.enabled ? 'active' : 'inactive'"
                    :text="stats.enabled ? '加速生效中' : '加速已停用'"
                    size="sm"
                  />
                </div>
                <span class="text-[11px] text-text-muted">分片缓存与高频域名优先响应</span>
              </div>
            </div>

            <!-- 加速总开关 -->
            <div class="flex items-center gap-2 shrink-0">
              <span class="text-[11px] font-semibold text-text-sub hidden sm:inline">
                {{ stats.enabled ? '加速开启' : '已关闭' }}
              </span>
              <md-switch
                :selected="stats.enabled"
                @change="handleToggleCache"
              />
            </div>
          </div>

          <!-- 命中率视觉焦点区块 -->
          <div class="mt-4 p-4 rounded-md bg-surface-card-sub border border-surface-border-sub">
            <div class="flex items-baseline justify-between mb-2">
              <span class="text-[12px] font-semibold text-text-muted">缓存命中率</span>
              <span class="font-mono text-[28px] font-bold leading-none text-text-main tabular-nums">
                {{ hitRatioPercent }}
              </span>
            </div>
            <div class="w-full bg-surface-card rounded-full h-2 overflow-hidden border border-surface-border-sub">
              <div
                class="bg-accent-seal h-full rounded-full transition-all duration-300"
                :style="{ width: `${Math.min((stats.hitRatio || 0) * 100, 100)}%` }"
              />
            </div>
          </div>

          <p class="text-[12px] text-text-sub leading-relaxed mt-3">
            自动缓存已访问过的网址解析结果，下次访问直接瞬间返回；在断网或网络卡顿时自动启用应急保障。
          </p>
        </div>

        <!-- 底部操作按钮 -->
        <div class="flex items-center justify-between pt-2 border-t border-surface-border-sub">
          <button
            type="button"
            @click="loadData"
            :disabled="loading"
            class="app-btn-secondary app-btn-compact"
          >
            <M3Icon name="refresh" :size="13" :class="loading ? 'animate-spin' : ''" />
            <span>刷新数据</span>
          </button>

          <button
            type="button"
            @click="isClearDialogOpen = true"
            :disabled="loading || stats.entryCount === 0"
            class="app-btn-danger app-btn-compact"
          >
            <M3Icon name="delete" :size="13" />
            <span>一键清空缓存</span>
          </button>
        </div>
      </div>

      <!-- 2. 核心加速遥测指标卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="analytics" :size="18" class="text-text-main" />
              </div>
              <div>
                <h3 class="section-title text-[14px]">缓存遥测指标</h3>
                <span class="text-[11px] text-text-muted">多维度解析效率与命中统揽</span>
              </div>
            </div>

            <span class="text-[11px] font-mono text-text-muted">
              容量: {{ stats.entryCount }} / {{ stats.maxEntries }}
            </span>
          </div>

          <!-- 4 宫格遥测指标 -->
          <div class="grid grid-cols-2 gap-3 mt-3">
            <div class="app-card-sub p-3 space-y-1">
              <span class="text-[11px] text-text-muted block">总命中数</span>
              <span class="font-mono text-[18px] font-bold text-text-main block tabular-nums">
                {{ stats.totalHits.toLocaleString() }}
              </span>
            </div>

            <div class="app-card-sub p-3 space-y-1">
              <span class="text-[11px] text-text-muted block">实时联网解析</span>
              <span class="font-mono text-[18px] font-bold text-text-main block tabular-nums">
                {{ stats.totalMisses.toLocaleString() }}
              </span>
            </div>

            <div class="app-card-sub p-3 space-y-1">
              <span class="text-[11px] text-text-muted block">弱网应急保障</span>
              <span class="font-mono text-[18px] font-bold text-status-warning block tabular-nums">
                {{ stats.staleHits.toLocaleString() }}
              </span>
            </div>

            <div class="app-card-sub p-3 space-y-1">
              <span class="text-[11px] text-text-muted block">无效网址拦截</span>
              <span class="font-mono text-[18px] font-bold text-accent-seal block tabular-nums">
                {{ stats.negativeHits.toLocaleString() }}
              </span>
            </div>
          </div>
        </div>

        <!-- 缓存记录使用容量比例条 -->
        <div class="pt-2 border-t border-surface-border-sub space-y-1.5">
          <div class="flex items-center justify-between text-[11px] text-text-muted">
            <span>存储容量占用</span>
            <span class="font-mono">{{ ((stats.entryCount / Math.max(stats.maxEntries, 1)) * 100).toFixed(1) }}%</span>
          </div>
          <div class="w-full bg-surface-card-sub rounded-full h-1.5 overflow-hidden">
            <div
              class="bg-brand-primary h-full rounded-full transition-all duration-300"
              :style="{ width: `${Math.min((stats.entryCount / Math.max(stats.maxEntries, 1)) * 100, 100)}%` }"
            />
          </div>
        </div>
      </div>

      <!-- 3. 高频访问网址排行 -->
      <CacheTopDomains :topDomains="topDomains" />

      <!-- 4. 加速策略设置 -->
      <CacheConfigCard :config="config" :saving="saving" @save="handleSaveConfig" />
    </div>

    <!-- 5. 缓存条目明细表格 (卡片基元) -->
    <CacheEntriesTable
      :entries="entries"
      :totalCount="totalEntriesCount"
      @search="handleSearch"
    />

    <!-- 一键清空确认弹窗 -->
    <AppModal
      :open="isClearDialogOpen"
      @close="isClearDialogOpen = false"
      type="alert"
    >
      <template #headline>
        <div class="flex items-center gap-2 text-status-error font-bold">
          <M3Icon name="delete" :size="22" />
          <span>确定清空所有加速缓存？</span>
        </div>
      </template>

      <div class="space-y-2 pt-1 text-xs text-text-sub leading-relaxed">
        <p>
          此操作将清空当前全部已缓存的网址记录（共 {{ stats.entryCount }} 条）。清空后后续访问网址将重新通过网络解析。
        </p>
      </div>

      <template #actions>
        <button
          type="button"
          @click="isClearDialogOpen = false"
          class="app-btn-secondary"
        >
          取消
        </button>
        <button
          type="button"
          :disabled="clearing"
          @click="handleConfirmClear"
          class="app-btn-danger"
        >
          确认清空
        </button>
      </template>
    </AppModal>
  </div>
</template>
