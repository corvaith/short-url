<script setup lang="ts">
import { usePageMeta } from '../composables/usePageMeta'

usePageMeta('Page not found')
</script>

<template>
  <div class="notfound">
    <h1 class="nf-title" tabindex="-1">Page not found</h1>
    <p class="nf-text">The page you're looking for doesn't exist or has moved.</p>
    <RouterLink to="/" class="btn btn-primary">Go home</RouterLink>
  </div>
</template>

<style scoped>
.notfound {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  text-align: center;
  padding: var(--space-12) 0;
}
.nf-title {
  font-size: var(--text-3xl);
  font-weight: var(--weight-semibold);
}
.nf-title:focus {
  outline: none;
}
.nf-text {
  color: var(--color-muted);
}
</style>
