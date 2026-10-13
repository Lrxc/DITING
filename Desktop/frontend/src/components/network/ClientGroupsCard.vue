<script setup lang="ts">
import { ref, computed } from 'vue';
import { ipc } from '../../api/ipc';
import type { ClientGroup, ClientGroupRuleListOption, ClientGroupConfigResponse } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import StatusBadge from '../StatusBadge.vue';
import ToastBanner from '../ToastBanner.vue';
import ConfirmModal from '../ConfirmModal.vue';
import { extractSwitchValue } from '../../utils/switch';
import { useToast } from '../../composables/useToast';

/** 分组编辑草稿：成员以多行文本编辑，保存时切回数组 */
interface GroupDraft extends Omit<ClientGroup, 'clients'> {
  clientsText: string;
}

/** 组级拦截方式选项（空 = 继承全局） */
const BLOCK_MODES = [
  { value: '', label: '继承全局' },
  { value: 'null_ip', label: '空 IP (0.0.0.0 / ::)' },
  { value: 'nxdomain', label: 'NXDOMAIN (域名不存在)' },
  { value: 'refused', label: 'REFUSED (拒绝应答)' },
] as const;

const enabled = ref(false);
const drafts = ref<GroupDraft[]>([]);
const ruleLists = ref<ClientGroupRuleListOption[]>([]);
const saving = ref(false);
const deleteIndex = ref(-1);
const { toast, show: showToast, dismiss } = useToast();

const enabledGroups = computed(() => drafts.value.filter((g) => g.enabled).length);

function toDraft(g: ClientGroup): GroupDraft {
  const { clients, ...rest } = g;
  return { ...rest, clientsText: (clients || []).join('\n') };
}

function newDraft(index: number): GroupDraft {
  return {
    id: '',
    name: `新分组 ${index + 1}`,
    enabled: true,
    clientsText: '',
    ruleListIds: [],
    blockMode: '',
  };
}

function splitClients(text: string): string[] {
  return text
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function toggleRuleList(row: GroupDraft, id: string) {
  const ids = row.ruleListIds || [];
  const idx = ids.indexOf(id);
  if (idx >= 0) {
    ids.splice(idx, 1);
  } else {
    ids.push(id);
  }
  row.ruleListIds = ids;
}

/** 该组是否继承全部订阅（未勾选任何列表） */
function inheritsAll(row: GroupDraft): boolean {
  return !row.ruleListIds || row.ruleListIds.length === 0;
}

async function fetchData() {
  try {
    const res: ClientGroupConfigResponse = await ipc.getClientGroups();
    if (!res) return;
    enabled.value = res.enabled;
    drafts.value = (res.groups || []).map(toDraft);
    ruleLists.value = res.ruleLists || [];
  } catch {
    // 轮询失败静默，避免打断正在编辑的表单
  }
}

function addGroup() {
  drafts.value.push(newDraft(drafts.value.length));
}

function requestDelete(index: number) {
  deleteIndex.value = index;
}

function confirmDelete() {
  if (deleteIndex.value >= 0) {
    drafts.value.splice(deleteIndex.value, 1);
  }
  deleteIndex.value = -1;
}

async function saveConfig() {
  try {
    saving.value = true;
    // 过滤掉没有任何成员的分组，避免落库无意义配置
    const groups: ClientGroup[] = drafts.value
      .map((d) => {
        const { clientsText, ...rest } = d;
        return { ...rest, clients: splitClients(clientsText) };
      })
      .filter((g) => g.clients.length > 0);

    await ipc.configureClientGroups({ enabled: enabled.value, groups });
    showToast('客户端分组策略已保存并生效');
    await fetchData();
  } catch (err: any) {
    showToast(err?.message || '保存客户端分组失败', true);
  } finally {
    saving.value = false;
  }
}

fetchData();
</script>

<template>
  <div class="space-y-4">
    <ToastBanner v-if="toast" :message="toast.message" :type="toast.isError ? 'error' : 'success'" dismissible @dismiss="dismiss" />

    <AppCard
      title="客户端分组 · 精细策略"
      subtitle="为不同设备组绑定专属订阅规则集与拦截方式，实现精细化管控"
      icon="router"
      :iconSize="16"
    >
      <template #badge>
        <span class="px-2 py-0.5 rounded-full text-[11px] font-mono font-semibold bg-surface-card-sub border border-surface-border text-text-main">
          {{ enabledGroups }} / {{ drafts.length }} 个启用分组
        </span>
        <StatusBadge
          class="ml-1.5"
          :status="enabled ? 'active' : 'inactive'"
          :text="enabled ? '分组生效中' : '分组未启用'"
          size="sm"
        />
      </template>

      <template #actions>
        <button
          type="button"
          @click="saveConfig"
          :disabled="saving"
          class="app-btn-primary app-btn-compact"
        >
          <M3Icon :name="saving ? 'refresh' : 'check'" :size="14" :class="saving ? 'animate-spin' : ''" />
          <span>{{ saving ? '保存中...' : '保存策略' }}</span>
        </button>
      </template>

      <div class="space-y-4 mt-3 text-xs">
        <!-- 总开关 -->
        <div class="app-card-sub p-3 flex items-center justify-between gap-3">
          <div class="min-w-0">
            <span class="text-[12px] font-bold text-text-main block">启用客户端分组</span>
            <p class="text-[11px] text-text-muted">
              关闭时所有客户端统一使用全局规则；开启后按下方网段匹配最近前缀分组。
            </p>
          </div>
          <md-switch
            :selected="enabled"
            @change="enabled = extractSwitchValue($event)"
            class="shrink-0"
          />
        </div>

        <template v-if="enabled">
          <!-- 分组列表 -->
          <div
            v-for="(row, index) in drafts"
            :key="index"
            class="app-card-sub p-3 space-y-3"
          >
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2 min-w-0 flex-1">
                <md-outlined-text-field
                  label="分组名称"
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
                title="删除该分组"
              >
                <M3Icon name="delete" :size="15" />
              </button>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-3">
              <!-- 成员网段 -->
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-text-muted block">成员 IP / 网段</label>
                <textarea
                  v-model="row.clientsText"
                  rows="3"
                  placeholder="例如:&#10;192.168.1.0/24&#10;10.0.0.7&#10;fd00::/8"
                  class="w-full rounded-md border border-surface-border bg-surface-card p-2 font-mono text-xs text-text-main focus:outline-none focus:border-accent-seal"
                ></textarea>
                <span class="text-[10px] text-text-muted block">每行一个 IP 或 CIDR，支持 IPv4/IPv6</span>
              </div>

              <!-- 拦截方式 -->
              <div class="space-y-1">
                <label class="text-[11px] font-semibold text-text-muted block">拦截应答方式</label>
                <select
                  v-model="row.blockMode"
                  class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card text-xs text-text-main focus:outline-none focus:border-accent-seal"
                >
                  <option v-for="m in BLOCK_MODES" :key="m.value" :value="m.value">{{ m.label }}</option>
                </select>
                <span class="text-[10px] text-text-muted block">仅覆盖本组的拦截应答策略</span>
              </div>
            </div>

            <!-- 规则集绑定 -->
            <div class="space-y-1.5">
              <div class="flex items-center justify-between">
                <label class="text-[11px] font-semibold text-text-muted">绑定的订阅规则集</label>
                <button
                  type="button"
                  class="text-[11px] text-brand-primary hover:underline"
                  @click="row.ruleListIds = []"
                >
                  继承全部
                </button>
              </div>
              <div v-if="ruleLists.length > 0" class="flex flex-wrap gap-1.5">
                <button
                  v-for="rl in ruleLists"
                  :key="rl.id"
                  type="button"
                  class="px-2.5 py-1 rounded-md text-[11px] font-medium border transition-colors"
                  :class="(row.ruleListIds || []).includes(rl.id)
                    ? 'bg-brand-primary text-white border-brand-primary'
                    : 'bg-surface-card text-text-muted border-surface-border-sub hover:border-brand-primary/60'"
                  @click="toggleRuleList(row, rl.id)"
                >
                  {{ rl.name || rl.id }}
                </button>
              </div>
              <p v-else class="text-[11px] text-text-muted">当前无可用订阅列表，该组将仅应用自定义规则。</p>
              <p v-if="inheritsAll(row)" class="text-[11px] text-text-muted">
                未勾选任何列表 → 继承全部订阅规则。
              </p>
            </div>
          </div>

          <button
            type="button"
            class="w-full py-2 rounded-lg border border-dashed border-surface-border-sub text-text-muted hover:text-brand-primary hover:border-brand-primary/60 transition-colors flex items-center justify-center gap-1.5"
            @click="addGroup"
          >
            <M3Icon name="add" :size="15" />
            <span>新增客户端分组</span>
          </button>

          <p v-if="drafts.length === 0" class="text-[11px] text-text-muted">
            尚未配置任何分组。未命中分组的客户端将统一使用全局规则（默认组）。
          </p>
        </template>
      </div>

      <template #footer>
        <div class="text-[11px] text-text-muted">
          采用最长前缀匹配：更具体的网段（如 /24）优先于更宽泛的网段（如 /16）。
        </div>
      </template>
    </AppCard>

    <ConfirmModal
      :open="deleteIndex >= 0"
      title="删除客户端分组"
      message="确定要删除该客户端分组吗？此操作将立即从内存移除，保存后不再生效。"
      confirm-text="删除"
      danger
      @confirm="confirmDelete"
      @cancel="deleteIndex = -1"
    />
  </div>
</template>
