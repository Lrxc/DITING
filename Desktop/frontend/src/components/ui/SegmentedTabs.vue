<script setup lang="ts">
/**
 * 分段页签（Segment Tabs）：类别内多段内容的二级导航。
 * 墨色 active 文字 + 一条滑动朱砂指示条；窄屏横向滚动并支持左右方向键切换。
 */
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue';

export interface SegmentDef {
  id: string;
  label: string;
}

const props = defineProps<{
  segments: readonly SegmentDef[] | SegmentDef[];
  modelValue: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const listRef = ref<HTMLElement | null>(null);
const itemRefs = ref<HTMLElement[]>([]);
const indicator = ref({ left: 0, width: 0, ready: false });

function setItemRef(el: Element | null, index: number) {
  if (el) itemRefs.value[index] = el as HTMLElement;
}

/** 依据当前激活项的位置同步滑动指示条 */
function syncIndicator() {
  const index = props.segments.findIndex((s) => s.id === props.modelValue);
  const el = index >= 0 ? itemRefs.value[index] : null;
  if (!el) {
    indicator.value = { left: 0, width: 0, ready: false };
    return;
  }
  indicator.value = { left: el.offsetLeft, width: el.offsetWidth, ready: true };
}

watch(
  () => [props.modelValue, props.segments],
  () => nextTick(syncIndicator)
);

let observer: ResizeObserver | null = null;

onMounted(() => {
  nextTick(syncIndicator);
  if (listRef.value && typeof ResizeObserver !== 'undefined') {
    observer = new ResizeObserver(() => syncIndicator());
    observer.observe(listRef.value);
  }
});

onUnmounted(() => {
  if (observer) observer.disconnect();
});

/** 键盘左右键在页签间移动 */
function move(delta: number) {
  const index = props.segments.findIndex((s) => s.id === props.modelValue);
  const next = Math.min(props.segments.length - 1, Math.max(0, (index < 0 ? 0 : index) + delta));
  const seg = props.segments[next];
  if (seg && seg.id !== props.modelValue) emit('update:modelValue', seg.id);
}
</script>

<template>
  <div
    ref="listRef"
    class="scrollbar-none relative flex items-stretch gap-1 overflow-x-auto border-b border-surface-border mb-5"
    role="tablist"
    @keydown.left.prevent="move(-1)"
    @keydown.right.prevent="move(1)"
  >
    <button
      v-for="(seg, index) in segments"
      :key="seg.id"
      :ref="(el) => setItemRef(el as Element | null, index)"
      type="button"
      role="tab"
      :aria-selected="modelValue === seg.id"
      :tabindex="modelValue === seg.id ? 0 : -1"
      class="relative shrink-0 px-3.5 py-2.5 text-[13px] font-semibold whitespace-nowrap cursor-pointer transition-colors duration-150 rounded-t-md"
      :class="
        modelValue === seg.id
          ? 'text-text-main bg-surface-card-sub'
          : 'text-text-muted hover:text-text-sub hover:bg-surface-hover'
      "
      @click="emit('update:modelValue', seg.id)"
    >
      {{ seg.label }}
    </button>

    <!-- 滑动朱砂指示条 -->
    <span
      v-if="indicator.ready"
      aria-hidden="true"
      class="pointer-events-none absolute bottom-0 h-0.5 rounded-full bg-accent-seal transition-all duration-200 ease-out"
      :style="{ left: `${indicator.left}px`, width: `${indicator.width}px` }"
    />
  </div>
</template>
