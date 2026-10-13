<script setup lang="ts">
import type { CacheConfig } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import { extractSwitchValue } from '../../utils/switch';

defineProps<{
  config: CacheConfig;
  saving: boolean;
}>();

const emit = defineEmits<{
  (e: 'save'): void;
}>();
</script>

<template>
  <AppCard
    title="加速策略设置"
    subtitle="有效期规则、弱网容灾与极速响应"
    icon="tune"
    iconBoxClass="w-7 h-7"
    :iconSize="16"
  >
    <template #actions>
      <button
        type="button"
        @click="emit('save')"
        :disabled="saving"
        class="app-btn-primary app-btn-compact"
      >
        <M3Icon :name="saving ? 'refresh' : 'check'" :size="14" :class="saving ? 'animate-spin' : ''" />
        <span>{{ saving ? '保存中...' : '保存策略' }}</span>
      </button>
    </template>

    <div class="space-y-3.5 mt-3 text-xs">
      <!-- TTL 策略模式 -->
      <div class="space-y-1">
        <label class="text-[11px] font-semibold text-text-muted block">记录有效期计算方式</label>
        <md-outlined-select :value="config.mode" @change="config.mode = ($event.target as any).value" class="w-full">
          <md-select-option value="limit_max_ttl"><div slot="headline">限制最长有效期 (推荐，兼顾速度与更新)</div></md-select-option>
          <md-select-option value="follow_dns_ttl"><div slot="headline">完全遵从上游返回的 TTL</div></md-select-option>
          <md-select-option value="fixed_ttl"><div slot="headline">强制使用统一固定有效期</div></md-select-option>
        </md-outlined-select>
      </div>

      <!-- 最长与最短时间 -->
      <div class="grid grid-cols-2 gap-3">
        <md-outlined-text-field
          label="最长保存时长 (秒)"
          type="number"
          :value="String(config.maxTtlSeconds)"
          @input="config.maxTtlSeconds = Number(($event.target as any).value)"
        ></md-outlined-text-field>

        <md-outlined-text-field
          label="最短保留时长 (秒)"
          type="number"
          :value="String(config.minTtlSeconds)"
          @input="config.minTtlSeconds = Number(($event.target as any).value)"
        ></md-outlined-text-field>
      </div>

      <!-- 弱网应急保障 (Stale 容灾) 与 Optimistic 响应 -->
      <div class="app-card-sub p-3 space-y-2.5">
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <span class="text-[12px] font-bold text-text-main block">弱网应急保障 (旧记录兜底)</span>
            <p class="text-[11px] text-text-muted">上游波动超时或断网时，复用近期缓存防止网页报错</p>
          </div>
          <md-switch
            :selected="config.staleFallbackEnabled"
            @change="config.staleFallbackEnabled = extractSwitchValue($event)"
            class="shrink-0"
          />
        </div>

        <div v-if="config.staleFallbackEnabled" class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 pt-1 border-t border-surface-border-sub/60">
          <md-outlined-text-field
            label="应急保留期 (秒)"
            type="number"
            :value="String(config.staleFallbackSeconds)"
            @input="config.staleFallbackSeconds = Number(($event.target as any).value)"
          ></md-outlined-text-field>

          <div class="flex items-center justify-between p-2 rounded bg-surface-card border border-surface-border-sub">
            <div>
              <span class="text-[11px] font-semibold text-text-main block">极速响应</span>
              <span class="text-[10px] text-text-muted">先返缓存，后台静默刷新</span>
            </div>
            <md-switch
              :selected="config.optimistic"
              @change="config.optimistic = extractSwitchValue($event)"
              class="shrink-0"
            />
          </div>
        </div>
      </div>

      <!-- 无效网址记忆加速 -->
      <div class="app-card-sub p-3 flex items-center justify-between gap-2">
        <div class="min-w-0">
          <span class="text-[12px] font-bold text-text-main block">无效网址记忆加速</span>
          <p class="text-[11px] text-text-muted">记住不存在或错误的域名，避免系统和软件频繁重复重试</p>
        </div>
        <md-switch
          :selected="config.negativeTtlEnabled"
          @change="config.negativeTtlEnabled = extractSwitchValue($event)"
          class="shrink-0"
        />
      </div>
    </div>
  </AppCard>
</template>
