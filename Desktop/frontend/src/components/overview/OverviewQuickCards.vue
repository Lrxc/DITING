<script setup lang="ts">
import type { StatusResponse } from '../../api/types';
import type { NavTab } from '../../constants/navigation';
import M3Icon from '../M3Icon.vue';

defineProps<{
  status: StatusResponse | null;
  formatDnsMode: string;
  allowLAN: boolean;
  primaryLanIP: string;
  takeoverActive: boolean;
}>();

const emit = defineEmits<{
  (e: 'navigate', tab: NavTab): void;
}>();
</script>

<template>
  <div class="contents">
    <!-- 1. 上游 DNS 服务摘要 -->
    <div class="app-card p-5 flex flex-col justify-between gap-4">
      <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-7 h-7 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
            <M3Icon name="upstream" :size="16" class="text-text-main" />
          </div>
          <div class="min-w-0">
            <h3 class="font-semibold text-text-main text-[13.5px] truncate">DNS 调度策略</h3>
            <span class="text-[11px] text-text-muted">上游解析服务与监听</span>
          </div>
        </div>
        <button
          type="button"
          @click="emit('navigate', 'network')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="settings" :size="13" />
          <span>管理配置</span>
        </button>
      </div>

      <div class="grid grid-cols-2 gap-2.5">
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">当前解析策略</div>
          <div class="text-[13px] font-semibold text-text-main truncate">
            {{ formatDnsMode }}
          </div>
        </div>
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">局域网服务范围</div>
          <div
            class="text-[13px] font-semibold truncate"
            :class="allowLAN ? 'text-status-success font-mono' : 'text-text-sub'"
          >
            {{ allowLAN ? (primaryLanIP || '全网监听 0.0.0.0') : '仅本机 (127.0.0.1)' }}
          </div>
        </div>
      </div>
    </div>

    <!-- 2. 智能解析加速摘要 -->
    <div class="app-card p-5 flex flex-col justify-between gap-4">
      <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-7 h-7 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
            <M3Icon name="cache" :size="16" class="text-text-main" />
          </div>
          <div class="min-w-0">
            <h3 class="font-semibold text-text-main text-[13.5px] truncate">智能解析加速</h3>
            <span class="text-[11px] text-text-muted">多层缓存与弱网保障</span>
          </div>
        </div>
        <button
          type="button"
          @click="emit('navigate', 'accel')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="open_in_new" :size="13" />
          <span>查看详情</span>
        </button>
      </div>

      <div class="grid grid-cols-2 gap-2.5">
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">缓存命中率</div>
          <div class="text-[13px] font-bold text-text-main font-mono">
            {{ ((status?.cache?.hitRatio || 0) * 100).toFixed(1) }}%
          </div>
        </div>
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">已存记录 / 存储上限</div>
          <div class="text-[13px] font-mono text-text-main truncate">
            {{ status?.cache?.entryCount || 0 }} / {{ status?.cache?.maxEntries || 4096 }}
          </div>
        </div>
      </div>
    </div>

    <!-- 3. 规则防护与拦截摘要 -->
    <div class="app-card p-5 flex flex-col justify-between gap-4">
      <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-7 h-7 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
            <M3Icon name="shield" :size="16" class="text-text-main" />
          </div>
          <div class="min-w-0">
            <h3 class="font-semibold text-text-main text-[13.5px] truncate">规则过滤防护</h3>
            <span class="text-[11px] text-text-muted">广告与威胁拦截引擎</span>
          </div>
        </div>
        <button
          type="button"
          @click="emit('navigate', 'rules')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="tune" :size="13" />
          <span>管理规则</span>
        </button>
      </div>

      <div class="grid grid-cols-2 gap-2.5">
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">请求拦截率</div>
          <div class="text-[13px] font-bold text-accent-seal font-mono">
            {{ (status?.filter?.blockRate || 0).toFixed(1) }}%
          </div>
        </div>
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">已拦截 / 活跃规则</div>
          <div class="text-[13px] font-mono text-text-main truncate">
            {{ status?.filter?.blockedQueries || 0 }} / {{ status?.filter?.totalRules || 0 }}
          </div>
        </div>
      </div>
    </div>

    <!-- 4. 网络接管与断网保障摘要 -->
    <div class="app-card p-5 flex flex-col justify-between gap-4">
      <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-7 h-7 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
            <M3Icon name="adapters" :size="16" class="text-text-main" />
          </div>
          <div class="min-w-0">
            <h3 class="font-semibold text-text-main text-[13.5px] truncate">网络保护接管</h3>
            <span class="text-[11px] text-text-muted">双栈接管与防断网自愈</span>
          </div>
        </div>
        <button
          type="button"
          @click="emit('navigate', 'network')"
          class="app-btn-secondary app-btn-compact"
        >
          <M3Icon name="open_in_new" :size="13" />
          <span>查看网卡</span>
        </button>
      </div>

      <div class="grid grid-cols-2 gap-2.5">
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">接管运行状态</div>
          <div
            class="text-[13px] font-semibold truncate"
            :class="takeoverActive ? 'text-status-success' : 'text-text-muted'"
          >
            {{ takeoverActive ? '已开启保护' : '未开启 (系统默认)' }}
          </div>
        </div>
        <div class="app-card-sub p-3 space-y-1">
          <div class="text-[11px] text-text-muted">防断网自动自愈</div>
          <div class="text-[12px] font-mono text-status-success truncate" title="异常退出自动恢复系统 DNS">
            已就绪 (状态托管守护)
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
