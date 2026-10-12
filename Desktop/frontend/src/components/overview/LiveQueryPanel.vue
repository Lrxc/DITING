<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue';
import { ipc } from '../../api/ipc';
import type { WebSocketEvent, QueryEventData } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import { formatClockTimeMs } from '../../utils/format';

interface LogItem extends QueryEventData {
  id: string;
  timestamp: number;
}

const logs = ref<LogItem[]>([]);
const isPaused = ref(false);
const searchFilter = ref('');
const selectedType = ref('ALL');
const selectedStatus = ref('ALL');
const onlyBlocked = ref(false);

const logContainer = ref<HTMLElement | null>(null);
let unsubEvents: (() => void) | null = null;
let idCounter = 0;

const queryTypes = ['ALL', 'A', 'AAAA', 'HTTPS', 'CNAME', 'PTR', 'TXT'] as const;
const statusOptions = ['ALL', 'NOERROR', 'NXDOMAIN', 'SERVFAIL'] as const;

const filteredLogs = computed(() => {
  return logs.value.filter((item) => {
    if (onlyBlocked.value && !item.blocked) {
      return false;
    }
    if (searchFilter.value) {
      const q = searchFilter.value.toLowerCase();
      if (!item.domain.toLowerCase().includes(q) && !(item.clientIP || '').toLowerCase().includes(q)) {
        return false;
      }
    }
    if (selectedType.value !== 'ALL' && item.qtype !== selectedType.value) {
      return false;
    }
    if (selectedStatus.value !== 'ALL') {
      const rcode = item.rcode || (item.success ? 'NOERROR' : 'SERVFAIL');
      if (rcode !== selectedStatus.value) {
        return false;
      }
    }
    return true;
  });
});

const blockedCount = computed(() => logs.value.filter((l) => l.blocked).length);
const successCount = computed(() => logs.value.filter((l) => l.success).length);
const bufferSuccessRate = computed(() => {
  if (logs.value.length === 0) return '100.0%';
  return `${((successCount.value / logs.value.length) * 100).toFixed(1)}%`;
});

function handleNewQuery(data: QueryEventData, timestamp: number) {
  if (isPaused.value) return;

  const item: LogItem = {
    ...data,
    id: `log-${++idCounter}`,
    timestamp: timestamp || Date.now(),
  };

  logs.value.unshift(item);
  if (logs.value.length > 500) {
    logs.value.pop();
  }

  nextTick(() => {
    if (logContainer.value && !isPaused.value) {
      logContainer.value.scrollTop = 0;
    }
  });
}

function clearLogs() {
  logs.value = [];
}

function togglePause() {
  isPaused.value = !isPaused.value;
}

onMounted(() => {
  unsubEvents = ipc.onEvent((event: WebSocketEvent) => {
    if (event.type === 'query') {
      handleNewQuery(event.data, event.timestamp);
    }
  });
});

onUnmounted(() => {
  if (unsubEvents) unsubEvents();
});
</script>

<template>
  <div class="space-y-5 select-none">
    <!-- 宽屏两列网格：左侧为控制与筛选中心，右侧为实时访问记录流 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 items-stretch">
      <!-- 1. 实时查询控制与检索筛选卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <!-- 头部标题与监听状态 -->
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="analytics" :size="18" class="text-text-main" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <h3 class="section-title text-[14px]">实时查询流控制</h3>
                  <span
                    class="px-2 py-0.5 rounded-full text-[11px] font-semibold flex items-center gap-1.5"
                    :class="isPaused ? 'bg-status-warning-bg text-status-warning' : 'bg-status-success-bg text-status-success'"
                  >
                    <span
                      class="w-1.5 h-1.5 rounded-full"
                      :class="isPaused ? 'bg-status-warning' : 'bg-accent-seal seal-pulse'"
                    />
                    {{ isPaused ? '滚屏暂停中' : '实时监听中' }}
                  </span>
                </div>
                <span class="text-[11px] text-text-muted">捕获当前设备及局域网发出的全部 DNS 请求</span>
              </div>
            </div>

            <!-- 控制按钮 -->
            <div class="flex items-center gap-2 shrink-0">
              <button
                type="button"
                @click="togglePause"
                class="app-btn-secondary app-btn-compact"
                :title="isPaused ? '继续自动滚动' : '暂停实时滚动'"
              >
                <M3Icon :name="isPaused ? 'play_arrow' : 'pause'" :size="14" />
                <span>{{ isPaused ? '继续' : '暂停' }}</span>
              </button>

              <button
                type="button"
                @click="clearLogs"
                class="app-btn-secondary app-btn-compact"
                title="清空当前缓冲记录"
              >
                <M3Icon name="delete" :size="14" />
                <span>清空</span>
              </button>
            </div>
          </div>

          <!-- 检索输入框 -->
          <div class="mt-3.5 space-y-3">
            <div>
              <label class="text-[11px] font-semibold text-text-muted block mb-1">检索网址或客户端 IP</label>
              <div class="relative">
                <M3Icon name="search" :size="15" class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                <input
                  v-model="searchFilter"
                  type="text"
                  placeholder="输入搜索网址或客户端 IP 过滤..."
                  class="w-full pl-9 pr-3 h-[34px] rounded-md border border-surface-border bg-surface-card-sub text-xs focus:outline-none focus:border-accent-seal text-text-main placeholder:text-text-muted font-mono"
                />
              </div>
            </div>

            <!-- 查询类型过滤 Chips -->
            <div>
              <label class="text-[11px] font-semibold text-text-muted block mb-1.5">DNS 解析记录类型</label>
              <div class="flex flex-wrap gap-1.5">
                <button
                  v-for="t in queryTypes"
                  :key="t"
                  @click="selectedType = t"
                  class="app-btn-chip font-mono text-[11px]"
                  :class="{ active: selectedType === t }"
                >
                  {{ t === 'ALL' ? '全部类型' : t }}
                </button>
              </div>
            </div>

            <!-- 解析状态过滤 Chips -->
            <div>
              <label class="text-[11px] font-semibold text-text-muted block mb-1.5">解析返回状态</label>
              <div class="flex flex-wrap gap-1.5">
                <button
                  v-for="s in statusOptions"
                  :key="s"
                  @click="selectedStatus = s"
                  class="app-btn-chip text-[11px]"
                  :class="{ active: selectedStatus === s }"
                >
                  {{ s === 'ALL' ? '全部状态' : s === 'NOERROR' ? '成功' : s === 'NXDOMAIN' ? '域名不存在' : '解析失败' }}
                </button>
              </div>
            </div>

            <!-- 仅拦截切换 -->
            <div class="pt-1">
              <button
                type="button"
                @click="onlyBlocked = !onlyBlocked"
                class="app-btn-chip"
                :class="onlyBlocked ? '!bg-accent-seal !text-white !border-accent-seal' : '!text-accent-seal !border-accent-seal/40'"
              >
                <M3Icon name="block" :size="13" />
                <span>{{ onlyBlocked ? '仅看已拦截 (已启用)' : '仅看已拦截请求' }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 缓冲区统计指标磁贴 (3 列等宽) -->
        <div class="grid grid-cols-3 gap-2.5 pt-2 border-t border-surface-border-sub text-xs">
          <div class="app-card-sub p-2.5 space-y-0.5">
            <span class="text-[11px] text-text-muted block">捕获总数</span>
            <span class="font-mono text-[16px] font-bold text-text-main block tabular-nums">
              {{ logs.length }}
            </span>
          </div>

          <div class="app-card-sub p-2.5 space-y-0.5">
            <span class="text-[11px] text-text-muted block">已拦截</span>
            <span class="font-mono text-[16px] font-bold text-accent-seal block tabular-nums">
              {{ blockedCount }}
            </span>
          </div>

          <div class="app-card-sub p-2.5 space-y-0.5">
            <span class="text-[11px] text-text-muted block">成功率</span>
            <span class="font-mono text-[16px] font-bold text-status-success block tabular-nums">
              {{ bufferSuccessRate }}
            </span>
          </div>
        </div>
      </div>

      <!-- 2. 实时访问记录流展示卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-3">
        <div>
          <!-- 头部标题与当前显示条数 -->
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="dns" :size="18" class="text-text-main" />
              </div>
              <div>
                <h3 class="section-title text-[14px]">实时访问数据流</h3>
                <span class="text-[11px] text-text-muted">域名解析明细、耗时与来源 IP</span>
              </div>
            </div>

            <span class="text-[11px] font-mono text-text-muted">
              符合筛选: {{ filteredLogs.length }} 条
            </span>
          </div>

          <!-- 滚动列表 -->
          <div
            ref="logContainer"
            class="mt-3 overflow-y-auto space-y-2 h-[460px] pr-1"
          >
            <div
              v-if="filteredLogs.length === 0"
              class="h-full min-h-[300px] flex flex-col items-center justify-center text-text-muted gap-2 text-center"
            >
              <M3Icon name="logs" :size="32" class="opacity-40" />
              <span class="text-xs">暂无符合条件的访问记录，上网浏览时将在此实时显示…</span>
            </div>

            <div
              v-for="item in filteredLogs"
              :key="item.id"
              class="p-2.5 rounded-md border-l-2 transition-colors flex flex-col gap-1.5"
              :class="[
                item.blocked
                  ? 'border-accent-seal bg-accent-seal-bg/70 dark:bg-accent-seal-bg/30'
                  : 'border-surface-border bg-surface-card-sub hover:border-surface-border hover:bg-surface-hover/60',
              ]"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2 min-w-0">
                  <!-- 拦截标识 -->
                  <span
                    v-if="item.blocked"
                    class="px-1.5 py-0.5 rounded-xs bg-accent-seal text-white font-bold text-[10px] shrink-0"
                    :title="`安全拦截: ${item.filterRule || item.filterReason || '已拦截'}`"
                  >
                    拦截
                  </span>

                  <!-- 状态色标 -->
                  <span
                    class="flex items-center gap-1 shrink-0 text-[11px] font-medium"
                    :class="item.success ? 'text-status-success' : 'text-status-error'"
                  >
                    <span class="w-1.5 h-1.5 rounded-full bg-current" />
                    {{ item.rcode === 'NXDOMAIN' ? '不存在' : !item.success || item.rcode === 'SERVFAIL' ? '失败' : '成功' }}
                  </span>

                  <!-- 查询类型 -->
                  <span class="px-1.5 py-0.5 rounded text-[10px] font-mono bg-surface-card border border-surface-border-sub text-text-muted shrink-0">
                    {{ item.qtype || 'A' }}
                  </span>

                  <!-- 域名 -->
                  <span
                    class="font-mono font-bold text-text-main text-[12px] truncate"
                    :class="item.blocked ? 'line-through decoration-accent-seal/60' : ''"
                    :title="item.domain"
                  >
                    {{ item.domain }}
                  </span>
                </div>

                <!-- 耗时 -->
                <span
                  class="font-mono text-[11px] font-semibold shrink-0"
                  :class="item.durationMs > 100 ? 'text-status-warning' : 'text-text-sub'"
                >
                  {{ item.durationMs.toFixed(1) }} ms
                </span>
              </div>

              <!-- 次行：客户端 IP 与精确时间戳 -->
              <div class="flex items-center justify-between text-[11px] font-mono text-text-muted pt-1 border-t border-surface-border-sub/40">
                <div class="flex items-center gap-1.5">
                  <span
                    v-if="item.clientIP && item.clientIP !== '127.0.0.1' && item.clientIP !== '::1'"
                    class="px-1 py-0.2 rounded text-[9px] border border-accent-seal/40 text-accent-seal"
                  >
                    局域网
                  </span>
                  <span>{{ item.clientIP || '127.0.0.1' }}</span>
                </div>
                <span>{{ formatClockTimeMs(item.timestamp) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="text-[11px] text-text-muted pt-2 border-t border-surface-border-sub flex items-center justify-between">
          <span>滚动缓冲区上限: 500 条</span>
          <span>高频访问将自动入缓存加速</span>
        </div>
      </div>
    </div>
  </div>
</template>
