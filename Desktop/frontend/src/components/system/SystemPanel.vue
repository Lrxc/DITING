<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { ipc } from '../../api/ipc';
import { themeManager, type ThemeMode } from '../../theme/theme';
import type { PortCheckResult } from '../../api/types';
import M3Icon from '../M3Icon.vue';
import AppModal from '../AppModal.vue';
import ToastBanner from '../ToastBanner.vue';
import CoreServiceCard from '../CoreServiceCard.vue';
import LanWebCard from '../LanWebCard.vue';
import { revertSwitch, extractSwitchValue } from '../../utils/switch';
import { isWebMode } from '../../utils/env';
import { useToast } from '../../composables/useToast';

const webMode = isWebMode();
const { toast, show: showToast, dismiss } = useToast();
const host = ref('127.0.0.1');
const port = ref('15353');
const token = ref('');
const testingConnection = ref(false);
const connResult = ref<{ success: boolean; message: string } | null>(null);

// 主题状态
const currentMode = ref<ThemeMode>(themeManager.getThemeMode());

// 端口诊断状态
const diagnosingPort = ref(false);
const portResult = ref<PortCheckResult | null>(null);

// 应急恢复状态
const restoring = ref(false);
const restoreMessage = ref('');
const showRestoreDialog = ref(false);

// 开机自启状态
const autostart = ref(false);
const togglingAutostart = ref(false);

async function checkAutostart() {
  try {
    autostart.value = await ipc.isAutoStartEnabled();
  } catch {}
}

async function handleToggleAutostart(e: Event) {
  const enable = extractSwitchValue(e);
  togglingAutostart.value = true;
  try {
    autostart.value = await ipc.setAutoStart(enable);
    showToast(`开机自启已${enable ? '开启' : '关闭'}`);
  } catch (err: any) {
    showToast(`设置开机自启失败: ${err.message}`, true);
    revertSwitch(e, !enable);
  } finally {
    togglingAutostart.value = false;
  }
}

function loadSettings() {
  const cfg = ipc.getConfig();
  host.value = cfg.host;
  port.value = cfg.port;
  token.value = cfg.token;
  currentMode.value = themeManager.getThemeMode();
  checkAutostart();
}

async function testConnection() {
  testingConnection.value = true;
  connResult.value = null;
  ipc.saveConfig(host.value, port.value, token.value);
  try {
    const ok = await ipc.checkHealth();
    if (ok) {
      connResult.value = { success: true, message: '成功连接后台核心服务' };
    } else {
      connResult.value = { success: false, message: '无法连接到后台服务，请确认核心服务已启动' };
    }
  } catch (err: any) {
    connResult.value = { success: false, message: err.message || '连接失败' };
  } finally {
    testingConnection.value = false;
  }
}

function handleSaveIPC() {
  ipc.saveConfig(host.value, port.value, token.value);
  testConnection();
}

function handleModeChange(mode: ThemeMode) {
  currentMode.value = mode;
  themeManager.setThemeMode(mode);
}

const autofixingPort = ref(false);

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

async function handleAutofixInSettings() {
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

onMounted(() => {
  loadSettings();
});
</script>

<template>
  <div class="space-y-5 select-none">
    <!-- 全局消息提示横幅 -->
    <ToastBanner
      v-if="toast"
      :message="toast.message"
      :type="toast.isError ? 'error' : 'success'"
      dismissible
      @dismiss="dismiss"
    />

    <!-- 两列响应式卡片网格 (宽屏两列，窄屏单列) -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 items-stretch">
      <!-- 1. 外观与主题卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="palette" :size="18" class="text-text-main" />
              </div>
              <div>
                <h3 class="section-title text-[14px]">外观与界面主题</h3>
                <span class="text-[11px] text-text-muted">墨朱视觉体系与浅深色模式切换</span>
              </div>
            </div>

            <span class="text-[11px] font-semibold text-text-muted">
              当前: {{ currentMode === 'dark' ? '深色墨夜' : currentMode === 'light' ? '浅色宣纸' : '跟随系统' }}
            </span>
          </div>

          <!-- 明暗模式切换按钮组 -->
          <div class="space-y-2 mt-3.5">
            <label class="text-[11px] font-semibold text-text-muted block">界面色彩模式</label>
            <div class="flex flex-wrap gap-2.5">
              <button
                v-for="m in ([{ id: 'system', name: '跟随系统' }, { id: 'light', name: '浅色宣纸' }, { id: 'dark', name: '深色墨夜' }] as const)"
                :key="m.id"
                @click="handleModeChange(m.id)"
                class="app-btn-base flex-1 min-w-[90px]"
                :class="[
                  currentMode === m.id
                    ? 'bg-brand-primary text-surface-card border border-brand-primary dark:text-[#1a1a1c]'
                    : 'bg-surface-card-sub border border-surface-border text-text-sub hover:bg-surface-hover',
                ]"
              >
                {{ m.name }}
              </button>
            </div>
          </div>
        </div>

        <!-- 墨·朱 视觉规范说明 -->
        <div class="app-card-sub p-3 flex items-center gap-3">
          <div class="flex items-center -space-x-1 shrink-0">
            <span class="w-5 h-5 rounded-md bg-[#1A1A1C] border border-surface-border z-10" title="墨 #1A1A1C"></span>
            <span class="w-5 h-5 rounded-md bg-[#8A8B8E] border border-surface-border" title="灰 #8A8B8E"></span>
            <span class="w-5 h-5 rounded-md bg-[#F1EFE9] border border-surface-border" title="宣纸 #F1EFE9"></span>
            <span class="w-5 h-5 rounded-md bg-[#9E2B25] border border-surface-border" title="朱砂 #9E2B25"></span>
          </div>
          <div>
            <div class="text-[12px] font-semibold text-text-main">
              墨 · 朱 视觉规范
            </div>
            <div class="text-[11px] text-text-muted leading-tight mt-0.5">
              以黑灰白墨色为主调，朱砂红用于关键强调与安全拦截警示，深浅双色对等适配。
            </div>
          </div>
        </div>
      </div>

      <!-- 2. IPC 通信配置卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="sync" :size="18" class="text-text-main" />
              </div>
              <div>
                <h3 class="section-title text-[14px]">后台服务连接设置</h3>
                <span class="text-[11px] text-text-muted">前端客户端与核心服务的通信管道</span>
              </div>
            </div>

            <StatusBadge
              :status="connResult?.success ? 'active' : connResult ? 'error' : 'inactive'"
              :text="connResult?.success ? '通信正常' : connResult ? '连接异常' : '已配置'"
              size="sm"
            />
          </div>

          <div class="space-y-3 mt-3">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <md-outlined-text-field
                label="服务地址 (IP 或主机名)"
                :value="host"
                @input="host = ($event.target as any).value"
                class="w-full"
              ></md-outlined-text-field>

              <md-outlined-text-field
                label="通信端口"
                :value="port"
                @input="port = ($event.target as any).value"
                class="w-full font-mono"
              ></md-outlined-text-field>
            </div>

            <md-outlined-text-field
              label="访问密钥 / 授权 Token (可选)"
              :value="token"
              @input="token = ($event.target as any).value"
              type="password"
              placeholder="留空表示本机无需密码"
              class="w-full"
            ></md-outlined-text-field>
          </div>
        </div>

        <div class="flex items-center justify-between pt-2 border-t border-surface-border-sub">
          <div v-if="connResult" class="text-[11px] font-medium flex items-center gap-1.5 truncate max-w-[180px]" :class="connResult.success ? 'text-status-success' : 'text-status-error'">
            <M3Icon :name="connResult.success ? 'check_circle' : 'error'" :size="14" />
            <span class="truncate">{{ connResult.message }}</span>
          </div>
          <div v-else />

          <div class="flex items-center gap-2">
            <button
              type="button"
              @click="testConnection"
              :disabled="testingConnection"
              class="app-btn-secondary app-btn-compact"
            >
              <M3Icon name="sync" :size="14" :class="testingConnection ? 'animate-spin' : ''" />
              <span>测试连接</span>
            </button>
            <button
              type="button"
              @click="handleSaveIPC"
              class="app-btn-primary app-btn-compact"
            >
              <M3Icon name="check" :size="14" />
              <span>保存配置</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 3. 后台核心服务管理卡片 -->
      <CoreServiceCard />

      <!-- 4. 局域网 Web 远程管理控制台卡片 -->
      <LanWebCard />

      <!-- 5. Windows 开机自启托管卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="power_settings_new" :size="18" class="text-text-main" />
              </div>
              <div>
                <h3 class="section-title text-[14px]">开机自动启动</h3>
                <span class="text-[11px] text-text-muted">Windows 用户登录自启配置</span>
              </div>
            </div>

            <span v-if="webMode" class="text-[11px] text-text-muted bg-surface-card-sub px-2 py-0.5 rounded border border-surface-border-sub">宿主机桌面生效</span>
            <div v-else class="flex items-center gap-2">
              <span class="text-[11px] font-semibold text-text-sub hidden sm:inline">
                {{ autostart ? '自启已开启' : '未开启' }}
              </span>
              <md-switch
                :selected="autostart"
                :disabled="togglingAutostart"
                @change="handleToggleAutostart"
              />
            </div>
          </div>

          <div class="space-y-3 mt-3 text-xs text-text-sub leading-relaxed">
            <p>
              在 Windows 系统开机登录后自动启动桌面应用，确保 DNS 拦截、加速和物理网卡保护持续稳定运行。
            </p>
            <div class="app-card-sub p-3 space-y-1">
              <div class="flex items-center justify-between">
                <span class="text-[11px] font-semibold text-text-main">系统自启托管状态</span>
                <span class="font-semibold text-[12px]" :class="autostart ? 'text-status-success' : 'text-text-muted'">
                  {{ autostart ? '✔ 已加入 Windows 启动项' : '未配置开机启动' }}
                </span>
              </div>
              <p class="text-[11px] text-text-muted">
                可通过此处随时开启或关闭，无需手动配置注册表或任务计划。
              </p>
            </div>
          </div>
        </div>

        <div class="text-[11px] text-text-muted pt-2 border-t border-surface-border-sub">
          提示：若同时安装了系统后台服务，核心服务将随 Windows 启动优先就绪。
        </div>
      </div>

      <!-- 6. 网络诊断与应急自愈卡片 -->
      <div class="app-card p-5 flex flex-col justify-between gap-4">
        <div>
          <div class="flex items-center justify-between gap-3 pb-3 border-b border-surface-border-sub">
            <div class="flex items-center gap-2.5 min-w-0">
              <div class="w-8 h-8 rounded-md bg-surface-card-sub border border-surface-border-sub flex items-center justify-center shrink-0">
                <M3Icon name="build" :size="18" class="text-text-main" />
              </div>
              <div>
                <h3 class="section-title text-[14px]">网络诊断与应急修复</h3>
                <span class="text-[11px] text-text-muted">端口占用排查与一键防断网自愈</span>
              </div>
            </div>
          </div>

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
                    @click="handleAutofixInSettings"
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
                  <M3Icon v-if="restoring" name="refresh" :size="13" class="animate-spin" />
                  <M3Icon v-else name="refresh" :size="13" />
                  <span>一键恢复</span>
                </button>
              </div>
              <p class="text-[11px] text-text-muted">当软件异常退出导致无法上网时，一键恢复系统默认 DNS 并清理缓存</p>

              <div v-if="restoreMessage" class="p-2 rounded bg-status-success-bg text-status-success border border-status-success/30 text-[11px] font-semibold">
                {{ restoreMessage }}
              </div>
            </div>
          </div>
        </div>

        <div class="text-[11px] text-text-muted pt-2 border-t border-surface-border-sub">
          自愈操作具备原生离线防护，不会破坏现有硬件驱动。
        </div>
      </div>
    </div>

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
