<script setup lang="ts">
import M3Icon from './M3Icon.vue';

export type ToastBannerType = 'success' | 'error' | 'warning' | 'info';

withDefaults(
  defineProps<{
    message?: string;
    type?: ToastBannerType;
    dismissible?: boolean;
    compact?: boolean;
  }>(),
  {
    message: '',
    type: 'success',
    dismissible: false,
    compact: false,
  }
);

const emit = defineEmits<{
  (e: 'dismiss'): void;
}>();

function typeStyles(type: ToastBannerType) {
  switch (type) {
    case 'error':
      return 'bg-status-error-bg border-accent-seal text-status-error';
    case 'warning':
      return 'bg-status-warning-bg border-status-warning text-status-warning';
    case 'info':
      return 'bg-brand-container border-brand-primary text-text-main';
    case 'success':
    default:
      return 'bg-status-success-bg border-status-success text-status-success';
  }
}

function typeIcon(type: ToastBannerType) {
  switch (type) {
    case 'error':
      return 'error';
    case 'warning':
      return 'warning';
    case 'info':
      return 'info';
    case 'success':
    default:
      return 'check_circle';
  }
}
</script>

<template>
  <div
    v-if="message || $slots.default"
    class="rounded-md border-l-2 border-y border-r flex items-center justify-between gap-3"
    :class="[
      typeStyles(type),
      compact ? 'p-3 text-xs' : 'px-4 py-3 text-[13px]',
    ]"
  >
    <div class="flex items-start gap-2 min-w-0">
      <M3Icon :name="typeIcon(type)" :size="16" class="shrink-0 mt-0.5" />
      <span class="min-w-0 break-words">
        <slot>{{ message }}</slot>
      </span>
    </div>

    <div class="flex items-center gap-2 shrink-0">
      <slot name="actions" />
      <button
        v-if="dismissible"
        type="button"
        @click="emit('dismiss')"
        class="app-btn-secondary app-btn-compact"
      >
        关闭
      </button>
    </div>
  </div>
</template>
