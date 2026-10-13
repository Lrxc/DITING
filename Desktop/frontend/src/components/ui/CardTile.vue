<script setup lang="ts">
withDefaults(
  defineProps<{
    label?: string;
    value?: string | number;
    unit?: string;
    subtext?: string;
    valueClass?: string;
    padded?: boolean;
    spaceY?: boolean;
  }>(),
  {
    label: '',
    value: '',
    unit: '',
    subtext: '',
    valueClass: 'text-text-main font-bold',
    padded: true,
    spaceY: true,
  }
);
</script>

<template>
  <div
    class="app-card-sub"
    :class="[
      padded ? 'p-3' : '',
      spaceY ? 'space-y-1' : '',
    ]"
  >
    <slot name="label">
      <div v-if="label" class="text-[11px] text-text-muted font-medium truncate">
        {{ label }}
      </div>
    </slot>

    <slot>
      <div v-if="value !== ''" class="flex items-baseline gap-1">
        <span
          class="font-mono text-[16px] leading-tight tabular-nums"
          :class="valueClass"
        >
          {{ value }}
        </span>
        <span v-if="unit" class="text-[11px] text-text-muted">{{ unit }}</span>
      </div>
    </slot>

    <slot name="subtext">
      <p v-if="subtext" class="text-[10px] text-text-muted leading-tight truncate mt-0.5">
        {{ subtext }}
      </p>
    </slot>
  </div>
</template>
