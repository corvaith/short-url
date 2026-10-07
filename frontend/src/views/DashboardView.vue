<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { urlsApi } from '../services/urls'
import { describeError } from '../services/api'
import type { Overview, UrlItem } from '../types'
import { useToast } from '../composables/useToast'
import { usePageMeta } from '../composables/usePageMeta'
import PageHeader from '../components/PageHeader.vue'
import StatCard from '../components/StatCard.vue'
import UrlList from '../components/UrlList.vue'
import BaseButton from '../components/BaseButton.vue'
import EmptyState from '../components/EmptyState.vue'
import ErrorState from '../components/ErrorState.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'

usePageMeta('Dashboard')
const toast = useToast()

const overview = ref<Overview | null>(null)
const items = ref<UrlItem[]>([])
const nextCursor = ref<string | null>(null)
const loading = ref(true)
const loadingMore = ref(false)
const error = ref<string | null>(null)
const pendingDelete = ref<UrlItem | null>(null)
const dialogOpen = ref(false)

async function load(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const [ov, list] = await Promise.all([
      urlsApi.overview(),
      urlsApi.list({ limit: 10 }),
    ])
    overview.value = ov
    items.value = list.items
    nextCursor.value = list.next_cursor
  } catch (e) {
    error.value = describeError(e)
  } finally {
    loading.value = false
  }
}

async function loadMore(): Promise<void> {
  if (!nextCursor.value) return
  loadingMore.value = true
  try {
    const res = await urlsApi.list({ limit: 10, cursor: nextCursor.value })
    items.value = [...items.value, ...res.items]
    nextCursor.value = res.next_cursor
  } catch (e) {
    toast.error(describeError(e))
  } finally {
    loadingMore.value = false
  }
}

async function onAction(id: string, key: string): Promise<void> {
  const item = items.value.find((u) => u.id === id)
  if (!item) return
  if (key === 'details') {
    useRouterPush(`/urls/${id}`)
  } else if (key === 'toggle') {
    try {
      const updated = await urlsApi.setActive(id, !item.is_active)
      items.value = items.value.map((u) => (u.id === id ? updated : u))
      toast.success(updated.is_active ? 'Link enabled' : 'Link disabled')
    } catch (e) {
      toast.error(describeError(e))
    }
  } else if (key === 'delete') {
    pendingDelete.value = item
    dialogOpen.value = true
  }
}

import { useRouter } from 'vue-router'
const router = useRouter()
function useRouterPush(path: string): void {
  void router.push(path)
}

async function confirmDelete(): Promise<void> {
  const item = pendingDelete.value
  if (!item) return
  try {
    await urlsApi.remove(item.id)
    items.value = items.value.filter((u) => u.id !== item.id)
    toast.success('Link deleted')
    void load()
  } catch (e) {
    toast.error(describeError(e))
  } finally {
    pendingDelete.value = null
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Dashboard" subtitle="All your short links in one place.">
      <template #actions>
        <RouterLink to="/" class="btn btn-primary" style="display:inline-flex;align-items:center;">New link</RouterLink>
      </template>
    </PageHeader>

    <div v-if="loading" class="stats-grid" aria-busy="true">
      <div v-for="n in 3" :key="n" class="skeleton card skel-stat" />
    </div>
    <div v-else class="stats-grid">
      <StatCard label="Total URLs" :value="overview?.total_urls ?? 0" />
      <StatCard label="Total clicks" :value="overview?.total_clicks ?? 0" />
      <StatCard label="Active URLs" :value="overview?.active_urls ?? 0" />
    </div>

    <section class="list-section" aria-labelledby="all-links-heading">
      <h2 id="all-links-heading" class="sr-only">All links</h2>
      <ErrorState v-if="error" :message="error" @retry="load" />
      <template v-else-if="loading">
        <div class="list-skeletons" aria-busy="true">
          <div v-for="n in 3" :key="n" class="skeleton card skel-row" />
        </div>
      </template>
      <EmptyState
        v-else-if="!items.length"
        title="No shortened URLs yet."
        description="Create your first short URL above."
      >
        <RouterLink to="/" class="btn btn-primary">Create a short URL</RouterLink>
      </EmptyState>
      <template v-else>
        <UrlList :urls="items" @action="onAction" />
        <div v-if="nextCursor" class="load-more">
          <BaseButton variant="secondary" :loading="loadingMore" @click="loadMore">
            {{ loadingMore ? 'Loading…' : 'Load more' }}
          </BaseButton>
        </div>
      </template>
    </section>

    <ConfirmDialog
      :open="dialogOpen"
      title="Delete this link?"
      message="This can't be undone."
      confirm-label="Delete"
      danger
      @cancel="dialogOpen = false; pendingDelete = null"
      @confirm="dialogOpen = false; confirmDelete()"
    />
  </div>
</template>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-4);
}
@media (min-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}
.skel-stat {
  height: 96px;
}
.list-section {
  margin-top: var(--space-8);
}
.list-skeletons {
  display: grid;
  gap: var(--space-4);
}
.skel-row {
  height: 120px;
}
.load-more {
  display: flex;
  justify-content: center;
  margin-top: var(--space-6);
}
</style>
