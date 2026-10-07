<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
}>()
</script>

<template>
  <div class="stat-card card">
    <p class="stat-label">{{ label }}</p>
    <p class="stat-value">{{ value }}</p>
  </div>
</template>

<style scoped>
.stat-card {
  padding: var(--space-5);
}
.stat-label {
  color: var(--color-muted);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
}
.stat-value {
  font-size: var(--text-2xl);
  font-weight: var(--weight-semibold);
  margin-top: var(--space-1);
  font-variant-numeric: tabular-nums;
}
</style>
