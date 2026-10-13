import { ref, onUnmounted } from 'vue';

/**
 * 通用定时闪烁提示（支持成功与错误消息的自动清理）
 */
export function useFlashMessage(defaultDurationMs = 3000) {
  const successMessage = ref('');
  const errorMessage = ref('');
  let successTimer: ReturnType<typeof setTimeout> | null = null;
  let errorTimer: ReturnType<typeof setTimeout> | null = null;

  function flashSuccess(msg: string, durationMs = defaultDurationMs) {
    if (successTimer) clearTimeout(successTimer);
    successMessage.value = msg;
    successTimer = setTimeout(() => {
      successMessage.value = '';
      successTimer = null;
    }, durationMs);
  }

  function flashError(msg: string, durationMs = defaultDurationMs) {
    if (errorTimer) clearTimeout(errorTimer);
    errorMessage.value = msg;
    errorTimer = setTimeout(() => {
      errorMessage.value = '';
      errorTimer = null;
    }, durationMs);
  }

  function clearMessages() {
    if (successTimer) {
      clearTimeout(successTimer);
      successTimer = null;
    }
    if (errorTimer) {
      clearTimeout(errorTimer);
      errorTimer = null;
    }
    successMessage.value = '';
    errorMessage.value = '';
  }

  onUnmounted(() => {
    if (successTimer) clearTimeout(successTimer);
    if (errorTimer) clearTimeout(errorTimer);
  });

  return {
    successMessage,
    errorMessage,
    flashSuccess,
    flashError,
    clearMessages,
  };
}
