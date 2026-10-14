<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { ipc } from '../../api/ipc';
import type { StatusResponse, WebSocketEvent, PortCheckResult } from '../../api/types';
import type { NavTab } from '../../constants/navigation';
import StatusBadge from '../StatusBadge.vue';
import { type StatEntry } from '../ui/MetricStrip.vue';
import MetricChart from '../MetricChart.vue';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import CardTile from '../ui/CardTile.vue';
import PortConflictModal from '../PortConflictModal.vue';
import ToastBanner from '../ToastBanner.vue';
import OverviewQuickCards from './OverviewQuickCards.vue';
import { revertSwitch, extractSwitchValue } from '../../utils/switch';
import { usePoller } from '../../composables/usePoller';
import { usePlatform } from '../../composables/usePlatform';
import { useToast } from '../../composables/useToast';
import { formatUptime } from '../../utils/format';

const emit = defineEmits<{
  (e: 'navigate', tab: NavTab, sub?: string): void;
}>();

const status = ref<StatusResponse | null>(null);
const loading = ref(false);
const togglingDNS = ref(false);
const togglingTakeover = ref(false);
const pauseBusy = ref(false);
const errorMessage = ref('');
const { toast: actionToast, show: showToast, dismiss: dismissToast } = useToast();
const { isLinux } = usePlatform();
/** 端口冲突重试提示：Linux 对应 systemd-resolved stub，Windows 对应 ICS */
const conflictRetryHint = computed(() =>
  isLinux.value
    ? '53 端口仍被占用中，请确认已关闭冲突程序或关闭 systemd-resolved stub 后再重试。'
    : '53 端口仍被占用中，请确认已关闭冲突程序或停止 ICS 服务后再重试。'
);
const portConflict = ref<PortCheckResult | null>(null);
const showConflictModal = ref(false);
const recheckError = ref('');
const isRetryingConflict = ref(false);
const isAutofixing = ref(false);

const qpsHistory = ref<number[]>([]);
const latencyHistory = ref<number[]>([]);

let unsubEvents: (() => void) | null = null;
let unsubConn: (() => void) | null = null;

const dnsRunning = computed(() => Boolean(status.value?.dns?.running));
const allowLAN = computed(() => Boolean(status.value?.dns?.allowLAN));
const primaryLanIP = computed(() => status.value?.dns?.lanAddresses?.[0] || '');
const takeoverActive = computed(() => Boolean(status.value?.takeover?.active));
const activeAdaptersCount = computed(() => status.value?.takeover?.adapters?.length || 0);

const successRate = computed(() => {
  const m = status.value?.metrics;
  if (!m || m.totalQueries === 0) return '100.0%';
  const rate = (m.successQueries / m.totalQueries) * 100;
  return `${rate.toFixed(1)}%`;
});

const uptimeText = computed(() => formatUptime(status.value?.uptimeSeconds || 0));

const metricStats = computed<StatEntry[]>(() => [
  {
    label: '实时查询速率',
    value: (status.value?.metrics?.qps || 0).toFixed(1),
    unit: '次/秒',
  },
  {
    label: '平均响应耗时',
    value: (status.value?.metrics?.avgLatencyMs || 0).toFixed(1),
    unit: '毫秒',
  },
  { label: '解析成功率', value: successRate.value },
  { label: '持续运行时长', value: uptimeText.value },
]);

const formatDnsMode = computed(() => {
  const mode = status.value?.dns?.mode?.toLowerCase();
  switch (mode) {
    case 'single':
      return '单服务器模式';
    case 'primary_backup':
      return '主备自动容灾';
    case 'parallel_race':
    case 'fastest':
      return '并发极速响应';
    case 'smart_prediction':
    case 'load_balance':
      return '智能延迟优选';
    default:
      return mode || '主备自动容灾';
  }
});

async function fetchStatus() {
  try {
    loading.value = true;
    errorMessage.value = '';
    const res = await ipc.getStatus();
    status.value = res;
    if (res.metrics) {
      pushMetricData(res.metrics.qps, res.metrics.avgLatencyMs);
    }
  } catch (err: any) {
    errorMessage.value = err.message || '获取服务状态失败';
  } finally {
    loading.value = false;
  }
}

function pushMetricData(qps: number, latency: number) {
  qpsHistory.value = [...qpsHistory.value.slice(-29), qps];
  latencyHistory.value = [...latencyHistory.value.slice(-29), latency];
}

async function handleStartFailure(err: any) {
  if (err.conflict) {
    portConflict.value = err.conflict;
    showConflictModal.value = true;
    recheckError.value = '';
    errorMessage.value = 'DNS 端口 (53) 被占用，请根据引导排查';
    return;
  }

  const msg = err.message || '';
  if (msg.includes('53') || msg.includes('bind') || msg.includes('Only one usage') || msg.includes('占用')) {
    try {
      const res = await ipc.checkPortConflicts();
      if (!res.available) {
        portConflict.value = res;
        showConflictModal.value = true;
        recheckError.value = '';
        errorMessage.value = 'DNS 端口 (53) 被占用，请根据引导排查';
        return;
      }
    } catch {}
  }

  errorMessage.value = `DNS 操作失败: ${err.message}`;
}

async function handleAutofix() {
  try {
    isAutofixing.value = true;
    recheckError.value = '';
    const res = await ipc.autofixPortConflicts(true);
    if (!res.available) {
      portConflict.value = res;
      recheckError.value = '自动修复执行完毕，但仍有外部进程占用 53 端口，请排查详细诊断。';
      return;
    }
    showConflictModal.value = false;
    portConflict.value = null;
    await fetchStatus();
  } catch (err: any) {
    if (err.conflict) {
      portConflict.value = err.conflict;
    }
    recheckError.value = `自动修复失败: ${err.message}`;
  } finally {
    isAutofixing.value = false;
  }
}

async function handleDowngradeLAN() {
  try {
    isRetryingConflict.value = true;
    recheckError.value = '';
    await ipc.configureLAN({ allowLAN: false });
    await ipc.startDNS();
    showConflictModal.value = false;
    portConflict.value = null;
    await fetchStatus();
  } catch (err: any) {
    if (err.conflict) {
      portConflict.value = err.conflict;
      recheckError.value = `降级启动仍遇到冲突: ${err.message}`;
    } else {
      recheckError.value = `降级操作失败: ${err.message}`;
    }
  } finally {
    isRetryingConflict.value = false;
  }
}

async function handleRetryAfterConflict() {
  try {
    isRetryingConflict.value = true;
    recheckError.value = '';
    const checkRes = await ipc.checkPortConflicts();
    const isSelfOnly =
      checkRes.conflicts &&
      checkRes.conflicts.length > 0 &&
      checkRes.conflicts.every((c) => c.isSelf);

    if (!checkRes.available && !isSelfOnly) {
      portConflict.value = checkRes;
      recheckError.value = conflictRetryHint.value;
      return;
    }

    showConflictModal.value = false;
    portConflict.value = null;
    togglingDNS.value = true;
    await ipc.startDNS();
    await fetchStatus();
  } catch (err: any) {
    if (err.conflict) {
      portConflict.value = err.conflict;
      recheckError.value = '启动仍遇到 53 端口冲突，请排查占用进程。';
      showConflictModal.value = true;
    } else {
      recheckError.value = `启动失败: ${err.message}`;
      showConflictModal.value = true;
    }
  } finally {
    isRetryingConflict.value = false;
    togglingDNS.value = false;
  }
}

async function handleToggleDNS(e: Event) {
  const wantStart = extractSwitchValue(e);
  try {
    togglingDNS.value = true;
    errorMessage.value = '';
    if (wantStart) {
      await ipc.startDNS();
    } else {
      await ipc.stopDNS();
    }
    await fetchStatus();
  } catch (err: any) {
    revertSwitch(e, !wantStart);
    if (wantStart) {
      await handleStartFailure(err);
    } else {
      errorMessage.value = `停止服务失败: ${err.message}`;
    }
  } finally {
    togglingDNS.value = false;
  }
}

async function handleToggleTakeover(e: Event) {
  const nextVal = extractSwitchValue(e);
  togglingTakeover.value = true;
  try {
    if (nextVal) {
      await ipc.enableTakeover();
    } else {
      await ipc.disableTakeover();
    }
    await fetchStatus();
  } catch (err: any) {
    revertSwitch(e, !nextVal);
    if (nextVal) {
      await handleStartFailure(err);
    } else {
      errorMessage.value = `网卡接管操作失败: ${err.message}`;
    }
  } finally {
    togglingTakeover.value = false;
  }
}

/** 总览页快捷暂停/恢复拦截：默认暂停 30 分钟，与规则页保持一致的交互结果。 */
async function handlePause(paused: boolean) {
  try {
    pauseBusy.value = true;
    await ipc.setFilterPause({ paused, durationSec: paused ? 30 * 60 : undefined });
    const ps = await ipc.getFilterPause();
    if (ps) {
      const sec = ps.remainingSeconds || 0;
      showToast(paused ? `拦截已暂停（${formatUptime(sec)}后自动恢复）` : '拦截已恢复');
    }
    await fetchStatus();
  } catch (err: any) {
    showToast(err?.message || '操作拦截暂停状态失败', true);
  } finally {
    pauseBusy.value = false;
  }
}

usePoller(fetchStatus, 5000);

onMounted(() => {
  unsubEvents = ipc.onEvent((event: WebSocketEvent) => {
    if (event.type === 'metrics') {
      const m = event.data;
      if (status.value) {
        status.value.metrics = m;
      }
      pushMetricData(m.qps || 0, m.avgLatencyMs || 0);
    } else if (event.type === 'dns' || event.type === 'takeover' || event.type === 'upstream') {
      fetchStatus();
    } else if (event.type === 'filter') {
      fetchStatus();
    }
  });

  unsubConn = ipc.onConnectionChange((connected) => {
    if (connected) {
      fetchStatus();
    }
  });
});

onUnmounted(() => {
  if (unsubEvents) unsubEvents();
  if (unsubConn) unsubConn();
});
</script>

<template>
  <div class="space-y-5 select-none">
    <!-- 顶部状态提示条 -->
    <ToastBanner v-if="errorMessage" :message="errorMessage" type="error">
      <template #actions>
        <button
          v-if="portConflict"
          type="button"
          @click="showConflictModal = true"
          class="app-btn-secondary app-btn-compact border-accent-seal/40 text-accent-seal"
        >
          <M3Icon name="help" :size="14" />
          <span>排查指引</span>
        </button>
        <button type="button" @click="fetchStatus" class="app-btn-secondary app-btn-compact">
          <M3Icon name="refresh" :size="14" />
          <span>重试</span>
        </button>
      </template>
    </ToastBanner>

    <!-- 快捷操作反馈（暂停/恢复拦截等） -->
    <ToastBanner
      v-if="!errorMessage && actionToast"
      :message="actionToast.message"
      :type="actionToast.isError ? 'error' : 'success'"
      dismissible
      @dismiss="dismissToast"
    />

    <!-- 两列响应式卡片网格 (宽屏两列，窄屏自动切换单列) -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 items-stretch">
      <!-- 1. 运行控制中心卡片 -->
      <AppCard
        title="运行控制中心"
        subtitle="域名解析服务与网络保护总控"
        icon="router"
        iconBoxClass="w-7 h-7"
        :iconSize="16"
      >
        <template #badge>
          <StatusBadge
            :status="dnsRunning ? 'active' : 'inactive'"
            :text="dnsRunning ? '运行中' : '已暂停'"
            :pulse="dnsRunning"
            size="sm"
          />
          <StatusBadge
            v-if="takeoverActive"
            status="active"
            text="保护生效"
            size="sm"
          />
        </template>

        <template #actions>
          <span class="text-[12px] font-semibold text-text-main hidden sm:inline">
            {{ dnsRunning ? '服务运行' : '服务停止' }}
          </span>
          <md-switch
            :selected="dnsRunning"
            :disabled="togglingDNS"
            @change="handleToggleDNS"
          />
        </template>

        <p class="text-[12px] text-text-sub leading-relaxed mt-3">
          谛听在后台稳定运行，提供域名解析加速、广告与威胁拦截，并具备防断网自动恢复保障。
        </p>

        <!-- 内部并排子控制磁贴 -->
        <template #footer>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <!-- DNS 监听服务 -->
            <div class="app-card-sub p-3 flex flex-col justify-between gap-2">
              <div>
                <div class="text-[11px] text-text-muted">解析监听状态</div>
                <div class="text-[13px] font-semibold text-text-main mt-0.5">
                  {{ allowLAN ? '局域网模式' : '本地专用' }}
                </div>
              </div>
              <div class="text-[11px] font-mono text-text-sub truncate" :title="allowLAN ? (primaryLanIP || '0.0.0.0') : '127.0.0.1:53'">
                {{ allowLAN ? (primaryLanIP || '全网监听') : '127.0.0.1:53' }}
              </div>
            </div>

            <!-- 网络接管开关 -->
            <div class="app-card-sub p-3 flex items-center justify-between gap-2">
              <div class="min-w-0">
                <div class="text-[11px] text-text-muted">物理网卡保护</div>
                <div class="text-[13px] font-semibold text-text-main mt-0.5 truncate">
                  {{ takeoverActive ? `已保护 ${activeAdaptersCount} 个网卡` : '未接管网卡' }}
                </div>
              </div>
              <md-switch
                :selected="takeoverActive"
                :disabled="togglingTakeover || !dnsRunning"
                @change="handleToggleTakeover"
                class="shrink-0"
              />
            </div>
          </div>
        </template>
      </AppCard>

      <!-- 2. 核心运行监控指标卡片 -->
      <AppCard
        title="实时遥测指标"
        subtitle="高精度网络解析与服务监控"
        icon="analytics"
        iconBoxClass="w-7 h-7"
        :iconSize="16"
      >
        <template #actions>
          <span class="text-[11px] text-text-muted font-mono hidden sm:inline">持续运行: {{ uptimeText }}</span>
          <button
            type="button"
            @click="fetchStatus"
            :disabled="loading"
            class="app-btn-secondary app-btn-compact"
            title="刷新实时状态"
          >
            <M3Icon name="refresh" :size="13" :class="loading ? 'animate-spin' : ''" />
            <span>刷新</span>
          </button>
        </template>

        <!-- 4 宫格核心指标 -->
        <div class="grid grid-cols-2 gap-3 mt-3">
          <CardTile
            v-for="st in metricStats"
            :key="st.label"
            :label="st.label"
            :value="st.value"
            :unit="st.unit"
            valueClass="text-[20px] font-bold text-text-main leading-none font-mono"
          />
        </div>
      </AppCard>

      <!-- 3-6. 快捷功能与监控摘要卡片组 (4 张卡片，在两列网格中各占一格) -->
      <OverviewQuickCards
        :status="status"
        :formatDnsMode="formatDnsMode"
        :allowLAN="allowLAN"
        :primaryLanIP="primaryLanIP"
        :takeoverActive="takeoverActive"
        :pauseBusy="pauseBusy"
        @navigate="(tab, sub) => emit('navigate', tab, sub)"
        @pause="handlePause"
      />

      <!-- 7. 查询速率趋势波形图卡片 -->
      <MetricChart
        label="查询速率趋势"
        :data="qpsHistory"
        unit="次/秒"
        icon="timeline"
        :height="120"
      />

      <!-- 8. 响应延迟波动波形图卡片 -->
      <MetricChart
        label="响应延迟波动"
        :data="latencyHistory"
        strokeColor="var(--app-accent-seal)"
        unit="毫秒"
        icon="speed"
        :height="120"
      />
    </div>

    <!-- 53 端口占用冲突引导弹窗 -->
    <PortConflictModal
      :open="showConflictModal"
      :conflictResult="portConflict"
      :recheckError="recheckError"
      :rechecking="isRetryingConflict"
      :autofixing="isAutofixing"
      :allowLANDowngradable="allowLAN"
      @close="showConflictModal = false; recheckError = ''"
      @resolved="handleRetryAfterConflict"
      @autofix="handleAutofix"
      @downgradeLAN="handleDowngradeLAN"
    />
  </div>
</template>
