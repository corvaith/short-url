<script setup lang="ts">
import type { UrlItem } from '../types'
import { useBreakpoint } from '../composables/useBreakpoint'
import UrlCard from './UrlCard.vue'
import UrlTable from './UrlTable.vue'

defineProps<{ urls: UrlItem[] }>()
const emit = defineEmits<{ action: [id: string, key: string] }>()

const { isTableMode } = useBreakpoint()
</script>

<template>
  <UrlTable v-if="isTableMode" :urls="urls" @action="(id, key) => emit('action', id, key)" />
  <div v-else class="grid">
    <UrlCard v-for="u in urls" :key="u.id" :url="u" @action="emit('action', u.id, $event)" />
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
}
@media (min-width: 768px) and (max-width: 1279.98px) {
  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
