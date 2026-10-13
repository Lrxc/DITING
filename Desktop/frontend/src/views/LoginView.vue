<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { ipc } from '../api/ipc';
import { themeManager, type ThemeMode } from '../theme/theme';
import {
  loadLoginBgConfig,
  saveLoginBgConfig,
  resolveBackgroundUrl,
  type LoginBgConfig,
} from '../utils/imageCompressor';
import { useAuthForm } from '../composables/useAuthForm';
import M3Icon from '../components/M3Icon.vue';
import InlineAlert from '../components/ui/InlineAlert.vue';
import LoginBgModal from '../components/auth/LoginBgModal.vue';
import appLogo from '../assets/images/logo-universal.png';

const props = withDefaults(
  defineProps<{
    initialMode?: 'login' | 'setup';
  }>(),
  {
    initialMode: 'login',
  }
);

const emit = defineEmits<{
  (e: 'success'): void;
}>();

const {
  currentMode,
  username,
  password,
  confirmPassword,
  showPassword,
  showConfirmPassword,
  loading,
  errorMsg,
  successMsg,
  lockoutSeconds,
  checkAuthStatus,
  submitSetup,
  submitLogin,
} = useAuthForm(props.initialMode);

watch(
  () => props.initialMode,
  (val) => {
    if (val) currentMode.value = val;
  }
);

// 背景图配置
const bgConfig = ref<LoginBgConfig>(loadLoginBgConfig());
const showBgModal = ref(false);

// 主题模式
const currentThemeMode = ref<ThemeMode>(themeManager.getThemeMode());

// 注入版本号
const appVersion = __APP_VERSION__;

const hasCustomBg = computed(() => bgConfig.value.type !== 'none' && Boolean(bgConfig.value.url));

const bgStyle = computed(() => {
  if (!bgConfig.value.url) return {};
  const cfg = ipc.getConfig();
  const baseUrl = cfg.host ? `http://${cfg.host}:${cfg.port}` : undefined;
  const fullUrl = resolveBackgroundUrl(bgConfig.value.url, baseUrl);
  const safeUrl = fullUrl.replace(/'/g, "\\'");
  return {
    backgroundImage: `url('${safeUrl}')`,
    filter: bgConfig.value.blur > 0 ? `blur(${bgConfig.value.blur}px)` : 'none',
    transform: bgConfig.value.blur > 0 ? 'scale(1.05)' : 'none',
  };
});

function cycleTheme() {
  const modes: ThemeMode[] = ['light', 'dark', 'system'];
  const nextIdx = (modes.indexOf(currentThemeMode.value) + 1) % modes.length;
  const nextMode = modes[nextIdx] ?? 'system';
  currentThemeMode.value = nextMode;
  themeManager.setThemeMode(nextMode);
}

const themeIcon = computed(() => {
  if (currentThemeMode.value === 'dark') return 'contrast';
  if (currentThemeMode.value === 'light') return 'palette';
  return 'settings';
});

const themeLabel = computed(() => {
  if (currentThemeMode.value === 'dark') return '深色模式';
  if (currentThemeMode.value === 'light') return '浅色模式';
  return '跟随系统';
});

async function handleSubmit() {
  let ok = false;
  if (currentMode.value === 'setup') {
    ok = await submitSetup();
  } else {
    ok = await submitLogin();
  }
  if (ok) {
    const delay = currentMode.value === 'setup' ? 700 : 500;
    setTimeout(() => {
      emit('success');
    }, delay);
  }
}

function handleBgChange(newCfg: LoginBgConfig) {
  const cfg = ipc.getConfig();
  const baseUrl = cfg.host ? `http://${cfg.host}:${cfg.port}` : undefined;
  const resolved = resolveBackgroundUrl(newCfg.url, baseUrl);
  bgConfig.value = {
    ...newCfg,
    url: resolved,
  };
}

async function fetchServerBackground() {
  try {
    const remote = await ipc.getLoginBackground();
    if (remote) {
      const cfg = ipc.getConfig();
      const baseUrl = cfg.host ? `http://${cfg.host}:${cfg.port}` : undefined;
      const resolved = resolveBackgroundUrl(remote.url, baseUrl);
      bgConfig.value = {
        type: remote.type,
        url: resolved,
        opacity: remote.opacity ?? 40,
        blur: remote.blur ?? 0,
        updatedAt: remote.updatedAt,
      };
      saveLoginBgConfig(remote.type, resolved, remote.opacity ?? 40, remote.blur ?? 0);
    }
  } catch {}
}

let unsubEvents: (() => void) | null = null;
let syncTimer: ReturnType<typeof setInterval> | null = null;

function handleVisibilityOrFocus() {
  if (typeof document !== 'undefined' && document.visibilityState === 'visible') {
    fetchServerBackground();
  }
}

onMounted(() => {
  checkAuthStatus(props.initialMode);
  fetchServerBackground();
  syncTimer = setInterval(fetchServerBackground, 30000);
  if (typeof window !== 'undefined') {
    window.addEventListener('visibilitychange', handleVisibilityOrFocus);
    window.addEventListener('focus', handleVisibilityOrFocus);
  }
  unsubEvents = ipc.onEvent((event) => {
    if (event.type === 'background') {
      fetchServerBackground();
    }
  });
});

onUnmounted(() => {
  if (syncTimer) clearInterval(syncTimer);
  if (typeof window !== 'undefined') {
    window.removeEventListener('visibilitychange', handleVisibilityOrFocus);
    window.removeEventListener('focus', handleVisibilityOrFocus);
  }
  if (unsubEvents) unsubEvents();
});
</script>

<template>
  <div class="relative min-h-screen w-full flex flex-col justify-between overflow-x-hidden overflow-y-auto font-sans select-none bg-surface-base text-text-main">
    <!-- 1. 自定义壁纸背景层 (支持平滑过渡与模糊) -->
    <div
      v-if="hasCustomBg"
      class="fixed inset-0 z-0 bg-cover bg-center bg-no-repeat transition-all duration-700 pointer-events-none"
      :style="bgStyle"
    />

    <!-- 2. 背景遮罩层 (保证前景卡片文字高清晰度) -->
    <div
      v-if="hasCustomBg"
      class="fixed inset-0 z-0 bg-black pointer-events-none transition-opacity duration-300"
      :style="{ opacity: bgConfig.opacity / 100 }"
    />

    <!-- 顶栏辅助操作：主题切换与自定义壁纸入口 -->
    <header class="relative z-10 w-full px-4 sm:px-8 py-4 flex items-center justify-between">
      <div
        class="flex items-center gap-2.5 transition-all duration-300"
        :class="hasCustomBg ? 'px-3 py-1.5 rounded-lg bg-surface-card/75 backdrop-blur-md border border-surface-border/60 shadow-xs' : ''"
      >
        <img :src="appLogo" alt="Logo" class="w-7 h-7 object-contain rounded-md" />
        <span class="font-serif text-sm font-semibold tracking-wide text-text-main">谛听 DITING</span>
      </div>

      <div class="flex items-center gap-2">
        <!-- 自定义背景入口按钮 -->
        <button
          type="button"
          @click="showBgModal = true"
          class="app-btn-secondary app-btn-compact relative backdrop-blur-md"
          :class="hasCustomBg ? 'border-accent-seal/50 text-accent-seal bg-surface-card/60' : 'bg-surface-card/70'"
          title="配置登录页自定义背景图片"
        >
          <M3Icon name="image" :size="14" />
          <span class="hidden sm:inline">自定义背景</span>
          <span
            v-if="hasCustomBg"
            class="w-1.5 h-1.5 rounded-full bg-accent-seal"
            title="已启用自定义背景"
          />
        </button>

        <!-- 主题切换按钮 -->
        <button
          type="button"
          @click="cycleTheme"
          class="app-btn-secondary app-btn-compact backdrop-blur-md bg-surface-card/70"
          :title="`当前主题：${themeLabel}（点击切换）`"
        >
          <M3Icon :name="themeIcon" :size="14" />
          <span class="hidden sm:inline">{{ themeLabel }}</span>
        </button>
      </div>
    </header>

    <!-- 中部主登录卡片容器（自适应不同屏幕尺寸） -->
    <main class="relative z-10 flex-1 flex items-center justify-center p-4 sm:p-6 py-6 sm:py-8">
      <div
        class="w-full max-w-sm sm:max-w-md rounded-xl transition-all duration-300"
        :class="[
          hasCustomBg
            ? 'bg-surface-card/90 dark:bg-surface-card/85 backdrop-blur-md shadow-2xl border border-surface-border/80'
            : 'bg-surface-card shadow-xl border border-surface-border'
        ]"
      >
        <!-- 卡片头部品牌标识 -->
        <div class="p-6 sm:p-8 pb-4 text-center">
          <div class="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-brand-container shadow-xs mb-3">
            <img :src="appLogo" alt="DITING Logo" class="w-8 h-8 object-contain rounded-md" />
          </div>
          <h1 class="font-serif text-xl sm:text-2xl font-bold tracking-tight text-text-main">
            {{ currentMode === 'setup' ? '初始化管理员账号' : '谛听 控制台登录' }}
          </h1>
          <p class="text-xs sm:text-[13px] text-text-muted mt-1 leading-relaxed">
            {{ currentMode === 'setup' ? '首次进入局域网 Web 控制台，请设置管理员凭据' : '安全可信的局域网 DNS 加速与防护引擎' }}
          </p>
        </div>

        <!-- 表单输入区域 -->
        <form @submit.prevent="handleSubmit" class="p-6 sm:p-8 pt-2 space-y-4">
          <InlineAlert v-if="errorMsg" :message="errorMsg" type="error" />
          <InlineAlert v-if="successMsg" :message="successMsg" type="success" />

          <!-- 安全锁定倒计时提示 -->
          <div
            v-if="lockoutSeconds > 0"
            class="p-3 rounded-lg bg-status-warning-bg border border-status-warning/30 text-status-warning text-xs flex items-center gap-2"
          >
            <M3Icon name="alarm" :size="16" class="shrink-0" />
            <span>安全保护频控锁定中，请等待 {{ lockoutSeconds }} 秒后再试</span>
          </div>

          <!-- 用户名 -->
          <div class="space-y-1">
            <label class="block text-xs font-medium text-text-sub">管理员账号</label>
            <div class="relative">
              <input
                v-model="username"
                type="text"
                autocomplete="username"
                :disabled="loading || lockoutSeconds > 0"
                class="w-full h-10 px-3 text-sm rounded-md border border-surface-border bg-surface-card text-text-main focus:outline-none focus:border-brand-primary transition-colors disabled:opacity-50"
                placeholder="admin"
              />
            </div>
          </div>

          <!-- 密码输入框 -->
          <div class="space-y-1">
            <div class="flex items-center justify-between">
              <label class="block text-xs font-medium text-text-sub">
                {{ currentMode === 'setup' ? '设置初始密码 (至少 4 位)' : '管理员密码' }}
              </label>
            </div>
            <div class="relative flex items-center">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                :disabled="loading || lockoutSeconds > 0"
                class="w-full h-10 pl-3 pr-10 text-sm rounded-md border border-surface-border bg-surface-card text-text-main focus:outline-none focus:border-brand-primary transition-colors disabled:opacity-50"
                placeholder="请输入密码"
              />
              <button
                type="button"
                @click="showPassword = !showPassword"
                class="absolute right-2.5 p-1 text-text-muted hover:text-text-main cursor-pointer"
                title="显隐密码"
              >
                <M3Icon :name="showPassword ? 'visibility_off' : 'visibility'" :size="16" />
              </button>
            </div>
          </div>

          <!-- 确认密码（仅设置模式展示） -->
          <div v-if="currentMode === 'setup'" class="space-y-1">
            <label class="block text-xs font-medium text-text-sub">确认初始密码</label>
            <div class="relative flex items-center">
              <input
                v-model="confirmPassword"
                :type="showConfirmPassword ? 'text' : 'password'"
                autocomplete="new-password"
                :disabled="loading || lockoutSeconds > 0"
                class="w-full h-10 pl-3 pr-10 text-sm rounded-md border border-surface-border bg-surface-card text-text-main focus:outline-none focus:border-brand-primary transition-colors disabled:opacity-50"
                placeholder="请再次输入以确认"
              />
              <button
                type="button"
                @click="showConfirmPassword = !showConfirmPassword"
                class="absolute right-2.5 p-1 text-text-muted hover:text-text-main cursor-pointer"
                title="显隐确认密码"
              >
                <M3Icon :name="showConfirmPassword ? 'visibility_off' : 'visibility'" :size="16" />
              </button>
            </div>
          </div>

          <!-- 模式切换与操作提示 -->
          <div class="pt-2">
            <button
              type="submit"
              :disabled="loading || lockoutSeconds > 0"
              class="w-full h-10 rounded-md font-semibold text-sm transition-all flex items-center justify-center gap-2 cursor-pointer shadow-xs disabled:opacity-50 disabled:cursor-not-allowed app-btn-primary"
            >
              <M3Icon v-if="loading" name="refresh" :size="16" class="animate-spin" />
              <M3Icon v-else :name="currentMode === 'setup' ? 'key' : 'lock_open'" :size="16" />
              <span>
                {{
                  loading
                    ? (currentMode === 'setup' ? '正在创建管理员账号...' : '正在验证安全凭证...')
                    : (currentMode === 'setup' ? '创建账号并登入系统' : '安全登录')
                }}
              </span>
            </button>
          </div>
        </form>

        <!-- 卡片底部说明与系统保障 -->
        <div class="px-6 sm:px-8 py-3.5 bg-surface-card-sub/60 rounded-b-xl border-t border-surface-border-sub flex items-center justify-between text-[11px] text-text-muted">
          <span class="flex items-center gap-1">
            <M3Icon name="verified_user" :size="13" class="text-status-success" />
            <span>SHA-256 安全哈希保护</span>
          </span>
          <span class="font-mono">v{{ appVersion }}</span>
        </div>
      </div>
    </main>

    <!-- 底栏版权与声明 -->
    <footer class="relative z-10 w-full px-4 sm:px-8 py-3 text-center text-xs text-text-muted">
      <span>谛听 DITING DNS · 跨平台局域网安全加速与网络托管引擎</span>
    </footer>

    <!-- 自定义壁纸配置弹窗 -->
    <LoginBgModal
      :open="showBgModal"
      :currentConfig="bgConfig"
      @close="showBgModal = false"
      @change="handleBgChange"
    />
  </div>
</template>
