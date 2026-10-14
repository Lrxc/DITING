<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ipc } from '../api/ipc';
import { useServiceControl } from '../composables/useServiceControl';
import { usePlatform } from '../composables/usePlatform';
import { useCopyFeedback } from '../composables/useCopyFeedback';
import { isWebMode } from '../utils/env';
import M3Icon from './M3Icon.vue';
import AppModal from './AppModal.vue';
import InlineAlert from './ui/InlineAlert.vue';

const props = defineProps<{
  open: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'connected'): void;
}>();

const {
  status: coreServiceStatus,
  loading: isStatusLoading,
  operating: serviceOperating,
  opMessage: serviceOpText,
  opError: serviceOpError,
  fetchStatus: loadCoreServiceStatus,
  run,
} = useServiceControl();

const webMode = isWebMode();
const { isLinux } = usePlatform();
const { copy, isCopied } = useCopyFeedback(2000);

// Linux + Web 控制台：无桌面端可用，改为提供宿主机 systemctl 命令指引
const linuxWebMode = computed(() => isLinux.value && webMode);

const installCmd = computed(() =>
  coreServiceStatus.value?.installed
    ? 'sudo systemctl enable --now diting-dnsservice'
    : 'sudo /usr/local/bin/diting-service -install   # 或运行安装包内 install.sh'
);

/** 权限描述：Windows 用管理员/UAC 措辞，Linux 强调 root 身份 */
const isElevatedText = computed(() =>
  coreServiceStatus.value?.isElevated
    ? isLinux.value
      ? 'root 特权模式'
      : '管理员特权模式'
    : isLinux.value
      ? '需以 root 身份运行'
      : '按需请求系统权限'
);

const notInstalledText = computed(() =>
  isLinux.value
    ? '检测到宿主机尚未安装谛听核心 systemd 服务。'
    : '检测到当前 Windows 系统尚未安装谛听核心特权服务组件。'
);

const isRetrying = ref(false);

async function retryConnect(): Promise<boolean> {
  if (isRetrying.value) return false;
  isRetrying.value = true;
  try {
    const healthy = await ipc.checkHealth();
    ipc.reconnectWS();
    if (healthy) {
      emit('connected');
      emit('close');
      return true;
    }
    await loadCoreServiceStatus();
    return false;
  } finally {
    isRetrying.value = false;
  }
}

async function handleInstallService() {
  const ok = await run('install');
  if (ok) {
    await loadCoreServiceStatus();
  }
}

async function handleInstallAndStart() {
  const ok = await run('install_and_start');
  if (ok) {
    const healthy = await retryConnect();
    if (!healthy) {
      await loadCoreServiceStatus();
    }
  }
}

async function handleStartService() {
  const ok = await run('start');
  if (ok) {
    const healthy = await retryConnect();
    if (!healthy) {
      await loadCoreServiceStatus();
    }
  }
}

async function handleRestartService() {
  const ok = await run('restart');
  if (ok) {
    await retryConnect();
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      loadCoreServiceStatus();
    }
  }
);
</script>

<template>
  <AppModal
    :open="open"
    @close="emit('close')"
    type="alert"
  >
    <template #headline>
      <div
        class="flex items-center gap-2 font-bold"
        :class="
          isStatusLoading
            ? 'text-text-main'
            : coreServiceStatus?.installed
            ? 'text-status-warning'
            : 'text-status-error'
        "
      >
        <M3Icon :name="isStatusLoading ? 'sync' : 'warning'" :size="22" :class="isStatusLoading ? 'animate-spin' : ''" />
        <span>
          {{
            isStatusLoading
              ? '正在检测后台核心服务...'
              : !coreServiceStatus || !coreServiceStatus.installed
              ? '未安装后台核心服务'
              : coreServiceStatus.running
              ? '后台核心服务未连接'
              : '后台核心服务未启动'
          }}
        </span>
      </div>
    </template>

    <div class="space-y-3 pt-1">
      <div class="flex items-center justify-between p-3 rounded-md bg-surface-card-sub border border-surface-border">
        <div class="flex items-center gap-2">
          <span class="text-xs font-semibold text-text-sub">当前服务状态:</span>
          <span
            class="px-2 py-0.5 rounded-full text-[11px] font-bold"
            :class="[
              isStatusLoading
                ? 'bg-brand-container text-text-main'
                : coreServiceStatus?.installed && coreServiceStatus?.running
                ? 'bg-status-success-bg text-status-success'
                : coreServiceStatus?.installed
                ? 'bg-status-warning-bg text-status-warning'
                : 'bg-status-error-bg text-status-error'
            ]"
          >
            {{ isStatusLoading ? '检测中...' : (coreServiceStatus?.stateText || '未知状态') }}
          </span>
        </div>
        <span class="text-[11px] text-text-muted">
          {{ isElevatedText }}
        </span>
      </div>

      <div v-if="isStatusLoading" class="p-4 flex flex-col items-center justify-center gap-2 text-xs text-text-sub">
        <M3Icon name="sync" :size="20" class="animate-spin text-text-main" />
        <span>正在与系统服务管理器通讯以确认核心服务状态...</span>
      </div>

      <div v-else-if="!coreServiceStatus || !coreServiceStatus.installed" class="space-y-2 text-xs text-text-sub leading-relaxed">
        <p>{{ notInstalledText }}</p>
        <p v-if="!linuxWebMode">核心服务负责双栈物理网卡 DNS 安全接管与多协议上游转发。点击下方<b>“安装服务”</b>或<b>“安装并启动服务”</b>，客户端将自动向系统注册服务并按需申请管理员权限。</p>
        <p v-else>核心服务负责双栈物理网卡 DNS 安全接管与多协议上游转发。请在宿主机以 root 身份安装并启用服务，下方提供了对应命令。</p>
        <div v-if="coreServiceStatus?.executablePath" class="text-[11px] font-mono text-text-muted truncate" :title="coreServiceStatus.executablePath">
          组件位置: {{ coreServiceStatus.executablePath }}
        </div>
      </div>

      <div v-else-if="!coreServiceStatus?.running" class="space-y-2 text-xs text-text-sub leading-relaxed">
        <p>后台核心服务已向系统注册，但当前处于<b>停止状态</b>。</p>
        <p v-if="!linuxWebMode">点击下方<b>“启动服务”</b>即可直接恢复网络加速与防护，系统将在需要时按需申请系统权限。</p>
        <p v-else>请在上表中列出的宿主机以 root 身份启动服务，下方提供了对应命令。</p>
      </div>

      <div v-else class="space-y-2 text-xs text-text-sub leading-relaxed">
        <p>核心服务进程已在系统中运行，但客户端尚未成功建立通信。可能服务正在启动初始化中，或者本地端口被占用。</p>
        <p v-if="!linuxWebMode">您可以点击“重新连接”尝试握手，或点击“重启服务”刷新服务状态。</p>
        <p v-else>请检查宿主机网络与端口占用情况，确认后点击“重新尝试连接”。</p>
      </div>

      <!-- Linux Web 控制台：无桌面端可用，提供宿主机 systemctl 命令指引 -->
      <div v-if="linuxWebMode" class="space-y-2 p-3 rounded-md bg-surface-card-sub border border-surface-border">
        <div class="text-[11px] font-semibold text-text-main flex items-center gap-1.5">
          <M3Icon name="terminal" :size="14" />
          <span>宿主机命令指引（以 root 执行）</span>
        </div>
        <div class="flex items-center justify-between gap-2 p-2 rounded-md bg-surface-card border border-surface-border font-mono text-[11px]">
          <span class="text-text-main select-all truncate">{{ installCmd }}</span>
          <button
            type="button"
            @click="copy(installCmd)"
            class="app-btn-secondary app-btn-compact shrink-0"
          >
            <M3Icon :name="isCopied(installCmd) ? 'check' : 'content_copy'" :size="12" />
            <span>{{ isCopied(installCmd) ? '已复制' : '复制' }}</span>
          </button>
        </div>
        <p class="text-[11px] text-text-muted leading-relaxed">
          亦可通过以下命令查看服务状态与日志：
          <span class="font-mono text-text-sub">systemctl status diting-dnsservice</span> /
          <span class="font-mono text-text-sub">journalctl -u diting-dnsservice -e</span>
        </p>
      </div>

      <InlineAlert
        v-if="serviceOperating"
        type="info"
        icon="refresh"
        iconClass="animate-spin"
        :message="serviceOpText || '正在处理核心服务操作...'"
      />

      <InlineAlert v-if="serviceOpError" :message="serviceOpError" type="error" />
    </div>

    <template #actions>
      <button
        type="button"
        @click="emit('close')"
        :disabled="serviceOperating"
        class="app-btn-secondary"
      >
        稍后处理
      </button>

      <template v-if="!linuxWebMode && (!coreServiceStatus || !coreServiceStatus.installed)">
        <button
          type="button"
          @click="handleInstallService"
          :disabled="serviceOperating || !coreServiceStatus?.canInstall"
          class="app-btn-secondary"
        >
          <M3Icon name="add" :size="16" />
          <span>仅安装服务</span>
        </button>
        <button
          type="button"
          @click="handleInstallAndStart"
          :disabled="serviceOperating || !coreServiceStatus?.canInstall"
          class="app-btn-primary"
        >
          <M3Icon name="bolt" :size="16" />
          <span>{{ serviceOperating ? '正在处理...' : '一键安装并启动服务' }}</span>
        </button>
      </template>

      <template v-else-if="!linuxWebMode && !coreServiceStatus?.running">
        <button
          type="button"
          @click="handleInstallService"
          :disabled="serviceOperating || !coreServiceStatus?.canInstall"
          class="app-btn-secondary"
        >
          <M3Icon name="build" :size="16" />
          <span>重新安装服务</span>
        </button>
        <button
          type="button"
          @click="handleStartService"
          :disabled="serviceOperating"
          class="app-btn-primary"
        >
          <M3Icon name="play_arrow" :size="16" />
          <span>{{ serviceOperating ? '正在启动...' : '启动服务' }}</span>
        </button>
      </template>

      <template v-else-if="!linuxWebMode">
        <button
          type="button"
          @click="handleRestartService"
          :disabled="serviceOperating"
          class="app-btn-secondary"
        >
          <M3Icon name="refresh" :size="16" :class="serviceOperating ? 'animate-spin' : ''" />
          <span>重启服务</span>
        </button>
        <button
          type="button"
          @click="retryConnect"
          :disabled="isRetrying || serviceOperating"
          class="app-btn-primary"
        >
          <M3Icon name="refresh" :size="16" :class="isRetrying ? 'animate-spin' : ''" />
          <span>{{ isRetrying ? '正在连接...' : '重新尝试连接' }}</span>
        </button>
      </template>

      <!-- Linux Web 控制台：仅保留重新连接，服务管理交由宿主机命令指引 -->
      <button
        v-else
        type="button"
        @click="retryConnect"
        :disabled="isRetrying"
        class="app-btn-primary"
      >
        <M3Icon name="refresh" :size="16" :class="isRetrying ? 'animate-spin' : ''" />
        <span>{{ isRetrying ? '正在连接...' : '重新尝试连接' }}</span>
      </button>
    </template>
  </AppModal>
</template>
