<script setup lang="ts">
import { ref } from 'vue'
import type { UrlItem } from '../types'
import CopyButton from './CopyButton.vue'

defineProps<{ url: UrlItem }>()
const emit = defineEmits<{ 'create-another': [] }>()
const cardRef = ref<HTMLElement | null>(null)

defineExpose({ focusSelf: () => cardRef.value?.focus() })
</script>

<template>
  <div ref="cardRef" class="result card" tabindex="-1" role="status" aria-live="polite">
    <h2 class="result-title">Short URL created</h2>
    <p class="result-url mono">{{ url.short_url }}</p>
    <p class="result-target clamp-2">{{ url.target_url }}</p>
    <div class="result-actions">
      <CopyButton :text="url.short_url" variant="primary" />
      <a class="btn btn-secondary" :href="url.short_url" target="_blank" rel="noopener">Open</a>
      <button type="button" class="btn btn-ghost" @click="emit('create-another')">Create another</button>
    </div>
  </div>
</template>

<style scoped>
.result:focus {
  outline: none;
}
.result-title {
  font-size: var(--text-lg);
  color: var(--color-success);
}
.result-url {
  margin-top: var(--space-3);
  font-size: var(--text-lg);
  overflow-wrap: anywhere;
  background: var(--color-primary-soft);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
}
.result-target {
  color: var(--color-muted);
  font-size: var(--text-sm);
  margin-top: var(--space-2);
}
.result-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  margin-top: var(--space-4);
}
</style>
