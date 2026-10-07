<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { urlsApi } from '../services/urls'
import { describeError } from '../services/api'
import type { UrlItem } from '../types'
import { useAuth } from '../composables/useAuth'
import { usePageMeta } from '../composables/usePageMeta'
import UrlCreateForm from '../components/UrlCreateForm.vue'
import UrlList from '../components/UrlList.vue'
import ErrorState from '../components/ErrorState.vue'

const { isLoggedIn } = useAuth()
usePageMeta('Home', { public: true })

const recent = ref<UrlItem[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const listLimit = computed(() => 5)

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
    <section class="hero" :class="{ 'hero--full': !isLoggedIn }" aria-labelledby="hero-title">
      <div class="hero-copy">
        <p class="hero-eyebrow">ShortINK — URL shortener</p>
        <h1 id="hero-title" class="hero-title" tabindex="-1">Shorten long URLs. Track every click.</h1>
        <p class="hero-sub">
          Create short links in seconds and manage them all in one place — click counts,
          expiry dates, and one-click copy.
        </p>
        <ul class="hero-points">
          <li>No account needed to shorten</li>
          <li>Custom aliases and expiry for members</li>
          <li>Per-link click analytics</li>
        </ul>
      </div>
      <div class="hero-panel card">
        <h2 class="hero-panel-title">Create a short link</h2>
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
.home {
  width: 100%;
}
.hero {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-6);
  padding: var(--space-6) 0 var(--space-4);
  align-items: start;
}
.hero--full {
  padding-bottom: var(--space-8);
  align-content: center;
  min-height: min(78dvh, 720px);
}
.hero-copy {
  text-align: center;
}
.hero-eyebrow {
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  color: var(--color-primary);
  margin-bottom: var(--space-2);
}
.hero-title {
  font-size: var(--text-3xl);
  font-weight: var(--weight-semibold);
  letter-spacing: -0.02em;
}
.hero-title:focus {
  outline: none;
}
.hero-sub {
  color: var(--color-muted);
  margin-top: var(--space-3);
  max-width: 34rem;
  margin-inline: auto;
}
.hero-points {
  list-style: none;
  padding: 0;
  margin: var(--space-5) 0 0;
  display: inline-grid;
  gap: var(--space-2);
  justify-content: center;
  text-align: left;
  font-size: var(--text-sm);
  color: var(--color-muted);
}
.hero-points li {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  justify-content: flex-start;
}
.hero-points li::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  flex: none;
}
.hero-panel {
  padding: var(--space-5);
}
.hero-panel-title {
  font-size: var(--text-lg);
  font-weight: var(--weight-semibold);
  margin-bottom: var(--space-4);
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
.section-title {
  font-size: var(--text-xl);
  font-weight: var(--weight-semibold);
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

/* Desktop: side-by-side hero, wider reading rhythm. */
@media (min-width: 900px) {
  .hero {
    grid-template-columns: minmax(0, 1fr) minmax(360px, 460px);
    gap: var(--space-10);
    padding: var(--space-10) 0 var(--space-6);
    align-items: center;
  }
  .hero-copy {
    text-align: left;
  }
  .hero-sub {
    margin-inline: 0;
  }
  .hero-points {
    justify-content: start;
  }
  .hero-points li {
    justify-content: start;
  }
  .hero-panel {
    padding: var(--space-6);
  }
}
</style>
