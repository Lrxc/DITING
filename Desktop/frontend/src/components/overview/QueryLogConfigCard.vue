<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { ipc } from '../../api/ipc';
import type { QueryLogConfig } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import ToastBanner from '../ToastBanner.vue';
import { extractSwitchValue, revertSwitch } from '../../utils/switch';
import { useToast } from '../../composables/useToast';

const config = ref<QueryLogConfig>({
  enabled: true,
  maxEntries: 10000,
  fileEnabled: true,
  retentionDays: 7,
  logClients: true,
  dataDir: '',
});

const saving = ref(false);
const loading = ref(false);
const { toast, show: showToast, dismiss } = useToast();

async function loadConfig() {
  loading.value = true;
  try {
    const cfg = await ipc.getQueryLogConfig();
    if (cfg) config.value = cfg;
  } catch (err: any) {
    showToast(err.message || '加载查询日志配置失败', true);
  } finally {
    loading.value = false;
  }
}

async function handleSave() {
  saving.value = true;
  try {
    await ipc.updateQueryLogConfig(config.value);
    showToast('查询日志配置已保存并生效');
  } catch (err: any) {
    showToast(`保存失败: ${err.message}`, true);
  } finally {
    saving.value = false;
  }
}

async function handleToggle(e: Event, key: 'enabled' | 'fileEnabled' | 'logClients') {
  const value = extractSwitchValue(e);
  const previous = config.value[key];
  try {
    const updated = { ...config.value, [key]: value };
    await ipc.updateQueryLogConfig(updated);
    config.value = updated;
    const label = key === 'enabled' ? '查询日志记录' : key === 'fileEnabled' ? '历史落盘' : '客户端信息记录';
    showToast(`${label}已${value ? '开启' : '关闭'}`);
  } catch (err: any) {
    showToast(`切换失败: ${err.message}`, true);
    revertSwitch(e, previous);
  }
}

onMounted(() => {
  void loadConfig();
});
</script>

<template>
  <AppCard
    title="查询日志与留存策略"
    subtitle="历史记录容量、落盘留存与隐私开关"
    icon="tune"
    iconBoxClass="w-7 h-7"
    :iconSize="16"
  >
    <template #actions>
      <button type="button" @click="handleSave" :disabled="saving || loading" class="app-btn-primary app-btn-compact">
        <M3Icon :name="saving ? 'refresh' : 'check'" :size="14" :class="saving ? 'animate-spin' : ''" />
        <span>{{ saving ? '保存中...' : '保存策略' }}</span>
      </button>
    </template>

    <ToastBanner
      v-if="toast"
      :message="toast.message"
      :type="toast.isError ? 'error' : 'success'"
      dismissible
      compact
      class="mb-3"
      @dismiss="dismiss"
    />

    <div class="space-y-3.5 mt-3 text-xs">
      <div class="app-card-sub p-3 flex items-center justify-between gap-2">
        <div class="min-w-0">
          <span class="text-[12px] font-bold text-text-main block">启用查询日志记录</span>
          <p class="text-[11px] text-text-muted">关闭后不再记录任何查询历史与统计，DNS 解析不受影响</p>
        </div>
        <md-switch
          :selected="config.enabled"
          @change="handleToggle($event, 'enabled')"
          class="shrink-0"
        />
      </div>

      <template v-if="config.enabled">
        <div class="grid grid-cols-2 gap-3">
          <md-outlined-text-field
            label="内存缓冲条数"
            type="number"
            :value="String(config.maxEntries)"
            @input="config.maxEntries = Number(($event.target as any).value)"
          ></md-outlined-text-field>

          <md-outlined-text-field
            label="落盘留存天数"
            type="number"
            :value="String(config.retentionDays)"
            @input="config.retentionDays = Number(($event.target as any).value)"
          ></md-outlined-text-field>
        </div>

        <div class="app-card-sub p-3 flex items-center justify-between gap-2">
          <div class="min-w-0">
            <span class="text-[12px] font-bold text-text-main block">历史记录落盘</span>
            <p class="text-[11px] text-text-muted">按天写入 JSON-Lines 文件，供历史检索回溯（目录不可写时自动降级为纯内存）</p>
          </div>
          <md-switch
            :selected="config.fileEnabled"
            @change="handleToggle($event, 'fileEnabled')"
            class="shrink-0"
          />
        </div>

        <div class="app-card-sub p-3 flex items-center justify-between gap-2">
          <div class="min-w-0">
            <span class="text-[12px] font-bold text-text-main block">记录客户端信息</span>
            <p class="text-[11px] text-text-muted">记录发起查询的设备 IP 与主机名；关闭后仅保留域名维度统计</p>
          </div>
          <md-switch
            :selected="config.logClients"
            @change="handleToggle($event, 'logClients')"
            class="shrink-0"
          />
        </div>
      </template>

      <div v-if="config.dataDir" class="text-[11px] text-text-muted font-mono break-all px-1">
        落盘目录: {{ config.dataDir }}
      </div>
    </div>
  </AppCard>
</template>
