<script setup lang="ts">
import type { UrlItem } from '../types'
import { useClipboard } from '../composables/useClipboard'
import { useToast } from '../composables/useToast'
import StatusBadge from './StatusBadge.vue'
import ActionMenu from './ActionMenu.vue'
import type { MenuItem } from './ActionMenu.vue'

const props = defineProps<{ url: UrlItem }>()
const emit = defineEmits<{ action: [key: string] }>()

const { copy } = useClipboard()
const toast = useToast()

const menuItems: MenuItem[] = [
  { key: 'details', label: 'Details' },
  { key: 'toggle', label: 'Disable' },
  { key: 'delete', label: 'Delete', danger: true },
]

function itemsFor(u: UrlItem): MenuItem[] {
  return menuItems.map((i) =>
    i.key === 'toggle' ? { ...i, label: u.is_active ? 'Disable' : 'Enable' } : i,
  )
}

async function onCopy(): Promise<void> {
  const ok = await copy(props.url.short_url)
  toast.info(ok ? 'Copied to clipboard' : 'Copy failed')
}

function onMenu(key: string): void {
  emit('action', key)
}

function fmtDate(iso: string | null): string {
  if (!iso) return ''
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}
</script>

<template>
  <article class="url-card card">
    <a class="short mono" :href="`/urls/${url.id}`">{{ url.short_url }}</a>
    <a
      class="target clamp-2"
      :href="url.target_url"
      target="_blank"
      rel="noopener"
      :title="url.target_url"
    >
      {{ url.target_url }}
    </a>
    <p class="meta">{{ url.clicks }} clicks · <StatusBadge :status="url.status" /></p>
    <p class="meta muted">
      Created {{ fmtDate(url.created_at)
      }}<template v-if="url.expires_at"> · Expires {{ fmtDate(url.expires_at) }}</template>
    </p>
    <div class="actions">
      <button type="button" class="btn btn-secondary btn-sm" @click="onCopy">Copy</button>
      <a class="btn btn-secondary btn-sm" :href="url.short_url" target="_blank" rel="noopener"
        >Open</a
      >
      <ActionMenu :items="itemsFor(url)" @select="onMenu" />
    </div>
  </article>
</template>

<style scoped>
.url-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-5);
}
.short {
  font-weight: var(--weight-semibold);
  font-size: var(--text-base);
  color: var(--color-primary);
  text-decoration: none;
  overflow-wrap: anywhere;
}
.short:hover {
  text-decoration: underline;
}
.target {
  color: var(--color-muted);
  font-size: var(--text-sm);
  text-decoration: none;
}
.target:hover {
  color: var(--color-text);
}
.meta {
  font-size: var(--text-sm);
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.meta.muted {
  color: var(--color-muted);
  font-size: var(--text-xs);
}
.actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: var(--space-1);
}
</style>
