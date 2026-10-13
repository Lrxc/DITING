<script setup lang="ts">
import { ref, watch } from 'vue';
import type { RewriteRule } from '../../api/types';
import AppModal from '../AppModal.vue';

const props = defineProps<{
  open: boolean;
  editingItem: RewriteRule | null;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'save', rule: RewriteRule): void;
}>();

const domain = ref('');
const ruleType = ref<'ip' | 'cname' | 'txt'>('ip');
const ipsText = ref('');
const cname = ref('');
const txtText = ref('');
const ttl = ref<number | ''>('');
const error = ref('');

watch(
  () => props.open,
  (isOpen) => {
    if (!isOpen) return;
    error.value = '';
    if (props.editingItem) {
      domain.value = props.editingItem.domain;
      ttl.value = props.editingItem.ttl || '';
      if (props.editingItem.cname) {
        ruleType.value = 'cname';
        cname.value = props.editingItem.cname;
        ipsText.value = '';
        txtText.value = '';
      } else if (props.editingItem.txt && props.editingItem.txt.length > 0) {
        ruleType.value = 'txt';
        txtText.value = props.editingItem.txt.join('\n');
        ipsText.value = '';
        cname.value = '';
      } else {
        ruleType.value = 'ip';
        ipsText.value = (props.editingItem.ips || []).join('\n');
        cname.value = '';
        txtText.value = '';
      }
    } else {
      domain.value = '';
      ruleType.value = 'ip';
      ipsText.value = '';
      cname.value = '';
      txtText.value = '';
      ttl.value = '';
    }
  }
);

function handleSave() {
  error.value = '';
  const d = domain.value.trim().toLowerCase();
  if (!d) {
    error.value = '请输入匹配域名（支持精准匹配或 *.lan 泛域名）';
    return;
  }

  const rule: RewriteRule = {
    domain: d,
    ttl: ttl.value ? Number(ttl.value) : undefined,
  };

  if (ruleType.value === 'ip') {
    const list = ipsText.value
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);
    if (list.length === 0) {
      error.value = '请输入至少一个 IPv4 或 IPv6 目标解析地址';
      return;
    }
    rule.ips = list;
  } else if (ruleType.value === 'cname') {
    const c = cname.value.trim().toLowerCase();
    if (!c) {
      error.value = '请输入 CNAME 别名重定向目标域名';
      return;
    }
    rule.cname = c;
  } else if (ruleType.value === 'txt') {
    const list = txtText.value
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);
    if (list.length === 0) {
      error.value = '请输入至少一条 TXT 记录文本';
      return;
    }
    rule.txt = list;
  }

  emit('save', rule);
}
</script>

<template>
  <AppModal
    :open="open"
    :title="editingItem ? '编辑 DNS 重写规则' : '添加 DNS 重写与伪装规则'"
    @close="emit('close')"
  >
    <div class="space-y-4 text-xs">
      <div>
        <label class="font-semibold text-text-main block mb-1">匹配域名</label>
        <input
          v-model="domain"
          type="text"
          placeholder="如 host.lan 或 *.home.local"
          class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono focus:outline-none focus:border-accent-seal"
        />
        <span class="text-[10px] text-text-muted mt-1 block">
          支持精确域名（如 router.lan）或泛域名（如 *.cluster.internal）
        </span>
      </div>

      <div>
        <label class="font-semibold text-text-muted block mb-1.5">重写目标类型</label>
        <div class="grid grid-cols-3 gap-2">
          <label
            class="flex items-center justify-center gap-1.5 py-2 px-2.5 rounded-md border cursor-pointer transition-colors text-[11px] font-semibold"
            :class="ruleType === 'ip' ? 'bg-brand-container/40 border-brand-primary text-brand-primary' : 'border-surface-border bg-surface-card-sub text-text-sub'"
          >
            <input type="radio" v-model="ruleType" value="ip" class="sr-only" />
            <span>A / AAAA 地址</span>
          </label>
          <label
            class="flex items-center justify-center gap-1.5 py-2 px-2.5 rounded-md border cursor-pointer transition-colors text-[11px] font-semibold"
            :class="ruleType === 'cname' ? 'bg-brand-container/40 border-brand-primary text-brand-primary' : 'border-surface-border bg-surface-card-sub text-text-sub'"
          >
            <input type="radio" v-model="ruleType" value="cname" class="sr-only" />
            <span>CNAME 别名</span>
          </label>
          <label
            class="flex items-center justify-center gap-1.5 py-2 px-2.5 rounded-md border cursor-pointer transition-colors text-[11px] font-semibold"
            :class="ruleType === 'txt' ? 'bg-brand-container/40 border-brand-primary text-brand-primary' : 'border-surface-border bg-surface-card-sub text-text-sub'"
          >
            <input type="radio" v-model="ruleType" value="txt" class="sr-only" />
            <span>TXT 文本</span>
          </label>
        </div>
      </div>

      <!-- IP 模式输入 -->
      <div v-if="ruleType === 'ip'">
        <label class="font-semibold text-text-main block mb-1">IPv4 / IPv6 目标解析地址</label>
        <textarea
          v-model="ipsText"
          rows="3"
          placeholder="例如:&#10;192.168.1.100&#10;fe80::1"
          class="w-full rounded-md border border-surface-border bg-surface-card-sub p-2.5 font-mono text-xs text-text-main focus:outline-none focus:border-accent-seal"
        ></textarea>
        <span class="text-[10px] text-text-muted mt-1 block">每行一个 IP 地址，支持多 IP 负载均衡返回</span>
      </div>

      <!-- CNAME 模式输入 -->
      <div v-else-if="ruleType === 'cname'">
        <label class="font-semibold text-text-main block mb-1">CNAME 目标重定向域名</label>
        <input
          v-model="cname"
          type="text"
          placeholder="例如: target.myinternal.net"
          class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono focus:outline-none focus:border-accent-seal"
        />
        <span class="text-[10px] text-text-muted mt-1 block">系统将自动遵循别名链进行递归解析</span>
      </div>

      <!-- TXT 模式输入 -->
      <div v-else-if="ruleType === 'txt'">
        <label class="font-semibold text-text-main block mb-1">TXT 记录内容</label>
        <textarea
          v-model="txtText"
          rows="3"
          placeholder="在此输入 TXT 文本，每行一条..."
          class="w-full rounded-md border border-surface-border bg-surface-card-sub p-2.5 font-mono text-xs text-text-main focus:outline-none focus:border-accent-seal"
        ></textarea>
      </div>

      <div>
        <label class="font-semibold text-text-muted block mb-1">生存时间 TTL (秒，可选)</label>
        <input
          v-model.number="ttl"
          type="number"
          min="1"
          placeholder="留空则使用全局默认 TTL (60s)"
          class="w-full h-[34px] px-3 rounded-md border border-surface-border bg-surface-card-sub text-xs text-text-main font-mono focus:outline-none focus:border-accent-seal"
        />
      </div>

      <div v-if="error" class="p-2 rounded bg-status-error/10 text-status-error text-[11px] font-medium">
        {{ error }}
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-2 w-full">
        <button type="button" @click="emit('close')" class="app-btn-secondary">
          取消
        </button>
        <button type="button" @click="handleSave" class="app-btn-primary">
          保存重写规则
        </button>
      </div>
    </template>
  </AppModal>
</template>
