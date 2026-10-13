<script setup lang="ts">
import { watch } from 'vue';
import { useAuthForm, type AuthMode } from '../composables/useAuthForm';
import M3Icon from './M3Icon.vue';
import AppModal from './AppModal.vue';
import InlineAlert from './ui/InlineAlert.vue';

const props = withDefaults(
  defineProps<{
    open: boolean;
    mode?: AuthMode;
    forceModal?: boolean;
  }>(),
  {
    open: false,
    mode: 'login',
    forceModal: false,
  }
);

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'success'): void;
}>();

const {
  currentMode,
  username,
  password,
  confirmPassword,
  oldPassword,
  loading,
  errorMsg,
  successMsg,
  lockoutSeconds,
  checkAuthStatus,
  resetForm,
  submitSetup,
  submitLogin,
  submitChangePassword,
} = useAuthForm(props.mode);

watch(
  () => props.open,
  (val) => {
    if (val) {
      resetForm();
      checkAuthStatus(props.mode);
    }
  }
);

watch(
  () => props.mode,
  (val) => {
    if (val) currentMode.value = val;
  }
);

async function handleSubmit() {
  let ok = false;
  if (currentMode.value === 'setup') {
    ok = await submitSetup();
  } else if (currentMode.value === 'login') {
    ok = await submitLogin();
  } else if (currentMode.value === 'change_password') {
    ok = await submitChangePassword();
  }

  if (ok) {
    const delay = currentMode.value === 'change_password' ? 1500 : 800;
    setTimeout(() => {
      emit('success');
      emit('close');
    }, delay);
  }
}

function handleClose() {
  if (props.forceModal) return;
  emit('close');
}
</script>

<template>
  <AppModal
    :open="open"
    @close="handleClose"
    max-width="max-w-md"
  >
    <template #headline>
      <div class="flex items-center gap-3 w-full">
        <div class="w-10 h-10 rounded-md bg-brand-container flex items-center justify-center shrink-0">
          <M3Icon
            :name="currentMode === 'setup' ? 'key' : (currentMode === 'change_password' ? 'lock_reset' : 'verified_user')"
            :size="22"
            class="text-text-main"
          />
        </div>
        <div>
          <h3 class="section-title">
            {{ currentMode === 'setup' ? '初始化管理员账号' : (currentMode === 'change_password' ? '修改管理员密码' : '登录谛听 Web 管理控制台') }}
          </h3>
          <p class="text-xs text-text-sub mt-0.5">
            {{ currentMode === 'setup' ? '设置初始账号与密码，守护局域网远程管理安全' : (currentMode === 'change_password' ? '更新凭据后所有外部会话将自动失效' : '请输入管理员凭据以远程管理谛听 DNS') }}
          </p>
        </div>
      </div>
    </template>

    <div class="space-y-4 py-1">
      <InlineAlert v-if="errorMsg" :message="errorMsg" type="error" />
      <InlineAlert v-if="successMsg" :message="successMsg" type="success" />

      <div v-if="lockoutSeconds > 0" class="p-3 rounded-md bg-status-warning-bg border border-status-warning/30 text-status-warning text-xs flex items-center gap-2">
        <M3Icon name="alarm" :size="16" class="shrink-0" />
        <span>安全保护锁定中，请在 {{ lockoutSeconds }} 秒后再试</span>
      </div>

      <div class="space-y-3.5">
        <div>
          <md-outlined-text-field
            label="管理员账号"
            :value="username"
            @input="username = ($event.target as any).value"
            :disabled="loading || lockoutSeconds > 0"
            class="w-full"
          ></md-outlined-text-field>
        </div>

        <div v-if="currentMode === 'change_password'">
          <md-outlined-text-field
            label="原密码 (受信任宿主机桌面可留空)"
            type="password"
            :value="oldPassword"
            @input="oldPassword = ($event.target as any).value"
            :disabled="loading || lockoutSeconds > 0"
            class="w-full"
          ></md-outlined-text-field>
        </div>

        <div>
          <md-outlined-text-field
            :label="currentMode === 'change_password' ? '新密码 (至少 4 位)' : (currentMode === 'setup' ? '设置管理员密码 (至少 4 位)' : '管理员密码')"
            type="password"
            :value="password"
            @input="password = ($event.target as any).value"
            @keydown.enter="handleSubmit"
            :disabled="loading || lockoutSeconds > 0"
            class="w-full"
          ></md-outlined-text-field>
        </div>

        <div v-if="currentMode === 'setup' || currentMode === 'change_password'">
          <md-outlined-text-field
            label="确认新密码"
            type="password"
            :value="confirmPassword"
            @input="confirmPassword = ($event.target as any).value"
            @keydown.enter="handleSubmit"
            :disabled="loading || lockoutSeconds > 0"
            class="w-full"
          ></md-outlined-text-field>
        </div>
      </div>

      <div class="p-3 rounded-md bg-surface-card-sub border border-surface-border text-[11px] text-text-sub space-y-1">
        <div class="flex items-center gap-1.5 font-medium text-text-main">
          <M3Icon name="shield" :size="14" class="text-text-main" />
          <span>安全认证提示</span>
        </div>
        <p>• 登录凭据基于 SHA-256 加盐安全哈希存储，有效防范暴力碰撞与侧信道探测。</p>
        <p>• 连续输错密码 5 次将触发 5 分钟频控锁定，保障家庭局域网访问安全。</p>
      </div>
    </div>

    <template #actions>
      <button
        v-if="!forceModal"
        type="button"
        @click="handleClose"
        :disabled="loading"
        class="app-btn-secondary"
      >
        取消
      </button>
      <button
        type="button"
        @click="handleSubmit"
        :disabled="loading || lockoutSeconds > 0"
        class="app-btn-primary"
      >
        <M3Icon :name="loading ? 'sync' : 'check'" :size="16" :class="loading ? 'animate-spin' : ''" />
        <span>{{ currentMode === 'setup' ? '完成初始化并登录' : (currentMode === 'change_password' ? '保存新密码' : '立即登录') }}</span>
      </button>
    </template>
  </AppModal>
</template>
