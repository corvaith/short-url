<template>
  <span class="brand">
    <svg class="brand-icon" width="24" height="24" viewBox="0 0 64 64" aria-hidden="true">
      <rect width="64" height="64" rx="14" fill="var(--color-primary)" />
      <path d="M27 37l10-10M25 31l-5 5a7 7 0 1 0 10 10l5-5M39 33l5-5a7 7 0 1 0-10-10l-5 5" stroke="#fff" stroke-width="4" fill="none" stroke-linecap="round" />
    </svg>
    <span class="brand-name">Short URL</span>
  </span>
</template>

<style scoped>
.brand {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-weight: var(--weight-semibold);
  font-size: var(--text-lg);
  color: var(--color-text);
}
.brand-icon {
  border-radius: var(--radius-sm);
}
</style>
