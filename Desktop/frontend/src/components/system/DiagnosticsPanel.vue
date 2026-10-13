<script setup lang="ts">
import { ref } from 'vue';
import { ipc } from '../../api/ipc';
import type { PortCheckResult } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppCard from '../ui/AppCard.vue';
import AppModal from '../AppModal.vue';
import { isWebMode } from '../../utils/env';
import { useToast } from '../../composables/useToast';

const webMode = isWebMode();
const { show: showToast } = useToast();

// 端口诊断状态
const diagnosingPort = ref(false);
const autofixingPort = ref(false);
const portResult = ref<PortCheckResult | null>(null);

// 应急恢复状态
const restoring = ref(false);
const restoreMessage = ref('');
const showRestoreDialog = ref(false);

async function handleDiagnosePort() {
  diagnosingPort.value = true;
  portResult.value = null;
  try {
    const res = await ipc.checkPortConflicts();
    portResult.value = res;
    if (res.available) {
      showToast('DNS 端口正常可用');
    } else {
      showToast('检测到 DNS 端口被占用或冲突', true);
    }
  } catch (err: any) {
    showToast(`诊断失败: ${err.message}`, true);
  } finally {
    diagnosingPort.value = false;
  }
}

async function handleAutofix() {
  autofixingPort.value = true;
  try {
    const res = await ipc.autofixPortConflicts(false);
    portResult.value = res;
    if (res.available) {
      showToast('已成功修复 DNS 端口冲突');
    } else {
      showToast('自动修复完成，但仍有冲突', true);
    }
  } catch (err: any) {
    showToast(`自动修复失败: ${err.message}`, true);
  } finally {
    autofixingPort.value = false;
  }
}

function handleEmergencyRestore() {
  showRestoreDialog.value = true;
}

async function doEmergencyRestore() {
  showRestoreDialog.value = false;
  restoring.value = true;
  restoreMessage.value = '';
  try {
    const msg = await ipc.runNativeEmergencyRestore();
    restoreMessage.value = msg || '网络设置已成功恢复。';
  } catch (err: any) {
    restoreMessage.value = `恢复失败: ${err.message}`;
  } finally {
    restoring.value = false;
  }
}
</script>

<template>
  <div class="space-y-5 select-none">
    <AppCard
      title="网络诊断与应急修复"
      subtitle="端口占用排查与一键防断网自愈"
      icon="build"
    >
      <div class="space-y-3 mt-3 text-xs">
        <!-- 端口诊断小区块 -->
        <div class="app-card-sub p-3 space-y-2">
          <div class="flex items-center justify-between gap-2">
            <span class="font-bold text-text-main text-[12px]">DNS 端口 (53) 占用检测</span>
            <button
              type="button"
              @click="handleDiagnosePort"
              :disabled="diagnosingPort"
              class="app-btn-secondary app-btn-compact"
            >
              <M3Icon v-if="diagnosingPort" name="refresh" :size="13" class="animate-spin" />
              <M3Icon v-else name="search" :size="13" />
              <span>{{ diagnosingPort ? '检测中...' : '检测端口' }}</span>
            </button>
          </div>
          <p class="text-[11px] text-text-muted">排查 ICS 共享或外部程序是否占用 53 端口导致无法启动</p>

          <div v-if="portResult" class="p-2.5 rounded bg-surface-card border border-surface-border text-[11px] space-y-1.5">
            <div class="font-bold" :class="portResult.available ? 'text-status-success' : 'text-status-warning'">
              {{ portResult.available ? '✔ 端口正常可用' : '⚠ 检测到端口冲突' }}
            </div>
            <p class="text-text-sub whitespace-pre-line leading-tight">{{ portResult.diagnostic }}</p>
            <div v-if="!portResult.available && (portResult.canAutofix || portResult.hasICS)">
              <button
                type="button"
                @click="handleAutofix"
                :disabled="autofixingPort"
                class="app-btn-primary app-btn-compact w-full"
              >
                <M3Icon :name="autofixingPort ? 'refresh' : 'bolt'" :size="12" :class="autofixingPort ? 'animate-spin' : ''" />
                <span>{{ autofixingPort ? '正在修复...' : '一键自动修复 ICS 冲突' }}</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 应急自愈小区块 -->
        <div class="app-card-sub p-3 space-y-2">
          <div class="flex items-center justify-between gap-2">
            <span class="font-bold text-text-main text-[12px]">一键恢复系统网络默认</span>
            <span v-if="webMode" class="text-[10px] text-text-muted bg-surface-card px-1.5 py-0.5 rounded border border-surface-border">宿主机可用</span>
            <button
              v-else
              type="button"
              @click="handleEmergencyRestore"
              :disabled="restoring"
              class="app-btn-primary app-btn-compact"
            >
              <M3Icon name="refresh" :size="13" :class="restoring ? 'animate-spin' : ''" />
              <span>一键恢复</span>
            </button>
          </div>
          <p class="text-[11px] text-text-muted">当软件异常退出导致无法上网时，一键恢复系统默认 DNS 并清理缓存</p>

          <div v-if="restoreMessage" class="p-2 rounded bg-status-success-bg text-status-success border border-status-success/30 text-[11px] font-semibold">
            {{ restoreMessage }}
          </div>
        </div>
      </div>

      <template #footer>
        <div class="text-[11px] text-text-muted">
          自愈操作具备原生离线防护，不会破坏现有硬件驱动。
        </div>
      </template>
    </AppCard>

    <!-- 离线恢复确认对话框 (使用 AppModal 居中架构) -->
    <AppModal
      :open="showRestoreDialog"
      @close="showRestoreDialog = false"
      type="alert"
    >
      <template #headline>
        <div class="flex items-center gap-2 text-status-warning font-bold">
          <M3Icon name="warning" :size="24" />
          <span>确定一键恢复网络设置？</span>
        </div>
      </template>

      <div class="space-y-2 text-xs text-text-sub leading-relaxed">
        <p>此操作将安全恢复所有网络连接的原有 DNS 设置（恢复为自动获取或原有设置），并刷新网络缓存。</p>
        <p>适用于因软件异常退出导致电脑无法打开网页、需要紧急恢复上网的场景。</p>
      </div>

      <template #actions>
        <button
          type="button"
          @click="showRestoreDialog = false"
          class="app-btn-secondary"
        >
          取消
        </button>
        <button
          type="button"
          @click="doEmergencyRestore"
          class="app-btn-warning"
        >
          立即修复网络
        </button>
      </template>
    </AppModal>
  </div>
</template>
