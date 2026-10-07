<script setup lang="ts">
import BrandMark from '../components/BrandMark.vue'
</script>

<template>
  <div class="auth-layout">
    <header class="auth-top">
      <RouterLink to="/" class="auth-brand" aria-label="ShortINK home">
        <BrandMark />
      </RouterLink>
    </header>
    <main id="main" class="auth-main">
      <div class="auth-card">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<style scoped>
.auth-layout {
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--color-background);
}
.auth-top {
  padding: var(--space-4) var(--page-pad);
}
.auth-brand {
  text-decoration: none;
  color: inherit;
  display: inline-flex;
}
.auth-main {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-8) var(--page-pad) var(--space-12);
}
.auth-card {
  width: 100%;
  max-width: 26rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-8) var(--space-6);
  box-shadow: var(--shadow-md);
}
</style>
