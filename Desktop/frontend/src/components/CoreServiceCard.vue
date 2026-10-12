<script setup lang="ts">
import { onMounted } from 'vue';
import M3Icon from './M3Icon.vue';
import StatusBadge from './StatusBadge.vue';
import ToastBanner from './ToastBanner.vue';
import { useServiceControl, type ServiceAction } from '../composables/useServiceControl';
import { isWebMode } from '../utils/env';

const { status: serviceStatus, loading, operating, opMessage, opError, fetchStatus, run } =
  useServiceControl();

const webMode = isWebMode();

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
  <div class="app-card p-5 flex flex-col justify-between gap-4 transition-colors">
    <div>
      <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
            <M3Icon name="router" :size="18" class="text-text-main" />
          </div>
          <div>
            <h3 class="section-title text-[14px]">后台核心服务管理</h3>
            <span class="text-[11px] text-text-muted">Windows 系统服务守护与进程自启</span>
          </div>
        </div>

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
      </div>

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
            <span v-else>{{ serviceStatus?.executablePath ? 'diting-service.exe' : '未检测到' }}</span>
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
              {{ serviceStatus?.isElevated ? '管理员特权' : '标准用户' }}
            </span>
          </div>
          <p class="text-[10px] text-text-muted truncate mt-1">
            {{ serviceStatus?.isElevated ? '已具备特权无弹窗' : '按需请求 UAC 提权' }}
          </p>
        </div>
      </div>
    </div>

    <!-- 反馈通知横幅 -->
    <ToastBanner v-if="opMessage" :message="opMessage" type="success" compact />
    <ToastBanner v-if="opError" :message="opError" type="error" compact />

    <!-- 操作工具栏 -->
    <div class="flex flex-wrap items-center justify-between gap-3 pt-1">
      <div v-if="webMode" class="w-full flex items-center gap-2 text-xs text-text-sub bg-surface-card-sub p-3 rounded-md border border-surface-border">
        <M3Icon name="info" :size="16" class="text-text-main shrink-0" />
        <span>当前为局域网 Web 远程管理控制台，后台服务正在宿主机守护运行中。Windows 系统服务注册与提权安装仅限在宿主机桌面端操作。</span>
      </div>
      <template v-else>
        <div class="text-xs text-text-sub flex items-center gap-1.5">
          <M3Icon name="info" :size="15" class="text-text-muted" />
          <span>支持一键向 Windows 注册服务并开机托管，操作将在需要时请求管理员权限。</span>
        </div>

        <div class="flex items-center gap-2">
          <!-- 首次加载中占位 -->
          <span v-if="!serviceStatus && loading" class="text-xs text-text-muted animate-pulse">
            正在检测服务状态...
          </span>

          <!-- 未安装状态：提供安装服务 -->
          <template v-else-if="serviceStatus && !serviceStatus.installed">
            <button
              type="button"
              @click="handleAction('install')"
              :disabled="operating || !serviceStatus.canInstall"
              class="app-btn-secondary app-btn-compact"
            >
              <M3Icon name="add" :size="14" />
              <span>仅注册服务</span>
            </button>
            <button
              type="button"
              @click="handleAction('install_and_start')"
              :disabled="operating || !serviceStatus.canInstall"
              class="app-btn-primary app-btn-compact"
            >
              <M3Icon name="bolt" :size="14" />
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
              <M3Icon name="delete" :size="14" />
              <span>卸载服务</span>
            </button>
            <button
              type="button"
              @click="handleAction('start')"
              :disabled="operating"
              class="app-btn-primary app-btn-compact"
            >
              <M3Icon name="play_arrow" :size="14" />
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
              <M3Icon name="pause" :size="14" />
              <span>停止服务</span>
            </button>
            <button
              type="button"
              @click="handleAction('restart')"
              :disabled="operating"
              class="app-btn-primary app-btn-compact"
            >
              <M3Icon name="refresh" :size="14" :class="operating ? 'animate-spin' : ''" />
              <span>{{ operating ? '正在重启...' : '重启服务' }}</span>
            </button>
          </template>
        </div>
      </template>
    </div>
  </div>
</template>
