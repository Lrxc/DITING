<script setup lang="ts">
import { computed, onMounted } from 'vue';
import M3Icon from './M3Icon.vue';
import StatusBadge from './StatusBadge.vue';
import ToastBanner from './ToastBanner.vue';
import AppCard from './ui/AppCard.vue';
import { useServiceControl, type ServiceAction } from '../composables/useServiceControl';
import { isWebMode } from '../utils/env';
import { usePlatform } from '../composables/usePlatform';

const { status: serviceStatus, loading, operating, opMessage, opError, fetchStatus, run } =
  useServiceControl();

const webMode = isWebMode();
const { isLinux } = usePlatform();

/** 服务卡片副题：按平台切换术语（Windows 服务 / systemd 服务） */
const serviceSubtitle = computed(() =>
  isLinux.value ? 'systemd 系统服务守护与开机自启' : 'Windows 系统服务守护与进程自启'
);

/** 核心程序文件名：优先取实际可执行路径的文件名，回退按平台给默认名 */
const executableName = computed(() => {
  const p = serviceStatus.value?.executablePath || '';
  const base = p.split(/[\\/]/).filter(Boolean).pop();
  if (base) return base;
  return isLinux.value ? 'diting-service' : 'diting-service.exe';
});

/** 权限提示：Linux 无 UAC 弹窗，强调需要 root 身份 */
const privilegeHint = computed(() => {
  if (serviceStatus.value?.isElevated) {
    return isLinux.value ? '已具备 root 特权' : '已具备特权无弹窗';
  }
  return isLinux.value ? '需以 root 身份运行' : '按需请求 UAC 提权';
});

const privilegeLabel = computed(() =>
  serviceStatus.value?.isElevated ? (isLinux.value ? 'root 特权' : '管理员特权') : '标准用户'
);

/** 注册说明与 Web 模式横幅文案 */
const registerHint = computed(() =>
  isLinux.value
    ? '注册为 systemd 服务后，开机无需登录即可守护运行。'
    : '注册为 Windows 系统服务后，开机免登录即可守护运行。'
);

const webModeHint = computed(() =>
  isLinux.value
    ? '当前为局域网 Web 控制台，服务由宿主机 systemd 守护运行中。请在宿主机以 root 执行 systemctl 命令管理 diting-dnsservice。'
    : '当前为局域网 Web 控制台，后台服务在宿主机守护运行中。Windows 服务注册请在桌面端操作。'
);

onMounted(() => {
  fetchStatus({ silent: Boolean(serviceStatus.value) });
});

function handleAction(action: ServiceAction) {
  run(action);
}

function handleRefresh() {
  fetchStatus();
}
</script>

<template>
  <AppCard
    title="后台核心服务管理"
    :subtitle="serviceSubtitle"
    icon="router"
  >
    <template #actions>
      <button
        type="button"
        @click="handleRefresh"
        :disabled="loading || operating"
        class="app-btn-secondary app-btn-compact"
        title="刷新核心服务状态"
      >
        <M3Icon name="refresh" :size="13" :class="loading ? 'animate-spin' : ''" />
        <span>刷新状态</span>
      </button>
    </template>

    <!-- 状态概览卡片区 -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5 mt-3">
      <!-- 1. 服务运行状态 -->
      <div class="app-card-sub p-3 space-y-1">
        <span class="text-[11px] font-semibold text-text-muted">系统服务状态</span>
        <div class="flex items-center gap-1.5 pt-0.5">
          <StatusBadge
            v-if="serviceStatus?.installed && serviceStatus?.running"
            status="active"
            text="运行中"
            pulse
            size="sm"
          />
          <StatusBadge
            v-else-if="serviceStatus?.installed && !serviceStatus?.running"
            status="warning"
            text="已停止"
            size="sm"
          />
          <StatusBadge
            v-else-if="serviceStatus && !serviceStatus.installed"
            status="error"
            text="未安装"
            size="sm"
          />
          <StatusBadge
            v-else
            status="inactive"
            text="检测中..."
            size="sm"
          />
        </div>
        <p class="text-[10px] text-text-muted leading-tight truncate mt-1">
          {{ serviceStatus?.message || '负责双栈网卡接管与 DNS 转发' }}
        </p>
      </div>

      <!-- 2. 可执行程序路径 -->
      <div class="app-card-sub p-3 space-y-1">
        <span class="text-[11px] font-semibold text-text-muted">核心程序位置</span>
        <div class="font-mono text-[11px] text-text-main truncate pt-0.5" :title="serviceStatus?.executablePath">
          <span v-if="!serviceStatus && loading" class="text-text-muted animate-pulse">扫描中...</span>
          <span v-else>{{ serviceStatus?.executablePath ? executableName : '未检测到' }}</span>
        </div>
        <p class="text-[10px] truncate mt-1" :class="serviceStatus?.canInstall ? 'text-status-success' : 'text-status-warning'">
          {{ serviceStatus?.canInstall ? '✔ 核心组件就绪' : '⚠ 未检测到组件' }}
        </p>
      </div>

      <!-- 3. 管理权限模式 -->
      <div class="app-card-sub p-3 space-y-1">
        <span class="text-[11px] font-semibold text-text-muted">系统权限级别</span>
        <div class="flex items-center gap-1 pt-0.5">
          <M3Icon name="shield" :size="14" :class="serviceStatus?.isElevated ? 'text-status-success' : 'text-text-muted'" />
          <span class="text-[11px] font-bold text-text-main truncate">
            {{ privilegeLabel }}
          </span>
        </div>
        <p class="text-[10px] text-text-muted truncate mt-1">
          {{ privilegeHint }}
        </p>
      </div>
    </div>

    <!-- 反馈通知横幅 -->
    <div v-if="opMessage || opError" class="space-y-2 mt-3">
      <ToastBanner v-if="opMessage" :message="opMessage" type="success" compact />
      <ToastBanner v-if="opError" :message="opError" type="error" compact />
    </div>

    <!-- 底部操作工具栏 -->
    <template #footer>
      <div class="space-y-2 pt-1">
        <div v-if="webMode" class="w-full flex items-center gap-2 text-xs text-text-sub bg-surface-card-sub p-2.5 rounded-md border border-surface-border">
          <M3Icon name="info" :size="15" class="text-text-main shrink-0" />
          <span>{{ webModeHint }}</span>
        </div>
        <template v-else>
          <div class="flex items-center justify-between gap-3 text-[11px] text-text-muted">
            <span>{{ registerHint }}</span>
            <span v-if="!serviceStatus && loading" class="text-text-muted animate-pulse">
              正在检测服务状态...
            </span>
          </div>

          <div class="flex items-center justify-end gap-2 pt-1">
            <!-- 未安装状态：提供安装服务 -->
            <template v-if="serviceStatus && !serviceStatus.installed">
              <button
                type="button"
                @click="handleAction('install')"
                :disabled="operating || !serviceStatus.canInstall"
                class="app-btn-secondary app-btn-compact"
              >
                <M3Icon name="add" :size="13" />
                <span>仅注册服务</span>
              </button>
              <button
                type="button"
                @click="handleAction('install_and_start')"
                :disabled="operating || !serviceStatus.canInstall"
                class="app-btn-primary app-btn-compact"
              >
                <M3Icon name="bolt" :size="13" />
                <span>{{ operating ? '正在处理...' : '一键安装并启动' }}</span>
              </button>
            </template>

            <!-- 已安装但已停止：提供启动与卸载 -->
            <template v-else-if="serviceStatus?.installed && !serviceStatus?.running">
              <button
                type="button"
                @click="handleAction('uninstall')"
                :disabled="operating"
                class="app-btn-danger app-btn-compact"
              >
                <M3Icon name="delete" :size="13" />
                <span>卸载服务</span>
              </button>
              <button
                type="button"
                @click="handleAction('start')"
                :disabled="operating"
                class="app-btn-primary app-btn-compact"
              >
                <M3Icon name="play_arrow" :size="13" />
                <span>{{ operating ? '正在启动...' : '启动服务' }}</span>
              </button>
            </template>

            <!-- 已安装且运行中：提供重启与停止 -->
            <template v-else-if="serviceStatus?.installed && serviceStatus?.running">
              <button
                type="button"
                @click="handleAction('stop')"
                :disabled="operating"
                class="app-btn-secondary app-btn-compact"
              >
                <M3Icon name="pause" :size="13" />
                <span>停止服务</span>
              </button>
              <button
                type="button"
                @click="handleAction('restart')"
                :disabled="operating"
                class="app-btn-primary app-btn-compact"
              >
                <M3Icon name="refresh" :size="13" :class="operating ? 'animate-spin' : ''" />
                <span>{{ operating ? '正在重启...' : '重启服务' }}</span>
              </button>
            </template>
          </div>
        </template>
      </div>
    </template>
  </AppCard>
</template>
