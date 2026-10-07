<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { urlsApi } from '../services/urls'
import { describeError } from '../services/api'
import type { UrlItem, UrlStats } from '../types'
import { useToast } from '../composables/useToast'
import { usePageMeta } from '../composables/usePageMeta'
import StatusBadge from '../components/StatusBadge.vue'
import CopyButton from '../components/CopyButton.vue'
import SegmentedControl from '../components/SegmentedControl.vue'
import ClicksChart from '../components/ClicksChart.vue'
import BaseButton from '../components/BaseButton.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'
import ErrorState from '../components/ErrorState.vue'

const route = useRoute()
const toast = useToast()
usePageMeta('Link details')

const item = ref<UrlItem | null>(null)
const stats = ref<UrlStats | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const notFound = ref(false)
const days = ref<'7' | '30'>('7')
const showTable = ref(false)
const confirmOpen = ref(false)

const rangeOptions = [
  { value: '7', label: '7 days' },
  { value: '30', label: '30 days' },
]

async function load(): Promise<void> {
  loading.value = true
  error.value = null
  notFound.value = false
  const id = route.params.id as string
  try {
    const [u] = await Promise.all([urlsApi.get(id)])
    item.value = u
    await loadStats()
  } catch (e) {
    if (e instanceof Error && 'status' in e && (e as { status?: number }).status === 404) {
      notFound.value = true
    } else {
      error.value = describeError(e)
    }
  } finally {
    loading.value = false
  }
}

async function loadStats(): Promise<void> {
  if (!item.value) return
  try {
    stats.value = await urlsApi.stats(item.value.id, Number(days.value) as 7 | 30)
  } catch (e) {
    toast.error(describeError(e))
  }
}

watch(days, () => void loadStats())

async function toggleActive(): Promise<void> {
  if (!item.value) return
  try {
    item.value = await urlsApi.setActive(item.value.id, !item.value.is_active)
    toast.success(item.value.is_active ? 'Link enabled' : 'Link disabled')
  } catch (e) {
    toast.error(describeError(e))
  }
}

async function doDelete(): Promise<void> {
  if (!item.value) return
  try {
    await urlsApi.remove(item.value.id)
    toast.success('Link deleted')
  } catch (e) {
    toast.error(describeError(e))
  }
}

function onConfirmDelete(): void {
  confirmOpen.value = false
  void doDelete()
}

function fmt(iso: string | null): string {
  if (!iso) return 'Never'
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'medium' })
}

const summary = computed(
  () => `${stats.value?.range_clicks ?? 0} clicks in last ${days.value} days`,
)

onMounted(load)
</script>

<template>
  <div>
    <RouterLink to="/dashboard" class="back">← Dashboard</RouterLink>

    <ErrorState v-if="error && !loading" :message="error" @retry="load" />
    <div v-else-if="loading" class="skeleton card skel-page" aria-busy="true" />
    <div v-else-if="notFound" class="notfound card">
      <h1 class="page-title" tabindex="-1">Link not found</h1>
      <p class="muted-text">This link doesn't exist or you don't have access to it.</p>
      <RouterLink to="/dashboard" class="btn btn-secondary">Back to dashboard</RouterLink>
    </div>

    <template v-else-if="item">
      <h1 class="page-title" tabindex="-1">Link details</h1>
      <header class="detail-header card">
        <div class="detail-top">
          <p class="short mono">{{ item.short_url }}</p>
          <StatusBadge :status="item.status" />
        </div>
        <div class="detail-actions">
          <CopyButton :text="item.short_url" variant="primary" />
          <a class="btn btn-secondary" :href="item.short_url" target="_blank" rel="noopener"
            >Open</a
          >
        </div>
      </header>

      <div class="detail-grid">
        <section class="card">
          <h2 class="card-title">Details</h2>
          <dl class="dl">
            <div class="dl-row">
              <dt>Short URL</dt>
              <dd class="mono">{{ item.short_url }}</dd>
            </div>
            <div class="dl-row">
              <dt>Original URL</dt>
              <dd class="wrap">{{ item.target_url }}</dd>
            </div>
            <div class="dl-row">
              <dt>Created</dt>
              <dd>{{ fmt(item.created_at) }}</dd>
            </div>
            <div class="dl-row">
              <dt>Last accessed</dt>
              <dd>{{ fmt(item.last_accessed_at) }}</dd>
            </div>
            <div class="dl-row">
              <dt>Clicks</dt>
              <dd>{{ item.clicks }}</dd>
            </div>
            <div class="dl-row">
              <dt>Expiration</dt>
              <dd>{{ item.expires_at ? fmt(item.expires_at) : 'No expiration' }}</dd>
            </div>
            <div class="dl-row">
              <dt>Status</dt>
              <dd><StatusBadge :status="item.status" /></dd>
            </div>
          </dl>
        </section>

        <section class="card chart-card">
          <div class="chart-head">
            <h2 class="card-title">Clicks</h2>
            <SegmentedControl v-model="days" label="Date range" :options="rangeOptions" />
          </div>
          <p class="summary">{{ summary }}</p>
          <template v-if="stats">
            <ClicksChart :series="stats.series" />
            <button
              type="button"
              class="btn btn-ghost table-toggle"
              :aria-expanded="showTable"
              @click="showTable = !showTable"
            >
              {{ showTable ? 'Hide data table' : 'View data table' }}
            </button>
            <div v-if="showTable" class="table-wrap">
              <table class="data-table">
                <caption class="sr-only">
                  Clicks per day, last
                  {{
                    days
                  }}
                  days
                </caption>
                <thead>
                  <tr>
                    <th scope="col">Date</th>
                    <th scope="col" class="num">Clicks</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="p in stats.series" :key="p.date">
                    <td>{{ p.date }}</td>
                    <td class="num">{{ p.clicks }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </template>
        </section>

        <section class="card manage-card">
          <h2 class="card-title">Manage</h2>
          <div class="manage-actions">
            <BaseButton variant="secondary" block @click="toggleActive">
              {{ item.is_active ? 'Disable' : 'Enable' }}
            </BaseButton>
            <BaseButton variant="danger" block @click="confirmOpen = true">Delete</BaseButton>
          </div>
        </section>
      </div>
    </template>

    <ConfirmDialog
      :open="confirmOpen"
      title="Delete this link?"
      message="This can't be undone."
      confirm-label="Delete"
      danger
      @cancel="confirmOpen = false"
      @confirm="onConfirmDelete"
    />
  </div>
</template>

<style scoped>
.back {
  display: inline-flex;
  align-items: center;
  min-height: var(--tap-min);
  color: var(--color-muted);
  text-decoration: none;
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
}
.back:hover {
  color: var(--color-text);
}
.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--weight-semibold);
  margin: var(--space-4) 0 var(--space-4);
}
.page-title:focus {
  outline: none;
}
.detail-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.detail-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.short {
  font-size: var(--text-lg);
  font-weight: var(--weight-semibold);
  overflow-wrap: anywhere;
}
.detail-actions {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.detail-grid {
  display: grid;
  gap: var(--space-4);
  margin-top: var(--space-4);
}
@media (min-width: 1024px) {
  .detail-grid {
    grid-template-columns: 1fr 1fr;
  }
}
.card-title {
  font-size: var(--text-lg);
  font-weight: var(--weight-semibold);
}
.dl {
  margin: var(--space-4) 0 0;
}
.dl-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--color-border);
}
.dl-row:last-child {
  border-bottom: none;
}
dt {
  font-size: var(--text-xs);
  color: var(--color-muted);
  font-weight: var(--weight-medium);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
dd {
  margin: 0;
  font-size: var(--text-sm);
  overflow-wrap: anywhere;
}
.chart-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.summary {
  color: var(--color-muted);
  font-size: var(--text-sm);
  margin-top: var(--space-2);
}
.table-toggle {
  margin-top: var(--space-3);
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
  margin-top: var(--space-3);
}
.data-table th,
.data-table td {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}
.data-table .num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.manage-card {
  grid-column: 1 / -1;
}
.manage-actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: var(--space-4);
  max-width: 24rem;
}
@media (min-width: 768px) {
  .manage-actions {
    flex-direction: row;
    max-width: none;
  }
}
.skel-page {
  height: 320px;
  margin-top: var(--space-4);
}
.notfound {
  margin-top: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  align-items: flex-start;
}
.muted-text {
  color: var(--color-muted);
}
</style>
