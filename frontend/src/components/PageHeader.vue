<script setup lang="ts">
defineProps<{ title: string; subtitle?: string }>()
</script>

<template>
  <div class="page-header">
    <div>
      <h1 class="page-title" tabindex="-1">{{ title }}</h1>
      <p v-if="subtitle" class="page-subtitle">{{ subtitle }}</p>
    </div>
    <div class="page-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
}
.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--weight-semibold);
}
.page-title:focus {
  outline: none;
}
.page-subtitle {
  color: var(--color-muted);
  margin-top: var(--space-1);
  font-size: var(--text-sm);
}
</style>
