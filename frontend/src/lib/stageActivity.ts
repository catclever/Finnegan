export interface StageActivityCallbacks {
  onFocusChange: (isFocused: boolean) => void;
  onSleepChange: (isSleeping: boolean) => void;
  canSleep: () => boolean;
}

export interface StageActivityController {
  resetActivityTimer: () => void;
  destroy: () => void;
}

export function createStageActivityController(
  callbacks: StageActivityCallbacks,
  idleTimeout = 3000
): StageActivityController {
  let timer: any = null;

  function resetActivityTimer() {
    callbacks.onSleepChange(false);
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      if (callbacks.canSleep()) {
        callbacks.onSleepChange(true);
      }
    }, idleTimeout);
  }

  function handleFocus() {
    callbacks.onFocusChange(true);
    resetActivityTimer();
  }

  function handleBlur() {
    callbacks.onFocusChange(false);
  }

  function handleVisibility() {
    const isVisible = typeof document !== 'undefined' ? !document.hidden : true;
    callbacks.onFocusChange(isVisible);
    if (isVisible) {
      resetActivityTimer();
    }
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('focus', handleFocus);
    window.addEventListener('blur', handleBlur);
  }
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', handleVisibility);
  }
  resetActivityTimer();

  return {
    resetActivityTimer,
    destroy() {
      if (timer) clearTimeout(timer);
      if (typeof window !== 'undefined') {
        window.removeEventListener('focus', handleFocus);
        window.removeEventListener('blur', handleBlur);
      }
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', handleVisibility);
      }
    }
  };
}
