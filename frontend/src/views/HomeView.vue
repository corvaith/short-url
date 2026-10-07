<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { urlsApi } from '../services/urls'
import { describeError } from '../services/api'
import type { UrlItem } from '../types'
import { useAuth } from '../composables/useAuth'
import { useBreakpoint } from '../composables/useBreakpoint'
import { usePageMeta } from '../composables/usePageMeta'
import UrlCreateForm from '../components/UrlCreateForm.vue'
import UrlList from '../components/UrlList.vue'
import ErrorState from '../components/ErrorState.vue'

const { isLoggedIn } = useAuth()
const { isMobile } = useBreakpoint()
usePageMeta('Home', { public: true })

const recent = ref<UrlItem[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const listLimit = computed(() => (isMobile.value ? 5 : 5))

async function loadRecent(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const res = await urlsApi.list({ limit: listLimit.value })
    recent.value = res.items
  } catch (e) {
    error.value = describeError(e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (isLoggedIn.value) void loadRecent()
})
</script>

<template>
  <div class="home">
    <section class="hero">
      <h1 class="hero-title" tabindex="-1">Shorten long URLs. Track every click.</h1>
      <p class="hero-sub">Create short links in seconds and manage them all in one place.</p>
      <div class="hero-form">
        <UrlCreateForm />
      </div>
    </section>

    <section v-if="isLoggedIn" class="recent" aria-labelledby="recent-heading">
      <div class="recent-head">
        <h2 id="recent-heading" class="section-title">Recent links</h2>
        <RouterLink to="/dashboard" class="view-all">View all</RouterLink>
      </div>
      <ErrorState v-if="error" :message="error" @retry="loadRecent" />
      <div v-else-if="loading" class="recent-skeletons" aria-busy="true">
        <div v-for="n in 3" :key="n" class="skeleton card skel-card" />
      </div>
      <UrlList v-else-if="recent.length" :urls="recent" />
    </section>
  </div>
</template>

<style scoped>
.hero {
  text-align: center;
  padding: var(--space-8) 0 var(--space-6);
}
.hero-title {
  font-size: var(--text-3xl);
  font-weight: var(--weight-semibold);
  letter-spacing: -0.01em;
}
.hero-title:focus {
  outline: none;
}
.hero-sub {
  color: var(--color-muted);
  margin-top: var(--space-2);
}
.hero-form {
  max-width: 44rem;
  margin: var(--space-6) auto 0;
  text-align: left;
}
.section-title {
  font-size: var(--text-xl);
  font-weight: var(--weight-semibold);
}
.recent {
  margin-top: var(--space-8);
}
.recent-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}
.view-all {
  font-weight: var(--weight-medium);
  font-size: var(--text-sm);
}
.recent-skeletons {
  display: grid;
  gap: var(--space-4);
}
.skel-card {
  height: 120px;
}
</style>
