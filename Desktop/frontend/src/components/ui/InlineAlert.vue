<script setup lang="ts">
import M3Icon from '../M3Icon.vue';

export type InlineAlertType = 'error' | 'warning' | 'info' | 'success';

const props = withDefaults(
  defineProps<{
    message?: string;
    type?: InlineAlertType;
    icon?: string;
    iconClass?: string;
    iconSize?: number;
    shake?: boolean;
  }>(),
  {
    message: '',
    type: 'error',
    icon: '',
    iconClass: '',
    iconSize: 15,
    shake: false,
  }
);

function typeStyles(type: InlineAlertType) {
  switch (type) {
    case 'warning':
      return 'bg-status-warning-bg text-status-warning border-status-warning/20';
    case 'info':
      return 'bg-brand-container text-text-main border-brand-primary/20';
    case 'success':
      return 'bg-status-success-bg text-status-success border-status-success/20';
    case 'error':
    default:
      return 'bg-status-error-bg text-status-error border-status-error/20';
  }
}

function resolvedIcon(): string {
  if (props.icon) return props.icon;
  switch (props.type) {
    case 'warning':
      return 'warning';
    case 'info':
      return 'info';
    case 'success':
      return 'check_circle';
    case 'error':
    default:
      return 'error';
  }
}
</script>

<template>
  <div
    v-if="message || $slots.default"
    class="p-2.5 rounded-md text-xs flex items-center gap-1.5 border"
    :class="[
      typeStyles(type),
      shake ? 'animate-shake' : '',
    ]"
  >
    <M3Icon
      :name="resolvedIcon()"
      :size="iconSize"
      class="shrink-0"
      :class="iconClass"
    />
    <span class="min-w-0 break-words flex-1 leading-relaxed">
      <slot>{{ message }}</slot>
    </span>
    <slot name="actions" />
  </div>
</template>
