<script setup lang="ts">
import { useBreakpoint } from '../composables/useBreakpoint'
import { useToast, type ToastKind } from '../composables/useToast'

const { toasts, dismiss } = useToast()
const { isMobile } = useBreakpoint()

const icons: Record<ToastKind, string> = {
  success: 'M5 13l4 4 10-10',
  error: 'M12 8v4.5M12 16h.01',
  info: 'M12 8h.01M12 11v5',
}
</script>

<template>
  <div class="toast-host" :class="{ mobile: isMobile }" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <div v-for="t in toasts" :key="t.id" class="toast" :class="`toast-${t.kind}`" role="status">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <path :d="icons[t.kind]" />
        </svg>
        <span class="toast-msg">{{ t.message }}</span>
        <button type="button" class="toast-close" aria-label="Dismiss notification" @click="dismiss(t.id)">×</button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  top: calc(var(--topbar-h) + var(--space-4));
  right: var(--space-4);
  z-index: 150;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-width: 22rem;
  pointer-events: none;
}
.toast-host.mobile {
  top: auto;
  right: auto;
  left: var(--space-4);
  right: var(--space-4);
  bottom: calc(var(--bottomnav-h) + env(safe-area-inset-bottom) + var(--space-3));
  max-width: none;
}
.toast {
  pointer-events: auto;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-left: 4px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
  box-shadow: var(--shadow-md);
  font-size: var(--text-sm);
}
.toast-success {
  border-left-color: var(--color-success);
  color: var(--color-success);
}
.toast-error {
  border-left-color: var(--color-danger);
  color: var(--color-danger);
}
.toast-info {
  border-left-color: var(--color-primary);
  color: var(--color-primary);
}
.toast-msg {
  color: var(--color-text);
  flex: 1;
}
.toast-close {
  border: none;
  background: none;
  color: var(--color-muted);
  font-size: var(--text-lg);
  cursor: pointer;
  min-width: 32px;
  min-height: 32px;
  line-height: 1;
}
.toast-enter-active,
.toast-leave-active {
  transition: opacity var(--duration-base) var(--ease), transform var(--duration-base) var(--ease);
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
