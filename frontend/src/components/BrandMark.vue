<template>
  <span class="brand">
    <svg class="brand-icon" width="26" height="26" viewBox="0 0 64 64" aria-hidden="true">
      <rect width="64" height="64" rx="3" fill="var(--color-brand)" />
      <path
        d="M27 37l10-10M25 31l-5 5a7 7 0 1 0 10 10l5-5M39 33l5-5a7 7 0 1 0-10-10l-5 5"
        stroke="#0B0B0C"
        stroke-width="4"
        fill="none"
        stroke-linecap="round"
      />
    </svg>
    <span class="brand-name">Short<span class="brand-ink">INK</span></span>
  </span>
</template>

<style scoped>
.brand {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-weight: var(--weight-semibold);
  font-size: var(--text-lg);
  letter-spacing: -0.01em;
  color: inherit;
}
.brand-ink {
  color: var(--color-brand);
}
.brand-icon {
  border-radius: 2px;
  flex: none;
}
</style>
