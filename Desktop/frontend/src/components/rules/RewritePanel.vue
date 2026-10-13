<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { ipc } from '../../api/ipc';
import type { RewriteRule, RewriteConfig } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import StatusBadge from '../StatusBadge.vue';
import ToastBanner from '../ToastBanner.vue';
import ConfirmModal from '../ConfirmModal.vue';
import RewriteEditModal from './RewriteEditModal.vue';
import { extractSwitchValue } from '../../utils/switch';
import { useToast } from '../../composables/useToast';
import { useConfirmModal } from '../../composables/useConfirmModal';

const rewriteConfig = ref<RewriteConfig>({
  enabled: true,
  ttl: 60,
  rules: [],
});
const rules = ref<RewriteRule[]>([]);
const searchFilter = ref('');
const loading = ref(false);
const savingGlobal = ref(false);

const { toast, show: showToast, dismiss } = useToast();
const {
  isOpen: isConfirmOpen,
  options: confirmOptions,
  ask: askConfirm,
  handleConfirm: handleConfirmAction,
  handleCancel: handleCancelAction,
} = useConfirmModal();

// 编辑弹窗
const isModalOpen = ref(false);
const editingRule = ref<RewriteRule | null>(null);

const filteredRules = computed(() => {
  if (!searchFilter.value) return rules.value;
  const q = searchFilter.value.toLowerCase();
  return rules.value.filter((r) => {
    if (r.domain.toLowerCase().includes(q)) return true;
    if (r.cname && r.cname.toLowerCase().includes(q)) return true;
    if (r.ips && r.ips.some((ip) => ip.includes(q))) return true;
    if (r.txt && r.txt.some((t) => t.toLowerCase().includes(q))) return true;
    return false;
  });
});

async function loadData() {
  loading.value = true;
  try {
    const [rulesRes, statusRes] = await Promise.all([
      ipc.getRewriteRules().catch(() => []),
      ipc.getStatus().catch(() => null),
    ]);
    rules.value = rulesRes || [];
    if (statusRes && statusRes.rewrite) {
      rewriteConfig.value = statusRes.rewrite;
    }
  } catch (err: any) {
    showToast(err.message || '加载 DNS 重写规则失败', true);
  } finally {
    loading.value = false;
  }
}

async function handleToggleRewrite(e: Event) {
  const nextVal = extractSwitchValue(e);
  savingGlobal.value = true;
  try {
    await ipc.configureRewrite({ enabled: nextVal });
    rewriteConfig.value.enabled = nextVal;
    showToast(`自定义 DNS 重写已${nextVal ? '开启' : '关闭'}`);
  } catch (err: any) {
    showToast(err.message || '切换 DNS 重写状态失败', true);
  } finally {
    savingGlobal.value = false;
  }
}

async function handleSaveGlobalTTL() {
  savingGlobal.value = true;
  try {
    await ipc.configureRewrite({ ttl: Number(rewriteConfig.value.ttl) || 60 });
    showToast('默认重写 TTL 生存期已保存生效');
  } catch (err: any) {
    showToast(err.message || '保存 TTL 失败', true);
  } finally {
    savingGlobal.value = false;
  }
}

function openAddModal() {
  editingRule.value = null;
  isModalOpen.value = true;
}

function openEditModal(rule: RewriteRule) {
  editingRule.value = rule;
  isModalOpen.value = true;
}

async function handleSaveRule(rule: RewriteRule) {
  try {
    await ipc.addRewriteRule(rule);
    showToast(`重写规则 [${rule.domain}] 已成功保存生效！`);
    isModalOpen.value = false;
    await loadData();
  } catch (err: any) {
    showToast(err.message || '保存重写规则失败', true);
  }
}

async function handleDeleteRule(domain: string) {
  const confirmed = await askConfirm({
    title: '删除 DNS 重写规则',
    message: `确定要删除域名 "${domain}" 的 DNS 重写规则吗？`,
    confirmText: '确认删除',
    danger: true,
  });
  if (!confirmed) return;

  try {
    await ipc.deleteRewriteRule(domain);
    showToast(`重写规则 [${domain}] 已成功删除`);
    await loadData();
  } catch (err: any) {
    showToast(err.message || '删除规则失败', true);
  }
}

onMounted(() => {
  loadData();
});
</script>

<template>
  <div class="space-y-6 select-none max-w-7xl mx-auto pb-12">
    <ToastBanner v-if="toast" :message="toast.message" :type="toast.isError ? 'error' : 'success'" dismissible @dismiss="dismiss" />

    <!-- 1. 全局配置与状态顶栏 -->
    <AppCard
      title="自定义 DNS 重写与域名伪装"
      subtitle="类似 AdGuard Home DNS Rewrite 与 Pi-hole 本地解析，精准将域名指向特定 IP、别名或 TXT"
      icon="tune"
    >
      <template #badge>
        <StatusBadge
          :status="rewriteConfig.enabled ? 'active' : 'inactive'"
          :text="rewriteConfig.enabled ? '重写引擎已就绪' : '重写引擎已停用'"
          size="sm"
        />
      </template>

      <template #actions>
        <span class="text-[11px] font-semibold text-text-sub hidden sm:inline">
          {{ rewriteConfig.enabled ? '重写已启用' : '重写已关闭' }}
        </span>
        <md-switch
          :selected="rewriteConfig.enabled"
          :disabled="savingGlobal || loading"
          @change="handleToggleRewrite"
        />
      </template>

      <div class="mt-3.5 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
        <p class="text-[12px] text-text-sub leading-relaxed max-w-xl">
          支持精确域名与 <code class="font-mono bg-surface-card-sub px-1 py-0.5 rounded text-accent-seal">*.lan</code> 泛域名。匹配规则将在上游解析前直接权威应答，无需等待外网请求。
        </p>

        <div class="flex items-center gap-2 shrink-0">
          <label class="text-[11px] text-text-muted font-semibold">默认生存期 (TTL):</label>
          <input
            v-model.number="rewriteConfig.ttl"
            type="number"
            min="1"
            class="w-20 h-[30px] px-2 rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono text-center focus:outline-none focus:border-accent-seal"
          />
          <button
            type="button"
            @click="handleSaveGlobalTTL"
            :disabled="savingGlobal"
            class="app-btn-secondary app-btn-compact"
          >
            保存
          </button>
        </div>
      </div>
    </AppCard>

    <!-- 2. 重写规则列表卡片 -->
    <AppCard
      title="DNS 重写规则列表"
      subtitle="当前生效的自定义伪装映射"
      icon="dns"
    >
      <template #badge>
        <span class="px-2 py-0.5 rounded-full text-[11px] font-mono font-semibold bg-surface-card-sub border border-surface-border text-text-main">
          {{ rules.length }} 条规则
        </span>
      </template>

      <template #actions>
        <button
          type="button"
          @click="openAddModal"
          class="app-btn-primary app-btn-compact"
        >
          <M3Icon name="add" :size="14" />
          <span>添加重写规则</span>
        </button>
      </template>

      <div class="mt-3 space-y-3">
        <!-- 搜索检索栏 -->
        <div class="relative max-w-md">
          <M3Icon name="search" :size="15" class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
          <input
            v-model="searchFilter"
            type="text"
            placeholder="搜索域名、目标 IP 或别名..."
            class="w-full pl-9 pr-3 h-[32px] rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono placeholder:text-text-muted focus:outline-none focus:border-accent-seal"
          />
        </div>

        <!-- 规则数据表格 -->
        <div v-if="filteredRules.length > 0" class="overflow-x-auto rounded-md border border-surface-border">
          <table class="w-full text-left text-xs border-collapse">
            <thead>
              <tr class="bg-surface-card-sub border-b border-surface-border text-text-muted">
                <th class="py-2.5 px-3 font-semibold">匹配域名</th>
                <th class="py-2.5 px-3 font-semibold">类型</th>
                <th class="py-2.5 px-3 font-semibold">重写目标详情</th>
                <th class="py-2.5 px-3 font-semibold text-center">TTL</th>
                <th class="py-2.5 px-3 font-semibold text-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-surface-border-sub">
              <tr
                v-for="r in filteredRules"
                :key="r.domain"
                class="hover:bg-surface-hover/60 transition-colors"
              >
                <td class="py-2.5 px-3 font-mono font-bold text-text-main">
                  {{ r.domain }}
                </td>
                <td class="py-2.5 px-3">
                  <span
                    v-if="r.cname"
                    class="px-1.5 py-0.5 rounded text-[10px] font-mono font-semibold bg-brand-container text-brand-primary border border-brand-primary/20"
                  >
                    CNAME
                  </span>
                  <span
                    v-else-if="r.txt && r.txt.length > 0"
                    class="px-1.5 py-0.5 rounded text-[10px] font-mono font-semibold bg-surface-card-sub text-text-muted border border-surface-border"
                  >
                    TXT
                  </span>
                  <span
                    v-else
                    class="px-1.5 py-0.5 rounded text-[10px] font-mono font-semibold bg-status-success-bg text-status-success border border-status-success/20"
                  >
                    A / AAAA
                  </span>
                </td>
                <td class="py-2.5 px-3 font-mono text-text-sub">
                  <span v-if="r.cname" class="font-bold text-text-main">
                    {{ r.cname }}
                  </span>
                  <div v-else-if="r.txt && r.txt.length > 0" class="space-y-0.5">
                    <div v-for="(t, ti) in r.txt" :key="ti" class="text-[11px] truncate max-w-sm">
                      "{{ t }}"
                    </div>
                  </div>
                  <div v-else-if="r.ips && r.ips.length > 0" class="flex flex-wrap gap-1">
                    <span
                      v-for="ip in r.ips"
                      :key="ip"
                      class="px-1.5 py-0.2 rounded bg-surface-card-sub border border-surface-border-sub text-[11px]"
                    >
                      {{ ip }}
                    </span>
                  </div>
                  <span v-else class="text-text-muted italic">无目标</span>
                </td>
                <td class="py-2.5 px-3 font-mono text-center text-text-muted text-[11px]">
                  {{ r.ttl || rewriteConfig.ttl || 60 }}s
                </td>
                <td class="py-2.5 px-3 text-right">
                  <div class="flex items-center justify-end gap-1">
                    <button
                      type="button"
                      @click="openEditModal(r)"
                      class="app-btn-secondary app-btn-compact !px-2"
                      title="编辑重写规则"
                    >
                      <M3Icon name="edit" :size="13" />
                    </button>
                    <button
                      type="button"
                      @click="handleDeleteRule(r.domain)"
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

        <!-- 空白状态 -->
        <div
          v-else
          class="p-8 rounded-md border border-dashed border-surface-border-sub text-center text-text-muted text-xs flex flex-col items-center justify-center gap-2"
        >
          <M3Icon name="tune" :size="28" class="opacity-40" />
          <span>{{ searchFilter ? '未找到符合条件的重写规则' : '暂未添加任何 DNS 重写规则，点击右上角即可添加' }}</span>
        </div>
      </div>
    </AppCard>

    <!-- 编辑/添加弹窗 -->
    <RewriteEditModal
      :open="isModalOpen"
      :editingItem="editingRule"
      @close="isModalOpen = false"
      @save="handleSaveRule"
    />

    <!-- 删除确认弹窗 -->
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
  </div>
</template>
