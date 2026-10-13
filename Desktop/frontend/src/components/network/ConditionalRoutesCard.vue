<script setup lang="ts">
import { ref, computed } from 'vue';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import AppModal from '../AppModal.vue';
import ConfirmModal from '../ConfirmModal.vue';
import { useConfirmModal } from '../../composables/useConfirmModal';

const props = defineProps<{
  routes: Record<string, string>;
  saving?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:routes', value: Record<string, string>): void;
  (e: 'save', value: Record<string, string>): void;
}>();

const routeList = computed(() => {
  return Object.entries(props.routes || {}).map(([domain, server]) => ({
    domain,
    server,
  }));
});

// 编辑与新建弹窗状态
const isDialogOpen = ref(false);
const editingOriginalDomain = ref<string | null>(null);
const formDomain = ref('');
const formServer = ref('');
const formError = ref('');

const {
  isOpen: isConfirmOpen,
  options: confirmOptions,
  ask: askConfirm,
  handleConfirm: handleConfirmAction,
  handleCancel: handleCancelAction,
} = useConfirmModal();

function openAddDialog() {
  editingOriginalDomain.value = null;
  formDomain.value = '';
  formServer.value = '';
  formError.value = '';
  isDialogOpen.value = true;
}

function openEditDialog(item: { domain: string; server: string }) {
  editingOriginalDomain.value = item.domain;
  formDomain.value = item.domain;
  formServer.value = item.server;
  formError.value = '';
  isDialogOpen.value = true;
}

function saveRouteForm() {
  formError.value = '';
  const d = formDomain.value.trim().toLowerCase();
  const s = formServer.value.trim();

  if (!d) {
    formError.value = '请输入匹配域名或泛域名 (如 *.internal 或 corp.local)';
    return;
  }
  if (!s) {
    formError.value = '请输入目标 DNS 服务器地址 (如 10.0.0.1:53 或 192.168.1.1)';
    return;
  }

  const updated = { ...(props.routes || {}) };
  if (editingOriginalDomain.value && editingOriginalDomain.value !== d) {
    delete updated[editingOriginalDomain.value];
  }
  updated[d] = s;

  emit('update:routes', updated);
  emit('save', updated);
  isDialogOpen.value = false;
}

async function handleDeleteRoute(domain: string) {
  const confirmed = await askConfirm({
    title: '删除条件分流规则',
    message: `确定要删除匹配域名 "${domain}" 的定向分流解析规则吗？`,
    confirmText: '确认删除',
    danger: true,
  });
  if (!confirmed) return;

  const updated = { ...(props.routes || {}) };
  delete updated[domain];
  emit('update:routes', updated);
  emit('save', updated);
}
</script>

<template>
  <AppCard
    title="条件分流路由 (Split DNS)"
    subtitle="针对内部私有网段或办公域名定向指定专有 DNS 服务器，支持泛域名如 *.internal"
    icon="router"
    :iconSize="16"
  >
    <template #actions>
      <button
        type="button"
        @click="openAddDialog"
        :disabled="saving"
        class="app-btn-primary app-btn-compact"
      >
        <M3Icon name="add" :size="14" />
        <span>添加分流规则</span>
      </button>
    </template>

    <div class="mt-3">
      <!-- 规则表格 -->
      <div v-if="routeList.length > 0" class="overflow-x-auto rounded-md border border-surface-border">
        <table class="w-full text-left text-xs border-collapse">
          <thead>
            <tr class="bg-surface-card-sub border-b border-surface-border text-text-muted">
              <th class="py-2.5 px-3 font-semibold">匹配域名 / 泛域名</th>
              <th class="py-2.5 px-3 font-semibold">定向 DNS 服务器</th>
              <th class="py-2.5 px-3 font-semibold text-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-surface-border-sub">
            <tr
              v-for="item in routeList"
              :key="item.domain"
              class="hover:bg-surface-hover/60 transition-colors"
            >
              <td class="py-2 px-3 font-mono font-bold text-text-main">
                {{ item.domain }}
              </td>
              <td class="py-2 px-3 font-mono text-text-sub">
                {{ item.server }}
              </td>
              <td class="py-2 px-3 text-right">
                <div class="flex items-center justify-end gap-1">
                  <button
                    type="button"
                    @click="openEditDialog(item)"
                    :disabled="saving"
                    class="app-btn-secondary app-btn-compact !px-2"
                    title="编辑规则"
                  >
                    <M3Icon name="edit" :size="13" />
                  </button>
                  <button
                    type="button"
                    @click="handleDeleteRoute(item.domain)"
                    :disabled="saving"
                    class="app-btn-secondary app-btn-compact !px-2 !text-status-error hover:!bg-status-error/10"
                    title="删除规则"
                  >
                    <M3Icon name="delete" :size="13" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 空白提示 -->
      <div
        v-else
        class="p-5 rounded-md border border-dashed border-surface-border-sub text-center text-text-muted text-xs flex flex-col items-center justify-center gap-2"
      >
        <M3Icon name="info" :size="24" class="opacity-40" />
        <span>暂未配置条件分流路由。全部域名请求将遵循全局上游服务器调度模式。</span>
      </div>
    </div>

    <!-- 弹窗：新建/编辑分流规则 -->
    <AppModal
      :open="isDialogOpen"
      :title="editingOriginalDomain ? '编辑条件分流规则' : '添加条件分流规则'"
      @close="isDialogOpen = false"
    >
      <div class="space-y-4 text-xs">
        <div>
          <label class="font-semibold text-text-main block mb-1">匹配域名或后缀</label>
          <input
            v-model="formDomain"
            type="text"
            placeholder="如 *.corp.internal 或 mycompany.local"
            class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs focus:outline-none focus:border-accent-seal text-text-main font-mono"
          />
          <span class="text-[10px] text-text-muted mt-1 block">支持精准域名或以 *. 开头的泛域名</span>
        </div>

        <div>
          <label class="font-semibold text-text-main block mb-1">定向上游 DNS 服务器地址</label>
          <input
            v-model="formServer"
            type="text"
            placeholder="如 10.0.0.1:53 或 192.168.1.1"
            class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs focus:outline-none focus:border-accent-seal text-text-main font-mono"
          />
          <span class="text-[10px] text-text-muted mt-1 block">若省略端口，默认采用 53 端口</span>
        </div>

        <div v-if="formError" class="p-2 rounded bg-status-error/10 text-status-error text-[11px] font-medium">
          {{ formError }}
        </div>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-2 w-full">
          <button type="button" @click="isDialogOpen = false" class="app-btn-secondary">
            取消
          </button>
          <button type="button" @click="saveRouteForm" class="app-btn-primary">
            保存规则
          </button>
        </div>
      </template>
    </AppModal>

    <!-- 确认删除弹窗 -->
    <ConfirmModal
      :open="isConfirmOpen"
      :title="confirmOptions.title"
      :message="confirmOptions.message"
      :danger="confirmOptions.danger"
      :confirmText="confirmOptions.confirmText"
      :cancelText="confirmOptions.cancelText"
      :showCancel="confirmOptions.showCancel"
      @confirm="handleConfirmAction"
      @cancel="handleCancelAction"
    />
  </AppCard>
</template>
