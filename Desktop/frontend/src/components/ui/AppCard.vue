<script setup lang="ts">
import M3Icon from '../M3Icon.vue';

withDefaults(
  defineProps<{
    title?: string;
    subtitle?: string;
    icon?: string;
    iconSize?: number;
    iconBoxClass?: string;
    borderedHeader?: boolean;
    headerClass?: string;
    footerClass?: string;
    bodyClass?: string;
  }>(),
  {
    title: '',
    subtitle: '',
    icon: '',
    iconSize: 18,
    iconBoxClass: 'w-8 h-8',
    borderedHeader: true,
    headerClass: '',
    footerClass: '',
    bodyClass: '',
  }
);
</script>

<template>
  <div class="app-card p-5 flex flex-col justify-between gap-4 transition-colors">
    <div :class="bodyClass">
      <!-- 头部：图标 + 标题 + 提示 + 操作区域 -->
      <slot name="header">
        <div
          v-if="title || icon || $slots.actions || $slots['header-left']"
          class="flex items-center justify-between gap-3 pb-3"
          :class="[
            borderedHeader ? 'border-b border-surface-border-sub' : '',
            headerClass,
          ]"
        >
          <div class="flex items-center gap-2.5 min-w-0">
            <!-- 头部左侧图标盒子 -->
            <slot name="icon">
              <div
                v-if="icon"
                class="rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0"
                :class="iconBoxClass"
              >
                <M3Icon :name="icon" :size="iconSize" class="text-text-main" />
              </div>
            </slot>

            <!-- 标题与副标题 -->
            <div class="min-w-0">
              <slot name="headline">
                <div class="flex items-center gap-2 flex-wrap">
                  <h3 class="section-title text-[14px] truncate">{{ title }}</h3>
                  <slot name="badge" />
                </div>
                <span v-if="subtitle" class="text-[11px] text-text-muted truncate block">{{ subtitle }}</span>
              </slot>
            </div>
          </div>

          <!-- 头部右侧操作控件 -->
          <div v-if="$slots.actions" class="flex items-center gap-2 shrink-0">
            <slot name="actions" />
          </div>
        </div>
      </slot>

      <!-- 卡片主要内容主体 -->
      <slot />
    </div>

    <!-- 底部操作栏/说明栏 -->
    <div
      v-if="$slots.footer"
      class="pt-2 border-t border-surface-border-sub"
      :class="footerClass"
    >
      <slot name="footer" />
    </div>
  </div>
</template>
