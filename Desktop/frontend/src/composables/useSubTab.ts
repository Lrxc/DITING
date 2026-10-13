import { ref, watch, type Ref } from 'vue';
import type { SegmentDef } from '../components/ui/SegmentedTabs.vue';

/**
 * 统一管理带有子路由/Segment 页签的视图状态与 URL 同步
 */
export function useSubTab(
  segments: readonly SegmentDef[] | SegmentDef[],
  getInitialSub: () => string | undefined,
  defaultId?: string
): Ref<string> {
  const initial = getInitialSub();
  const fallback = defaultId || segments[0]?.id || '';
  const active = ref<string>(
    initial && segments.some((s) => s.id === initial) ? initial : fallback
  );

  watch(getInitialSub, (v) => {
    if (v && segments.some((s) => s.id === v)) {
      active.value = v;
    }
  });

  return active;
}
