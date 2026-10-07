<script setup lang="ts">
import type { UrlItem } from '../types'
import StatusBadge from './StatusBadge.vue'
import ActionMenu from './ActionMenu.vue'
import type { MenuItem } from './ActionMenu.vue'
import { useClipboard } from '../composables/useClipboard'
import { useToast } from '../composables/useToast'

defineProps<{ urls: UrlItem[] }>()
const emit = defineEmits<{ action: [id: string, key: string] }>()

const { copy } = useClipboard()
const toast = useToast()

function itemsFor(u: UrlItem): MenuItem[] {
  return [
    { key: 'details', label: 'Details' },
    { key: 'toggle', label: u.is_active ? 'Disable' : 'Enable' },
    { key: 'delete', label: 'Delete', danger: true },
  ]
}

async function onCopy(u: UrlItem): Promise<void> {
  const ok = await copy(u.short_url)
  toast.info(ok ? 'Copied to clipboard' : 'Copy failed')
}

function fmtDate(iso: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}
</script>

<template>
  <div class="table-wrap card">
    <table class="url-table">
      <caption class="sr-only">
        Your shortened URLs with clicks, status, and dates
      </caption>
      <thead>
        <tr>
          <th scope="col">Short URL</th>
          <th scope="col" class="num">Clicks</th>
          <th scope="col">Status</th>
          <th scope="col">Created</th>
          <th scope="col">Expires</th>
          <th scope="col">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="u in urls" :key="u.id">
          <td class="cell-url">
            <a class="mono short" :href="`/urls/${u.id}`">{{ u.short_url }}</a>
            <span class="target clamp-2" :title="u.target_url">{{ u.target_url }}</span>
          </td>
          <td class="num">{{ u.clicks }}</td>
          <td><StatusBadge :status="u.status" /></td>
          <td>{{ fmtDate(u.created_at) }}</td>
          <td>{{ fmtDate(u.expires_at) }}</td>
          <td class="cell-actions">
            <button type="button" class="btn btn-secondary btn-sm" @click="onCopy(u)">Copy</button>
            <a class="btn btn-secondary btn-sm" :href="u.short_url" target="_blank" rel="noopener"
              >Open</a
            >
            <ActionMenu :items="itemsFor(u)" @select="emit('action', u.id, $event)" />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  padding: 0;
  overflow-x: auto;
}
.url-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
th,
td {
  text-align: left;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  vertical-align: top;
}
th {
  font-weight: var(--weight-semibold);
  color: var(--color-muted);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
tbody tr:last-child td {
  border-bottom: none;
}
.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
th.num {
  text-align: right;
}
.short {
  color: var(--color-primary);
  font-weight: var(--weight-semibold);
  text-decoration: none;
  display: inline-block;
  max-width: 34ch;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
}
.short:hover {
  text-decoration: underline;
}
.target {
  display: block;
  color: var(--color-muted);
  font-size: var(--text-xs);
  max-width: 44ch;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}
.cell-actions {
  white-space: nowrap;
}
.cell-actions > * + * {
  margin-left: var(--space-1);
}
</style>
