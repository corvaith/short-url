<script setup lang="ts">
defineProps<{ message: string }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="error card" role="alert">
    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="var(--color-danger)" stroke-width="1.5" stroke-linecap="round" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8v4.5M12 16h.01" />
    </svg>
    <h2 class="error-title">Something went wrong</h2>
    <p class="error-msg">{{ message }}</p>
    <button type="button" class="btn btn-secondary" @click="emit('retry')">Try again</button>
  </div>
</template>

<style scoped>
.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  text-align: center;
  padding: var(--space-10) var(--space-6);
}
.error-title {
  font-size: var(--text-lg);
}
.error-msg {
  color: var(--color-muted);
  font-size: var(--text-sm);
  max-width: 44ch;
  overflow-wrap: anywhere;
}
</style>
