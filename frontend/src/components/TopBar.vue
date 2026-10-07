<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth'
import { useBreakpoint } from '../composables/useBreakpoint'
import BrandMark from './BrandMark.vue'

const router = useRouter()
const { isLoggedIn, logout } = useAuth()
const { isMobile } = useBreakpoint()

const showInlineNav = computed(() => isLoggedIn.value && !isMobile.value)

async function onLogout(): Promise<void> {
  await logout()
  await router.push('/login')
}
</script>

<template>
  <header class="topbar">
    <div class="container topbar-inner">
      <RouterLink to="/" class="topbar-brand">
        <BrandMark />
      </RouterLink>
      <nav v-if="showInlineNav" class="topbar-nav" aria-label="Main">
        <RouterLink to="/dashboard" class="topbar-link">Links</RouterLink>
        <RouterLink to="/settings" class="topbar-link">Settings</RouterLink>
        <button type="button" class="btn btn-ghost btn-sm" @click="onLogout">Log out</button>
      </nav>
      <nav v-else class="topbar-nav" aria-label="Main">
        <template v-if="!isLoggedIn">
          <RouterLink to="/login" class="topbar-link">Log in</RouterLink>
          <RouterLink to="/register" class="btn btn-primary btn-sm">Sign up</RouterLink>
        </template>
      </nav>
    </div>
  </header>
</template>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 100;
  height: var(--topbar-h);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}
.topbar-inner {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.topbar-brand {
  text-decoration: none;
  color: inherit;
  display: inline-flex;
}
.topbar-nav {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}
.topbar-link {
  color: var(--color-muted);
  text-decoration: none;
  font-weight: var(--weight-medium);
  font-size: var(--text-sm);
  min-height: var(--tap-min);
  display: inline-flex;
  align-items: center;
}
.topbar-link:hover,
.topbar-link.router-link-active {
  color: var(--color-text);
}
</style>
