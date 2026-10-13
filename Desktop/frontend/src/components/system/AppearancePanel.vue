<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { themeManager, type ThemeMode } from '../../theme/theme';
import AppCard from '../ui/AppCard.vue';

// 主题状态
const currentMode = ref<ThemeMode>(themeManager.getThemeMode());

function handleModeChange(mode: ThemeMode) {
  currentMode.value = mode;
  themeManager.setThemeMode(mode);
}

onMounted(() => {
  currentMode.value = themeManager.getThemeMode();
});
</script>

<template>
  <div class="space-y-5 select-none">
    <AppCard
      title="外观与界面主题"
      subtitle="墨朱视觉体系与浅深色模式切换"
      icon="palette"
    >
      <template #actions>
        <span class="text-[11px] font-semibold text-text-muted">
          当前: {{ currentMode === 'dark' ? '深色墨夜' : currentMode === 'light' ? '浅色宣纸' : '跟随系统' }}
        </span>
      </template>

      <!-- 明暗模式切换按钮组 -->
      <div class="space-y-2 mt-3.5">
        <label class="text-[11px] font-semibold text-text-muted block">界面色彩模式</label>
        <div class="flex flex-wrap gap-2.5">
          <button
            v-for="m in ([{ id: 'system', name: '跟随系统' }, { id: 'light', name: '浅色宣纸' }, { id: 'dark', name: '深色墨夜' }] as const)"
            :key="m.id"
            @click="handleModeChange(m.id)"
            class="app-btn-base flex-1 min-w-[90px]"
            :class="[
              currentMode === m.id
                ? 'bg-brand-primary text-surface-card border border-brand-primary dark:text-[#1a1a1c]'
                : 'bg-surface-card-sub border border-surface-border text-text-sub hover:bg-surface-hover',
            ]"
          >
            {{ m.name }}
          </button>
        </div>
      </div>

      <template #footer>
        <!-- 墨·朱 视觉规范说明 -->
        <div class="app-card-sub p-3 flex items-center gap-3">
          <div class="flex items-center -space-x-1 shrink-0">
            <span class="w-5 h-5 rounded-md bg-[#1A1A1C] border border-surface-border z-10" title="墨 #1A1A1C"></span>
            <span class="w-5 h-5 rounded-md bg-[#8A8B8E] border border-surface-border" title="灰 #8A8B8E"></span>
            <span class="w-5 h-5 rounded-md bg-[#F1EFE9] border border-surface-border" title="宣纸 #F1EFE9"></span>
            <span class="w-5 h-5 rounded-md bg-[#9E2B25] border border-surface-border" title="朱砂 #9E2B25"></span>
          </div>
          <div>
            <div class="text-[12px] font-semibold text-text-main">
              墨 · 朱 视觉规范
            </div>
            <div class="text-[11px] text-text-muted leading-tight mt-0.5">
              以黑灰白墨色为主调，朱砂红用于关键强调与安全拦截警示，深浅双色对等适配。
            </div>
          </div>
        </div>
      </template>
    </AppCard>
  </div>
</template>
