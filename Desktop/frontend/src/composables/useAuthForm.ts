import { ref, onUnmounted } from 'vue';
import { ipc } from '../api/ipc';

export type AuthMode = 'login' | 'setup' | 'change_password';

export function useAuthForm(initialMode: AuthMode = 'login') {
  const currentMode = ref<AuthMode>(initialMode);
  const username = ref('admin');
  const password = ref('');
  const confirmPassword = ref('');
  const oldPassword = ref('');
  const showPassword = ref(false);
  const showConfirmPassword = ref(false);

  const loading = ref(false);
  const errorMsg = ref('');
  const successMsg = ref('');
  const lockoutSeconds = ref(0);
  let lockoutTimer: ReturnType<typeof setInterval> | null = null;
  const isAuthInitialized = ref(true);

  function startLockoutCountdown(seconds: number) {
    lockoutSeconds.value = seconds;
    if (lockoutTimer) clearInterval(lockoutTimer);
    lockoutTimer = setInterval(() => {
      lockoutSeconds.value--;
      if (lockoutSeconds.value <= 0) {
        if (lockoutTimer) clearInterval(lockoutTimer);
        lockoutTimer = null;
        errorMsg.value = '';
      }
    }, 1000);
  }

  async function checkAuthStatus(preferredMode?: AuthMode) {
    try {
      const status = await ipc.getAuthStatus();
      isAuthInitialized.value = Boolean(status.initialized);
      if (!status.initialized) {
        currentMode.value = 'setup';
      } else if (preferredMode) {
        currentMode.value = preferredMode;
      }
      if (status.username) {
        username.value = status.username;
      }
      if (status.locked && status.lockoutRemainingSec > 0) {
        startLockoutCountdown(status.lockoutRemainingSec);
      }
    } catch {}
  }

  function resetForm() {
    password.value = '';
    confirmPassword.value = '';
    oldPassword.value = '';
    errorMsg.value = '';
    successMsg.value = '';
  }

  async function submitSetup(): Promise<boolean> {
    if (!username.value.trim()) {
      errorMsg.value = '管理员用户名不能为空';
      return false;
    }
    if (password.value.length < 4) {
      errorMsg.value = '密码长度至少为 4 位字符';
      return false;
    }
    if (password.value !== confirmPassword.value) {
      errorMsg.value = '两次输入的密码不一致';
      return false;
    }

    loading.value = true;
    errorMsg.value = '';
    try {
      await ipc.setupAuth({
        username: username.value.trim(),
        password: password.value,
      });
      await ipc.login({
        username: username.value.trim(),
        password: password.value,
      });
      successMsg.value = '管理员密码设置成功！正在进入系统...';
      return true;
    } catch (err: any) {
      errorMsg.value = err.message || '初始化管理员账号失败';
      return false;
    } finally {
      loading.value = false;
    }
  }

  async function submitLogin(): Promise<boolean> {
    if (!username.value.trim()) {
      errorMsg.value = '请输入用户名';
      return false;
    }
    if (!password.value) {
      errorMsg.value = '请输入密码';
      return false;
    }

    loading.value = true;
    errorMsg.value = '';
    try {
      await ipc.login({
        username: username.value.trim(),
        password: password.value,
      });
      successMsg.value = '管理员身份验证成功，正在进入控制台...';
      return true;
    } catch (err: any) {
      const msg = err.message || '';
      if (msg.includes('未初始化')) {
        currentMode.value = 'setup';
        errorMsg.value = '检测到管理员账号尚未初始化，请在此处设置初始密码以完成配置';
      } else if (msg.includes('锁定') || msg.includes('多次错误') || msg.includes('locked')) {
        await checkAuthStatus();
        errorMsg.value = '密码错误次数过多，请稍候再试';
      } else {
        errorMsg.value = msg || '用户名或密码错误';
      }
      return false;
    } finally {
      loading.value = false;
    }
  }

  async function submitChangePassword(): Promise<boolean> {
    if (!oldPassword.value) {
      errorMsg.value = '请输入当前原密码';
      return false;
    }
    if (password.value.length < 4) {
      errorMsg.value = '新密码长度至少为 4 位字符';
      return false;
    }
    if (password.value !== confirmPassword.value) {
      errorMsg.value = '两次输入的新密码不一致';
      return false;
    }

    loading.value = true;
    errorMsg.value = '';
    try {
      await ipc.changePassword({
        username: username.value.trim() || 'admin',
        oldPassword: oldPassword.value,
        newPassword: password.value,
      });
      successMsg.value = '管理员密码修改成功！请使用新密码重新登录。';
      return true;
    } catch (err: any) {
      errorMsg.value = err.message || '修改密码失败';
      return false;
    } finally {
      loading.value = false;
    }
  }

  onUnmounted(() => {
    if (lockoutTimer) clearInterval(lockoutTimer);
  });

  return {
    currentMode,
    username,
    password,
    confirmPassword,
    oldPassword,
    showPassword,
    showConfirmPassword,
    loading,
    errorMsg,
    successMsg,
    lockoutSeconds,
    isAuthInitialized,
    checkAuthStatus,
    resetForm,
    submitSetup,
    submitLogin,
    submitChangePassword,
  };
}
