<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useBreakpoint } from '../composables/useBreakpoint'

export interface MenuItem {
  key: string
  label: string
  danger?: boolean
}

defineProps<{ items: MenuItem[] }>()
const emit = defineEmits<{ select: [key: string] }>()

const { isMobile } = useBreakpoint()
const open = ref(false)
const rootRef = ref<HTMLElement | null>(null)
const menuRef = ref<HTMLElement | null>(null)
const menuId = `menu-${Math.random().toString(36).slice(2, 9)}`

function onDocClick(e: MouseEvent): void {
  if (open.value && rootRef.value && !rootRef.value.contains(e.target as Node)) open.value = false
}

function onKeydown(e: KeyboardEvent): void {
  if (!open.value) return
  if (e.key === 'Escape') {
    open.value = false
    rootRef.value?.querySelector('button')?.focus()
  } else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    const buttons = Array.from(menuRef.value?.querySelectorAll('button') ?? [])
    if (!buttons.length) return
    const idx = buttons.indexOf(document.activeElement as HTMLButtonElement)
    const next = e.key === 'ArrowDown' ? (idx + 1) % buttons.length : (idx - 1 + buttons.length) % buttons.length
    buttons[next]?.focus()
  }
}

onMounted(() => document.addEventListener('click', onDocClick))
onBeforeUnmount(() => document.removeEventListener('click', onDocClick))
watch(open, (v) => {
  if (v) requestAnimationFrame(() => menuRef.value?.querySelector<HTMLElement>('button')?.focus())
})

function choose(key: string): void {
  open.value = false
  emit('select', key)
}
</script>

<template>
  <div ref="rootRef" class="action-menu" @keydown="onKeydown">
    <button
      type="button"
      class="menu-trigger"
      aria-label="More actions"
      aria-haspopup="menu"
      :aria-expanded="open"
      :aria-controls="open ? menuId : undefined"
      @click.stop="open = !open"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <circle cx="5" cy="12" r="1.8" /><circle cx="12" cy="12" r="1.8" /><circle cx="19" cy="12" r="1.8" />
      </svg>
    </button>

    <Teleport to="body">
      <div v-if="open && isMobile" class="sheet-backdrop" @click="open = false">
        <div :id="menuId" ref="menuRef" class="sheet" role="menu" @click.stop>
          <div class="sheet-handle" aria-hidden="true" />
          <button v-for="item in items" :key="item.key" type="button" role="menuitem" class="sheet-item" :class="{ danger: item.danger }" @click="choose(item.key)">
            {{ item.label }}
          </button>
        </div>
      </div>
      <div v-else-if="open" :id="menuId" ref="menuRef" class="popover" role="menu">
        <button v-for="item in items" :key="item.key" type="button" role="menuitem" class="popover-item" :class="{ danger: item.danger }" @click="choose(item.key)">
          {{ item.label }}
        </button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.action-menu {
  position: relative;
  display: inline-flex;
}
.menu-trigger {
  min-width: var(--tap-min);
  min-height: var(--tap-min);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--color-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.menu-trigger:hover {
  background: var(--color-surface-muted);
  color: var(--color-text);
}
.popover {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  z-index: 120;
  min-width: 11rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  padding: var(--space-1);
}
.popover-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 0 var(--space-3);
  min-height: 40px;
  border: none;
  background: none;
  font: inherit;
  font-size: var(--text-sm);
  color: var(--color-text);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.popover-item:hover,
.popover-item:focus-visible {
  background: var(--color-surface-muted);
}
.popover-item.danger {
  color: var(--color-danger);
}
.sheet-backdrop {
  position: fixed;
  inset: 0;
  z-index: 130;
  background: rgb(18 22 28 / 0.4);
}
.sheet {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--color-surface);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  padding: var(--space-2) var(--space-2) calc(var(--space-4) + env(safe-area-inset-bottom));
  padding-bottom: calc(var(--space-4) + env(safe-area-inset-bottom));
}
.sheet-handle {
  width: 40px;
  height: 4px;
  border-radius: var(--radius-full);
  background: var(--color-border-strong);
  margin: 0 auto var(--space-2);
}
.sheet-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 0 var(--space-4);
  min-height: 48px;
  border: none;
  background: none;
  font: inherit;
  font-size: var(--text-base);
  color: var(--color-text);
  border-radius: var(--radius-md);
  cursor: pointer;
}
.sheet-item.danger {
  color: var(--color-danger);
}
.sheet-item:hover,
.sheet-item:focus-visible {
  background: var(--color-surface-muted);
}
</style>
