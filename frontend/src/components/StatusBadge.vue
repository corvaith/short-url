<script setup lang="ts">
import { computed } from 'vue'
import type { UrlStatus } from '../types'

const props = defineProps<{ status: UrlStatus }>()

const label = computed(
  () => ({ active: 'Active', expired: 'Expired', disabled: 'Disabled' })[props.status],
)
</script>

<template>
  <span class="badge" :class="`badge-${status}`">
    <span class="dot" aria-hidden="true" />
    {{ label }}
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px var(--space-3);
  border-radius: var(--radius-full);
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
}
.badge-active {
  background: var(--color-success-soft);
  color: var(--color-success);
}
.badge-active .dot {
  background: var(--color-success);
}
.badge-expired {
  background: var(--color-warning-soft);
  color: var(--color-warning);
}
.badge-expired .dot {
  background: var(--color-warning);
}
.badge-disabled {
  background: var(--color-danger-soft);
  color: var(--color-danger);
}
.badge-disabled .dot {
  background: var(--color-danger);
}
</style>
