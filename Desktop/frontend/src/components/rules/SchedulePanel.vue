<script setup lang="ts">
import { ref, computed } from 'vue';
import { ipc } from '../../api/ipc';
import type { BlockingSchedule, ScheduleConfigResponse } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import StatusBadge from '../StatusBadge.vue';
import ToastBanner from '../ToastBanner.vue';
import ConfirmModal from '../ConfirmModal.vue';
import { extractSwitchValue } from '../../utils/switch';
import { usePoller } from '../../composables/usePoller';
import { useToast } from '../../composables/useToast';

/** 星期选择项（键值与后端 BlockingSchedule.Weekdays 一致） */
const WEEKDAYS = [
  { key: 'mon', label: '一' },
  { key: 'tue', label: '二' },
  { key: 'wed', label: '三' },
  { key: 'thu', label: '四' },
  { key: 'fri', label: '五' },
  { key: 'sat', label: '六' },
  { key: 'sun', label: '日' },
] as const;

const emptySchedule = (): BlockingSchedule => ({
  id: '',
  name: '',
  enabled: true,
  weekdays: [],
  start: '08:00',
  end: '18:00',
});

const config = ref<ScheduleConfigResponse>({
  enabled: false,
  schedules: [],
  active: true,
  activeId: '',
});

const saving = ref(false);
const deleteIndex = ref<number>(-1);
const { toast, show: showToast, dismiss } = useToast();

const enabledCount = computed(() => config.value.schedules.filter((s) => s.enabled).length);

/** 顶部生效状态徽标：区分「常开」「窗口内生效」「已暂停」三态 */
const stateBadge = computed(() => {
  if (!config.value.enabled || enabledCount.value === 0) {
    return { status: 'inactive' as const, text: '未启用 · 拦截常开' };
  }
  if (config.value.active) {
    return { status: 'active' as const, text: '拦截生效中' };
  }
  return { status: 'warning' as const, text: '计划暂停 · 放行全部' };
});

function toggleWeekday(row: BlockingSchedule, key: string) {
  const idx = row.weekdays.indexOf(key);
  if (idx >= 0) {
    row.weekdays.splice(idx, 1);
  } else {
    row.weekdays.push(key);
  }
}

function addSchedule() {
  config.value.schedules.push(emptySchedule());
}

function requestDelete(index: number) {
  deleteIndex.value = index;
}

function confirmDelete() {
  if (deleteIndex.value >= 0) {
    config.value.schedules.splice(deleteIndex.value, 1);
  }
  deleteIndex.value = -1;
}

async function fetchData() {
  try {
    const res = await ipc.getBlockingSchedule();
    if (res) config.value = res;
  } catch {
    // 轮询失败静默，避免打扰用户操作
  }
}

async function saveConfig() {
  try {
    saving.value = true;
    // 过滤掉完全空白的行，避免落库无意义的计划
    const schedules = config.value.schedules
      .filter((s) => s.start && s.end)
      .map((s) => ({ ...s, weekdays: [...s.weekdays] }));
    await ipc.configureBlockingSchedule({
      enabled: config.value.enabled,
      schedules,
    });
    showToast('时间计划已保存并生效');
    await fetchData();
  } catch (err: any) {
    showToast(err?.message || '保存时间计划失败', true);
  } finally {
    saving.value = false;
  }
}

usePoller(fetchData, 5000);
</script>

<template>
  <div class="space-y-4">
    <ToastBanner v-if="toast" :message="toast.message" :type="toast.isError ? 'error' : 'success'" dismissible @dismiss="dismiss" />

    <AppCard
      title="分时拦截 · 时间计划"
      subtitle="仅在指定时段内执行广告拦截，其余时间自动放行"
      icon="alarm"
    >
      <template #actions>
        <StatusBadge :status="stateBadge.status" :text="stateBadge.text" pulse size="sm" />
        <button
          type="button"
          @click="saveConfig"
          :disabled="saving"
          class="app-btn-primary app-btn-compact ml-2"
        >
          <M3Icon :name="saving ? 'refresh' : 'check'" :size="14" :class="saving ? 'animate-spin' : ''" />
          <span>{{ saving ? '保存中...' : '保存计划' }}</span>
        </button>
      </template>

      <div class="space-y-4 mt-3 text-xs">
        <!-- 总开关 -->
        <div class="app-card-sub p-3 flex items-center justify-between gap-3">
          <div class="min-w-0">
            <span class="text-[12px] font-bold text-text-main block">启用分时拦截</span>
            <p class="text-[11px] text-text-muted">
              关闭时拦截始终生效；开启后仅在下列时间窗口内执行拦截。
            </p>
          </div>
          <md-switch
            :selected="config.enabled"
            @change="config.enabled = extractSwitchValue($event)"
            class="shrink-0"
          />
        </div>

        <!-- 计划列表 -->
        <div v-if="config.enabled" class="space-y-3">
          <div
            v-for="(row, index) in config.schedules"
            :key="index"
            class="app-card-sub p-3 space-y-3"
          >
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2 min-w-0 flex-1">
                <md-outlined-text-field
                  label="计划名称"
                  class="flex-1 min-w-0"
                  :value="row.name"
                  @input="row.name = ($event.target as any).value"
                ></md-outlined-text-field>
                <md-switch
                  :selected="row.enabled"
                  @change="row.enabled = extractSwitchValue($event)"
                  class="shrink-0"
                />
              </div>
              <button
                type="button"
                class="app-btn-icon shrink-0"
                @click="requestDelete(index)"
                title="删除该计划"
              >
                <M3Icon name="delete" :size="15" />
              </button>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-text-muted block">生效开始时刻</label>
                <input
                  type="time"
                  class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono focus:outline-none focus:border-accent-seal"
                  :value="row.start"
                  @input="row.start = ($event.target as HTMLInputElement).value"
                />
              </div>
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-text-muted block">
                  生效结束时刻
                  <span class="text-text-muted/70 font-normal">(早于开始即跨夜)</span>
                </label>
                <input
                  type="time"
                  class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono focus:outline-none focus:border-accent-seal"
                  :value="row.end"
                  @input="row.end = ($event.target as HTMLInputElement).value"
                />
              </div>
            </div>

            <div class="space-y-1.5">
              <div class="flex items-center justify-between">
                <label class="text-[11px] font-semibold text-text-muted">生效星期</label>
                <button
                  type="button"
                  class="text-[11px] text-brand-primary hover:underline"
                  @click="row.weekdays = []"
                >
                  每天
                </button>
              </div>
              <div class="flex flex-wrap gap-1.5">
                <button
                  v-for="day in WEEKDAYS"
                  :key="day.key"
                  type="button"
                  class="w-8 h-8 rounded-full text-[11px] font-semibold border transition-colors"
                  :class="row.weekdays.includes(day.key)
                    ? 'bg-brand-primary text-white border-brand-primary'
                    : 'bg-surface-card text-text-muted border-surface-border-sub hover:border-brand-primary/60'"
                  @click="toggleWeekday(row, day.key)"
                >
                  {{ day.label }}
                </button>
              </div>
            </div>
          </div>

          <button
            type="button"
            class="w-full py-2 rounded-lg border border-dashed border-surface-border-sub text-text-muted hover:text-brand-primary hover:border-brand-primary/60 transition-colors flex items-center justify-center gap-1.5"
            @click="addSchedule"
          >
            <M3Icon name="add" :size="15" />
            <span>新增时间计划</span>
          </button>

          <p v-if="enabledCount === 0" class="text-[11px] text-status-warning">
            当前无已启用计划，拦截将保持常开。请至少启用一条计划以限制拦截时段。
          </p>
        </div>
      </div>
    </AppCard>

    <ConfirmModal
      :open="deleteIndex >= 0"
      title="删除时间计划"
      :message="`确定要删除该时间计划吗？此操作将立即从内存移除，保存后不再生效。`"
      confirm-text="删除"
      danger
      @confirm="confirmDelete"
      @cancel="deleteIndex = -1"
    />
  </div>
</template>
