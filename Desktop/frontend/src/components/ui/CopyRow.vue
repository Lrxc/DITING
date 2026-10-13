<script setup lang="ts">
import { useCopyFeedback } from '../../composables/useCopyFeedback';
import M3Icon from '../M3Icon.vue';

const props = withDefaults(
  defineProps<{
    text: string;
    tag?: string;
    tagType?: 'ipv4' | 'ipv6' | 'default';
    copyLabel?: string;
    copiedLabel?: string;
    showDot?: boolean;
    dotColor?: string;
    openUrl?: string;
    openTitle?: string;
  }>(),
  {
    tag: '',
    tagType: 'default',
    copyLabel: '复制',
    copiedLabel: '已复制',
    showDot: false,
    dotColor: 'bg-status-success',
    openUrl: '',
    openTitle: '在默认浏览器中打开',
  }
);

const { copy, isCopied } = useCopyFeedback();

function openInBrowser(url: string) {
  window.open(url, '_blank');
}
</script>

<template>
  <div
    class="flex items-center justify-between gap-2 p-2.5 rounded-md bg-surface-card-sub border border-surface-border-sub hover:border-surface-border transition-colors"
  >
    <div class="flex items-center gap-2 min-w-0">
      <span
        v-if="showDot"
        class="w-2 h-2 rounded-full shrink-0"
        :class="dotColor"
      />
      <span
        v-else-if="tag"
        class="px-1.5 py-0.5 rounded text-[10px] font-bold shrink-0 border"
        :class="[
          tagType === 'ipv6'
            ? 'border-surface-border text-text-muted'
            : tagType === 'ipv4'
            ? 'border-accent-seal/40 text-accent-seal'
            : 'border-surface-border text-text-sub',
        ]"
      >
        {{ tag }}
      </span>
      <slot name="leading" />
      <span
        class="font-mono text-[12px] font-semibold text-text-main truncate select-all"
        :title="text"
      >
        {{ text }}
      </span>
    </div>

    <div class="flex items-center gap-1.5 shrink-0">
      <slot name="actions" />
      <button
        type="button"
        @click="copy(text)"
        class="app-btn-secondary app-btn-compact shrink-0"
        :title="isCopied(text) ? copiedLabel : copyLabel"
      >
        <M3Icon :name="isCopied(text) ? 'check' : 'content_copy'" :size="13" />
        <span>{{ isCopied(text) ? copiedLabel : copyLabel }}</span>
      </button>

      <button
        v-if="openUrl"
        type="button"
        @click="openInBrowser(openUrl)"
        class="app-btn-primary app-btn-compact shrink-0"
        :title="openTitle"
      >
        <M3Icon name="open_in_new" :size="13" />
        <span>打开</span>
      </button>
    </div>
  </div>
</template>
