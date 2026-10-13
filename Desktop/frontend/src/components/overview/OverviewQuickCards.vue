<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import type { StatusResponse } from '../../api/types';
import type { NavTab } from '../../constants/navigation';
import { formatDuration } from '../../utils/format';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import CardTile from '../ui/CardTile.vue';
import StatusBadge from '../StatusBadge.vue';

const props = withDefaults(
  defineProps<{
    status: StatusResponse | null;
    formatDnsMode: string;
    allowLAN: boolean;
    primaryLanIP: string;
    takeoverActive: boolean;
    /** 暂停/恢复请求进行中 */
    pauseBusy?: boolean;
  }>(),
  { pauseBusy: false }
);

const emit = defineEmits<{
  (e: 'navigate', tab: NavTab, sub?: string): void;
  (e: 'pause', paused: boolean): void;
}>();

/** 暂停剩余时间倒计时：以服务端截止时刻为准，本地每秒刷新，避免等待 5 秒轮询 */
const nowMs = ref(Date.now());
let ticker: ReturnType<typeof setInterval> | null = null;

onMounted(() => {
  ticker = setInterval(() => {
    nowMs.value = Date.now();
  }, 1000);
});

onUnmounted(() => {
  if (ticker) clearInterval(ticker);
});

const paused = computed(() => Boolean(props.status?.filter?.paused));

const pauseRemainingText = computed(() => {
  const f = props.status?.filter;
  if (!f?.paused) return '';
  const sec = f.pausedUntilMs
    ? Math.max(0, Math.round((f.pausedUntilMs - nowMs.value) / 1000))
    : f.pausedRemainingSec || 0;
  // 到期后由下一次状态轮询回收暂停态，此处先显示 0 秒避免出现负数
  return formatDuration(sec);
});
</script>

<template>
  <div class="contents">
    <!-- 1. 上游 DNS 服务摘要 -->
    <AppCard
      title="DNS 调度策略"
      subtitle="上游解析服务与监听"
      icon="upstream"
      iconBoxClass="w-7 h-7"
      :iconSize="16"
    >
      <template #actions>
        <button
          type="button"
          @click="emit('navigate', 'resolve', 'upstream')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="settings" :size="13" />
          <span>管理配置</span>
        </button>
      </template>

      <div class="grid grid-cols-2 gap-2.5">
        <CardTile label="当前解析策略">
          <div class="text-[13px] font-semibold text-text-main truncate">
            {{ formatDnsMode }}
          </div>
        </CardTile>

        <CardTile label="局域网服务范围">
          <div
            class="text-[13px] font-semibold truncate"
            :class="allowLAN ? 'text-status-success font-mono' : 'text-text-sub'"
          >
            {{ allowLAN ? (primaryLanIP || '全网监听 0.0.0.0') : '仅本机 (127.0.0.1)' }}
          </div>
        </CardTile>
      </div>
    </AppCard>

    <!-- 2. 智能解析加速摘要 -->
    <AppCard
      title="智能解析加速"
      subtitle="多层缓存与弱网保障"
      icon="cache"
      iconBoxClass="w-7 h-7"
      :iconSize="16"
    >
      <template #actions>
        <button
          type="button"
          @click="emit('navigate', 'resolve', 'cache')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="open_in_new" :size="13" />
          <span>查看详情</span>
        </button>
      </template>

      <div class="grid grid-cols-2 gap-2.5">
        <CardTile
          label="缓存命中率"
          :value="`${((status?.cache?.hitRatio || 0) * 100).toFixed(1)}%`"
          valueClass="text-[13px] font-bold text-text-main font-mono"
        />

        <CardTile label="已存记录 / 存储上限">
          <div class="text-[13px] font-mono text-text-main truncate">
            {{ status?.cache?.entryCount || 0 }} / {{ status?.cache?.maxEntries || 4096 }}
          </div>
        </CardTile>
      </div>
    </AppCard>

    <!-- 3. 规则防护与拦截摘要 -->
    <AppCard
      title="规则过滤防护"
      subtitle="广告与威胁拦截引擎"
      icon="shield"
      iconBoxClass="w-7 h-7"
      :iconSize="16"
    >
      <template #badge>
        <StatusBadge
          v-if="paused"
          status="warning"
          :text="`拦截已暂停 · ${pauseRemainingText}`"
          size="sm"
        />
      </template>

      <template #actions>
        <button
          type="button"
          @click="emit('navigate', 'protect')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="tune" :size="13" />
          <span>管理规则</span>
        </button>
      </template>

      <div class="grid grid-cols-2 gap-2.5">
        <CardTile
          label="请求拦截率"
          :value="`${(status?.filter?.blockRate || 0).toFixed(1)}%`"
          valueClass="text-[13px] font-bold text-accent-seal font-mono"
        />

        <CardTile label="已拦截 / 活跃规则">
          <div class="text-[13px] font-mono text-text-main truncate">
            {{ status?.filter?.blockedQueries || 0 }} / {{ status?.filter?.totalRules || 0 }}
          </div>
        </CardTile>
      </div>

      <!-- 快捷暂停 / 恢复拦截 -->
      <div class="mt-3 pt-3 border-t border-surface-border-sub flex items-center justify-between gap-3">
        <span class="text-[11px] text-text-sub min-w-0 truncate">
          {{ paused ? `剩余 ${pauseRemainingText} 自动恢复` : '临时暂停全部拦截' }}
        </span>
        <button
          v-if="paused"
          type="button"
          class="app-btn-primary app-btn-compact shrink-0"
          :disabled="pauseBusy"
          @click="emit('pause', false)"
        >
          <M3Icon name="play_arrow" :size="13" />
          <span>恢复拦截</span>
        </button>
        <button
          v-else
          type="button"
          class="app-btn-secondary app-btn-compact shrink-0"
          :disabled="pauseBusy"
          @click="emit('pause', true)"
        >
          <M3Icon name="pause" :size="13" />
          <span>暂停 30 分钟</span>
        </button>
      </div>
    </AppCard>

    <!-- 4. 网络接管与断网保障摘要 -->
    <AppCard
      title="网络保护接管"
      subtitle="双栈接管与防断网自愈"
      icon="adapters"
      iconBoxClass="w-7 h-7"
      :iconSize="16"
    >
      <template #actions>
        <button
          type="button"
          @click="emit('navigate', 'devices')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="open_in_new" :size="13" />
          <span>查看网卡</span>
        </button>
      </template>

      <div class="grid grid-cols-2 gap-2.5">
        <CardTile label="接管运行状态">
          <div
            class="text-[13px] font-semibold truncate"
            :class="takeoverActive ? 'text-status-success' : 'text-text-muted'"
          >
            {{ takeoverActive ? '已开启保护' : '未开启 (系统默认)' }}
          </div>
        </CardTile>

        <CardTile label="防断网自动自愈">
          <div class="text-[12px] font-mono text-status-success truncate" title="异常退出自动恢复系统 DNS">
            已就绪 (状态托管守护)
          </div>
        </CardTile>
      </div>
    </AppCard>

    <!-- 5. DNS 重写与伪装摘要 -->
    <AppCard
      title="DNS 重写与伪装"
      subtitle="自定义域名解析映射"
      icon="tune"
      iconBoxClass="w-7 h-7"
      :iconSize="16"
    >
      <template #actions>
        <button
          type="button"
          @click="emit('navigate', 'resolve', 'rewrite')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="open_in_new" :size="13" />
          <span>管理重写</span>
        </button>
      </template>

      <div class="grid grid-cols-2 gap-2.5">
        <CardTile label="重写引擎状态">
          <div
            class="text-[13px] font-semibold truncate"
            :class="status?.rewrite?.enabled ? 'text-status-success' : 'text-text-muted'"
          >
            {{ status?.rewrite?.enabled ? '已开启应答' : '未开启' }}
          </div>
        </CardTile>

        <CardTile label="重写规则总数">
          <div class="text-[13px] font-mono text-text-main truncate font-bold">
            {{ status?.rewrite?.rules?.length || 0 }} 条规则
          </div>
        </CardTile>
      </div>
    </AppCard>

    <!-- 6. 局域网设备与限流管控摘要 -->
    <AppCard
      title="设备管控与安全"
      subtitle="局域网终端与访问控制"
      icon="devices"
      iconBoxClass="w-7 h-7"
      :iconSize="16"
    >
      <template #actions>
        <button
          type="button"
          @click="emit('navigate', 'devices')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="open_in_new" :size="13" />
          <span>设备管理</span>
        </button>
      </template>

      <div class="grid grid-cols-2 gap-2.5">
        <CardTile label="已识别活跃终端">
          <div class="text-[13px] font-mono text-text-main font-bold truncate">
            {{ status?.clients?.length || 0 }} 台设备
          </div>
        </CardTile>

        <CardTile label="请求速率限制">
          <div
            class="text-[13px] font-semibold truncate"
            :class="status?.rateLimit?.enabled ? 'text-status-success' : 'text-text-muted'"
          >
            {{ status?.rateLimit?.enabled ? '防 DoS 开启' : '未开启限流' }}
          </div>
        </CardTile>
      </div>
    </AppCard>
  </div>
</template>
